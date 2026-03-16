#!/usr/bin/env python3
"""
Claude 1 automated trader + colonizer.
Sells resources every 60s, builds infrastructure, expands to new planets.
Run: python3 scripts/claude1_trader.py &
"""

import json
import time
import urllib.request
import urllib.error

API = "https://api.xandaris.space"
ADMIN_KEY = "e4287f2503866497ce414a00c55f4581"
PLAYER = "claude 1"
INTERVAL = 60

RESOURCES = ["Oil", "Helium-3", "Rare Metals", "Iron", "Water", "Fuel", "Electronics"]


def api_post(endpoint, data, key):
    req = urllib.request.Request(
        f"{API}{endpoint}",
        data=json.dumps(data).encode(),
        headers={"Content-Type": "application/json", "X-API-Key": key},
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            return json.loads(resp.read())
    except Exception:
        return {"ok": False}


def api_get(endpoint, key):
    req = urllib.request.Request(f"{API}{endpoint}", headers={"X-API-Key": key})
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            return json.loads(resp.read())
    except Exception:
        return {"ok": False}


def register():
    resp = api_post("/api/admin/register", {"name": PLAYER}, ADMIN_KEY)
    if resp.get("ok"):
        return resp["data"]["api_key"]
    return None


def get_status(key):
    resp = api_get("/api/status", key)
    if resp.get("ok"):
        return resp["data"]
    return None


def sell_all(key, planet_id):
    earned = 0
    for res in RESOURCES:
        for sz in [250, 250, 250, 250, 100, 50]:
            resp = api_post("/api/market/trade",
                {"resource": res, "quantity": sz, "action": "sell", "planet_id": planet_id}, key)
            if resp.get("ok"):
                earned += resp["data"]["total"]
            else:
                break
    return earned


def get_my_ships(key):
    resp = api_get("/api/ships", key)
    if not resp.get("ok"):
        return []
    return [s for s in resp["data"] if s["owner"] == PLAYER]


def get_system(key, sys_id):
    resp = api_get(f"/api/systems/{sys_id}", key)
    if resp.get("ok"):
        return resp["data"]
    return None


def get_galaxy(key):
    resp = api_get("/api/galaxy", key)
    if resp.get("ok"):
        return resp["data"]
    return []


def build(key, planet_id, building_type, resource_id=0):
    data = {"planet_id": planet_id, "building_type": building_type}
    if resource_id:
        data["resource_id"] = resource_id
    return api_post("/api/build", data, key)


def build_ship(key, planet_id, ship_type):
    return api_post("/api/ships/build", {"planet_id": planet_id, "ship_type": ship_type}, key)


def move_ship(key, ship_id, target_system):
    return api_post("/api/ships/move", {"ship_id": ship_id, "target_system_id": target_system}, key)


def colonize(key, ship_id, planet_id):
    return api_post("/api/colonize", {"ship_id": ship_id, "planet_id": planet_id}, key)


def upgrade(key, planet_id, building_index):
    return api_post("/api/upgrade", {"planet_id": planet_id, "building_index": building_index}, key)


def buy_resource(key, planet_id, resource, qty):
    return api_post("/api/market/trade",
        {"resource": resource, "quantity": qty, "action": "buy", "planet_id": planet_id}, key)


def get_rank(key):
    resp = api_get("/api/leaderboard", key)
    if not resp.get("ok"):
        return None, None
    for e in resp["data"]:
        if e and e["name"] == PLAYER:
            return e["rank"], e["credits"]
    return None, None


def ensure_infrastructure(key, status):
    """Build missing infrastructure on all owned planets."""
    for pl in status["player"]["planets"]:
        pid = pl["id"]
        buildings = set()
        planet_resp = api_get(f"/api/planets/{pid}", key)
        if not planet_resp.get("ok"):
            continue
        pd = planet_resp["data"]
        for b in pd["buildings"]:
            buildings.add(b["type"])

        # Must have Trading Post
        if "Trading Post" not in buildings:
            r = build(key, pid, "Trading Post")
            if r.get("ok"):
                print(f"  [Build] Trading Post on {pl['name']}")

        # Must have Generator
        if "Generator" not in buildings:
            r = build(key, pid, "Generator")
            if r.get("ok"):
                print(f"  [Build] Generator on {pl['name']}")

        # Build mines on all unmined deposits
        for dep in pd.get("resource_deposits", []):
            if not dep["has_mine"]:
                r = build(key, pid, "Mine", dep["id"])
                if r.get("ok"):
                    print(f"  [Build] Mine on {dep['resource_type']} at {pl['name']}")

        # Build Shipyard if missing (needed for colony ships)
        if "Shipyard" not in buildings:
            r = build(key, pid, "Shipyard")
            if r.get("ok"):
                print(f"  [Build] Shipyard on {pl['name']}")


def try_colonize(key, status):
    """Find a colony ship and send it to colonize an unclaimed planet."""
    ships = get_my_ships(key)
    colony_ships = [s for s in ships if s["type"] == "Colony" and s["status"] != "Moving"]

    if not colony_ships:
        # Try to build one if we have a shipyard and enough resources
        for pl in status["player"]["planets"]:
            credits = status["player"]["credits"]
            storage = pl["storage"]
            if credits >= 2000 and storage.get("Fuel", 0) >= 80 and storage.get("Iron", 0) >= 100:
                r = build_ship(key, pl["id"], "Colony")
                if r.get("ok"):
                    print(f"  [Ship] Colony ship building at {pl['name']}")
                return
        # Try buying resources for colony ship
        for pl in status["player"]["planets"]:
            credits = status["player"]["credits"]
            if credits >= 50000:
                storage = pl["storage"]
                if storage.get("Fuel", 0) < 80:
                    buy_resource(key, pl["id"], "Fuel", 100)
                if storage.get("Iron", 0) < 100:
                    buy_resource(key, pl["id"], "Iron", 100)
                if storage.get("Rare Metals", 0) < 20:
                    buy_resource(key, pl["id"], "Rare Metals", 25)
        return

    # Find unclaimed habitable planets in nearby systems
    galaxy = get_galaxy(key)
    my_systems = {pl["system_id"] for pl in status["player"]["planets"]}

    for ship in colony_ships:
        # Check current system for unclaimed planets
        sys_data = get_system(key, ship["system_id"])
        if sys_data:
            for p in sys_data.get("planets", []):
                if not p.get("owner") and p.get("habitability", 0) > 20:
                    r = colonize(key, ship["id"], p["id"])
                    if r.get("ok"):
                        print(f"  [Colonize] {p['name']} (hab={p['habitability']})")
                        return

        # Look at connected systems
        for sys in galaxy:
            if sys["id"] == ship["system_id"]:
                for link in sys.get("links", []):
                    if link in my_systems:
                        continue
                    linked_sys = get_system(key, link)
                    if not linked_sys:
                        continue
                    for p in linked_sys.get("planets", []):
                        if not p.get("owner") and p.get("habitability", 0) > 20:
                            # Move ship there first
                            if ship["system_id"] != link:
                                r = move_ship(key, ship["id"], link)
                                if r.get("ok"):
                                    print(f"  [Move] Colony ship -> SYS-{link+1}")
                                return
                            else:
                                r = colonize(key, ship["id"], p["id"])
                                if r.get("ok"):
                                    print(f"  [Colonize] {p['name']} in SYS-{link+1}")
                                    return


def try_build_cargo_ship(key, status):
    """Build a cargo ship if we don't have one."""
    ships = get_my_ships(key)
    cargo_count = sum(1 for s in ships if s["type"] == "Cargo")
    if cargo_count >= 2:
        return

    for pl in status["player"]["planets"]:
        storage = pl["storage"]
        credits = status["player"]["credits"]
        if credits >= 1000 and storage.get("Fuel", 0) >= 15 and storage.get("Iron", 0) >= 60:
            r = build_ship(key, pl["id"], "Cargo")
            if r.get("ok"):
                print(f"  [Ship] Cargo ship building at {pl['name']}")
                return


def main():
    print(f"[Claude 1 Trader+Colonizer] Starting — cycle every {INTERVAL}s")
    total_earned = 0
    cycles = 0

    while True:
        key = register()
        if not key:
            time.sleep(30)
            continue

        status = get_status(key)
        if not status:
            time.sleep(30)
            continue

        # Sell on all planets
        earned = 0
        for pl in status["player"]["planets"]:
            earned += sell_all(key, pl["id"])
        total_earned += earned
        cycles += 1

        rank, credits = get_rank(key)
        ts = time.strftime("%H:%M:%S")
        n_planets = len(status["player"]["planets"])

        print(f"[{ts}] Cycle {cycles}: +{earned:,} cr | Total: {total_earned:,} | #{rank} ({credits:,} cr) | {n_planets} planets")

        # Every 5 cycles: build infrastructure + try to expand
        if cycles % 5 == 1:
            ensure_infrastructure(key, status)
            try_colonize(key, status)
            try_build_cargo_ship(key, status)

        time.sleep(INTERVAL)


if __name__ == "__main__":
    main()
