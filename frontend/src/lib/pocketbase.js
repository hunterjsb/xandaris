import PocketBase from "pocketbase";

// Base URL for the PocketBase backend
const BASE_URL = import.meta.env.VITE_POCKETBASE_URL || "http://localhost:8090";

// Create a single PocketBase client instance
export const pb = new PocketBase(BASE_URL);

// Helper function to suppress auto-cancellation errors
function suppressAutoCancelError(error) {
  if (error.message?.includes("autocancelled") || error.status === 0) {
    // These are expected auto-cancellation errors, don't log them
    return;
  }
  throw error; // Re-throw non-autocancellation errors
}

// Auth state management
export class AuthManager {
  constructor() {
    this.callbacks = [];
    this.user = null;

    // Initialize auth state
    this.checkAuthStatus();

    // Listen for auth store changes
    pb.authStore.onChange(() => {
      this.checkAuthStatus();
      this.notifyCallbacks();
    });
  }

  checkAuthStatus() {
    this.user = pb.authStore.isValid ? pb.authStore.model : null;
  }

  subscribe(callback) {
    this.callbacks.push(callback);
    callback(this.user); // Call immediately with current state
  }

  unsubscribe(callback) {
    this.callbacks = this.callbacks.filter((cb) => cb !== callback);
  }

  notifyCallbacks() {
    this.callbacks.forEach((callback) => callback(this.user));
  }

  async loginWithDiscord() {
    try {
      const authData = await pb.collection("users").authWithOAuth2({
        provider: "discord",
      });
      return authData;
    } catch (error) {
      console.error("Discord login failed:", error);
      throw error;
    }
  }

  logout() {
    pb.authStore.clear();
  }

  isLoggedIn() {
    return pb.authStore.isValid;
  }

  getUser() {
    return this.user;
  }
}

// Create singleton auth manager
export const authManager = new AuthManager();

// Game data managers
export class GameDataManager {
  constructor() {
    this.ws = null;
    this.callbacks = {
      systems: [],
      fleets: [],
      trades: [],
      tick: [],
      fleet_orders: [], // Renamed from orders
    };
  }

  // Subscribe to real-time updates
  subscribe(type, callback) {
    if (this.callbacks[type]) {
      this.callbacks[type].push(callback);
    }
  }

  unsubscribe(type, callback) {
    if (this.callbacks[type]) {
      this.callbacks[type] = this.callbacks[type].filter(
        (cb) => cb !== callback,
      );
    }
  }

  async getHyperlanes() {
    if (!pb.authStore.isValid) throw new Error("Not authenticated");

    try {
      return await pb.send("/api/hyperlanes", {
        method: "GET",
      });
    } catch (error) {
      console.error("Failed to fetch hyperlanes:", error);
      throw error;
    }
  }

  notifyCallbacks(type, data) {
    if (this.callbacks[type]) {
      this.callbacks[type].forEach((callback) => callback(data));
    }
  }

  // Connect to WebSocket for real-time updates
  connectWebSocket() {
    try {
      this.ws = new WebSocket(`${BASE_URL.replace("http", "ws")}/api/stream`);

      this.ws.onopen = () => {
        console.log("WebSocket connected");
        // Update connection status in UI
        this.updateConnectionStatus("connected");

        // Send auth token if available (with small delay to ensure connection is ready)
        if (pb.authStore.isValid) {
          setTimeout(() => {
            if (this.ws && this.ws.readyState === WebSocket.OPEN) {
              this.ws.send(
                JSON.stringify({
                  type: "auth",
                  token: pb.authStore.token,
                }),
              );
            }
          }, 100);
        }
      };

      this.ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          this.handleWebSocketMessage(data);
        } catch (error) {
          console.error("Failed to parse WebSocket message:", error);
        }
      };

      this.ws.onclose = () => {
        console.log("WebSocket disconnected");
        this.updateConnectionStatus("disconnected");
        // Attempt to reconnect after 5 seconds
        setTimeout(() => this.connectWebSocket(), 5000);
      };

      this.ws.onerror = (error) => {
        console.error("WebSocket error:", error);
        this.updateConnectionStatus("error");
      };
    } catch (error) {
      console.error("Failed to connect WebSocket:", error);
      this.updateConnectionStatus("error");
    }
  }

  updateConnectionStatus(status) {
    // Update UI with connection status
    const statusElement = document.getElementById("ws-status");
    if (statusElement) {
      statusElement.textContent =
        status === "connected" ? "🟢" : status === "error" ? "🔴" : "🟡";
      statusElement.title = `WebSocket: ${status}`;
    }
  }

  handleWebSocketMessage(data) {
    switch (data.type) {
      case "tick":
        this.notifyCallbacks("tick", data.payload);
        break;
      case "system_update":
        this.notifyCallbacks("systems", data.payload);
        break;
      case "fleet_update":
        this.notifyCallbacks("fleets", data.payload);
        break;
      case "trade_update":
        this.notifyCallbacks("trades", data.payload);
        break;
      case "fleet_order_update": // Renamed from order_update
        this.notifyCallbacks("fleet_orders", data.payload);
        break;
      default:
        console.log("Unknown WebSocket message type:", data.type);
    }
  }

  async getFleetOrders(userId = null) { // Renamed from getOrders
    if (!pb.authStore.isValid && !userId) {
        // For player view, userId will always be present.
    }
    if (!userId && pb.authStore.isValid) { // If called without userId but user is logged in.
        userId = pb.authStore.model.id;
    }

    if (!userId) { // If still no userId, cannot fetch user-specific orders
        console.warn("getFleetOrders called without userId and no authenticated user.");
        return [];
    }

    try {
        const filterParts = [
            `(status != "completed" && status != "failed" && status != "cancelled")`,
            `user_id = "${userId}"`
        ];
        const filter = filterParts.join(" && ");
        
        return await pb.collection("fleet_orders").getFullList({ // Changed to fleet_orders
            filter: filter,
            sort: "execute_at_tick", // Ascending is default
            requestKey: `getFleetOrders-${userId}-${Date.now()}`, 
        });
    } catch (error) {
        try {
            suppressAutoCancelError(error);
        } catch (e) {
            console.error("Failed to fetch fleet orders:", e);
        }
        return [];
    }
  }

  // API methods
  async getSystems() {
    try {
      return await pb.collection("systems").getFullList({
        sort: "x,y",
      });
    } catch (error) {
      console.error("Failed to fetch systems:", error);
      return [];
    }
  }

  async getPlayer(userId) {
    try {
      const record = await pb.collection("users").getOne(userId, {
        requestKey: `getPlayer-${userId}-${Date.now()}`,
      });
      return record;
    } catch (error) {
      console.error("Failed to fetch player details:", error);
      return null;
    }
  }

  async getPlayerCredits(userId) {
    try {
      const record = await pb.collection("players").getOne(userId);
      return record.credits;
    } catch (error) {
      console.error("Failed to fetch player credits:", error);
      return 0;
    }
  }

  async getUserResources() {
    try {
      const response = await pb.send('/api/user/resources', {
        method: 'GET',
        requestKey: `getUserResources-${Date.now()}`,
      });
      return response.resources;
    } catch (error) {
      // Handle auto-cancellation silently - it's expected behavior
      if (error.status === 0 && error.message?.includes('autocancelled')) {
        console.debug("User resources request was auto-cancelled (expected behavior)");
        // Return null to indicate cancellation, caller should handle gracefully
        return null;
      }
      console.error("Failed to fetch user resources:", error);
      return {
        credits: 0,
        food: 0,
        ore: 0,
        fuel: 0,
        metal: 0,
        oil: 0,
        titanium: 0,
        xanium: 0
      };
    }
  }

  async getSystem(id) {
    try {
      return await pb.collection("systems").getOne(id);
    } catch (error) {
      console.error("Failed to fetch system:", error);
      return null;
    }
  }

  async getFleets(userId = null) {
    try {
      const url = userId ? `/api/fleets?owner_id=${userId}` : "/api/fleets";
      const response = await fetch(`${pb.baseUrl}${url}`, {
        headers: {
          Authorization: pb.authStore.token || "",
        },
      });
      
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }
      
      const data = await response.json();
      return data.items || [];
    } catch (error) {
      try {
        suppressAutoCancelError(error);
      } catch (e) {
        console.error("Failed to fetch fleets:", e);
      }
      return [];
    }
  }

  async getTrades(userId = null) {
    try {
      const filter = userId ? `owner_id = "${userId}"` : "";
      return await pb.collection("trade_routes").getFullList({
        filter,
        sort: "created",
      });
    } catch (error) {
      try {
        suppressAutoCancelError(error);
      } catch (e) {
        console.error("Failed to fetch trades:", e);
      }
      return [];
    }
  }

  async getBuildings(userId = null) {
    try {
      let url = "/api/buildings";
      const params = {};
      if (userId) {
        params.owner_id = userId;
      }

      const response = await pb.send(url, {
        method: "GET",
        params: params, // PocketBase SDK handles adding this to query string
      });
      return response.items || [];
    } catch (error) {
      try {
        suppressAutoCancelError(error);
      } catch (e) {
        console.error("Failed to fetch buildings:", e);
      }
      return [];
    }
  }

  async getTreaties(userId = null) {
    try {
      // Treaties collection doesn't exist in new schema yet
      // Return empty array for now
      return [];
    } catch (error) {
      try {
        suppressAutoCancelError(error);
      } catch (e) {
        console.error("Failed to fetch treaties:", e);
      }
      return [];
    }
  }

  // Action methods
  async sendFleet(fromId, toId, strength, fleetId = null) {
    if (!pb.authStore.isValid) throw new Error("Not authenticated");

    try {
      const payload = {
        from_id: fromId,
        to_id: toId,
        strength: strength,
      };
      
      if (fleetId) {
        payload.fleet_id = fleetId;
      }

      return await pb.send("/api/orders/fleet", {
        method: "POST",
        body: JSON.stringify(payload),
        headers: {
          "Content-Type": "application/json",
        },
      });
    } catch (error) {
      console.error("Failed to send fleet:", error);
      throw error;
    }
  }



  async sendFleetRoute(fleetId, routePath) {
    if (!pb.authStore.isValid) throw new Error("Not authenticated");

    try {
      return await pb.send("/api/orders/fleet-route", {
        method: "POST",
        body: JSON.stringify({
          fleet_id: fleetId,
          route_path: routePath,
        }),
        headers: {
          "Content-Type": "application/json",
        },
      });
    } catch (error) {
      console.error("Failed to send fleet route:", error);
      throw error;
    }
  }

  async queueBuilding(planetId, buildingType, fleetId) { // Added fleetId parameter
    if (!pb.authStore.isValid) throw new Error("Not authenticated");

    try {
      return await pb.send("/api/orders/build", {
        method: "POST",
        body: JSON.stringify({
          planet_id: planetId, // Changed system_id to planet_id
          building_type: buildingType,
          fleet_id: fleetId, // Include fleet_id for resource consumption
        }),
        headers: {
          "Content-Type": "application/json",
        },
      });
    } catch (error) {
      console.error("Failed to queue building:", error);
      throw error;
    }
  }

  async getShipCargo(fleetId) {
    if (!pb.authStore.isValid) throw new Error("Not authenticated");

    try {
      return await pb.send(`/api/ship_cargo?fleet_id=${fleetId}`, {
        method: "GET",
        headers: {
          "Content-Type": "application/json",
        },
      });
    } catch (error) {
      console.error("Failed to get ship cargo:", error);
      throw error;
    }
  }

  async createTradeRoute(fromId, toId, cargo, capacity) {
    if (!pb.authStore.isValid) throw new Error("Not authenticated");

    try {
      return await pb.send("/api/orders/trade", {
        method: "POST",
        body: JSON.stringify({
          from_id: fromId,
          to_id: toId,
          cargo: cargo,
          capacity: capacity,
        }),
        headers: {
          "Content-Type": "application/json",
        },
      });
    } catch (error) {
      console.error("Failed to create trade route:", error);
      throw error;
    }
  }

  async proposeTreaty(playerId, type, terms) {
    if (!pb.authStore.isValid) throw new Error("Not authenticated");

    try {
      return await pb.send("/diplomacy", {
        method: "POST",
        body: JSON.stringify({
          player_id: playerId,
          type: type,
          terms: terms,
        }),
        headers: {
          "Content-Type": "application/json",
        },
      });
    } catch (error) {
      console.error("Failed to propose treaty:", error);
      throw error;
    }
  }

  async getMap() {
    try {
      return await pb.send("/api/map", {
        method: "GET",
      });
    } catch (error) {
      // Suppress auto-cancellation errors (these are normal)
      if (!error.message?.includes("autocancelled")) {
        console.error("Failed to fetch map:", error);
      }
      return null;
    }
  }

  async getStatus() {
    try {
      return await pb.send("/api/status", {
        method: "GET",
      });
    } catch (error) {
      try {
        suppressAutoCancelError(error);
      } catch (e) {
        console.error("Failed to fetch status:", e);
      }
      return null;
    }
  }

  async getBuildingTypes() {
    try {
      const response = await pb.send("/api/building_types", {
        method: "GET",
      });
      return response.items || [];
    } catch (error) {
      try {
        suppressAutoCancelError(error);
      } catch (e) {
        console.error("Failed to fetch building types:", e);
      }
      return [];
    }
  }

  async getResourceTypes() {
    try {
      const response = await pb.send("/api/resource_types", {
        method: "GET",
      });
      return response.items || [];
    } catch (error) {
      try {
        suppressAutoCancelError(error);
      } catch (e) {
        console.error("Failed to fetch resource types:", e);
      }
      return [];
    }
  }

  async getPopulations(userId = null) {
    try {
      let url = "/api/collections/populations/records";
      const params = {};
      if (userId) {
        params.filter = `owner_id='${userId}'`;
      }
      params.expand = 'employed_at,planet_id';

      const response = await pb.send(url, {
        method: "GET",
        params: params,
      });
      return response.items || [];
    } catch (error) {
      try {
        suppressAutoCancelError(error);
      } catch (e) {
        console.error("Failed to fetch populations:", e);
      }
      return [];
    }
  }
}

// Create singleton game data manager
export const gameData = new GameDataManager();

// Initialize WebSocket connection when auth state changes
authManager.subscribe((user) => {
  // Always connect to WebSocket for tick updates
  if (!gameData.ws) {
    gameData.connectWebSocket();
  }
});

// Connect immediately for tick updates
gameData.connectWebSocket();
