// Game state management similar to vibe-chuck's events store
import { gameData, authManager } from "../lib/pocketbase.js";

export class GameState {
  constructor() {
    this.systems = [];
    this.fleets = [];
    this.trades = [];
    this.treaties = [];
    this.buildings = [];
    this.mapData = null;
    this.selectedSystem = null;
    this.currentTick = 1;
    this.ticksPerMinute = 6;
    this.playerResources = {
      credits: 0,
      food: 0,
      ore: 0,
      goods: 0,
      fuel: 0,
    };

    this.callbacks = [];
    this.initialized = false;

    // Subscribe to auth changes
    authManager.subscribe((user) => {
      if (user && !this.initialized) {
        this.initialize();
      } else if (!user) {
        this.reset();
      }
    });

    // Load map data immediately (even without auth)
    this.loadMapData();

    // Subscribe to game data updates
    gameData.subscribe("systems", (data) => this.updateSystems(data));
    gameData.subscribe("fleets", (data) => this.updateFleets(data));
    gameData.subscribe("trades", (data) => this.updateTrades(data));
    gameData.subscribe("tick", (data) => this.handleTick(data));
  }

  async initialize() {
    if (this.initialized) return;

    try {
      await this.loadGameData();
      this.initialized = true;
    } catch (error) {
      console.error("Failed to initialize game state:", error);
    }
  }

  async loadGameData() {
    try {
      // Load game data with individual requests to avoid auto-cancellation
      const userId = authManager.getUser()?.id;

      // Load map data first (most important)
      const mapData = await gameData.getMap();
      if (mapData && mapData.systems) {
        this.systems = mapData.systems;
        this.mapData = mapData;
      }

      // Load user-specific data with delays to prevent auto-cancellation
      if (userId) {
        this.fleets = await gameData.getFleets(userId);
        await new Promise((resolve) => setTimeout(resolve, 50)); // Small delay

        this.trades = await gameData.getTrades(userId);
        await new Promise((resolve) => setTimeout(resolve, 50));

        this.treaties = await gameData.getTreaties(userId);
        await new Promise((resolve) => setTimeout(resolve, 50));

        this.buildings = await gameData.getBuildings(userId);
        await new Promise((resolve) => setTimeout(resolve, 50));

        const player = await gameData.getPlayer(userId);
        await new Promise((resolve) => setTimeout(resolve, 50));
        this.playerResources.credits = player.credits;
      }

      // Load status last
      const status = await gameData.getStatus();
      if (status) {
        this.currentTick = status.current_tick || 1;
        this.ticksPerMinute = status.ticks_per_minute || 6;
      }

      this.updatePlayerResources();
      this.notifyCallbacks();
    } catch (error) {
      console.error("Failed to load game data:", error);
    }
  }

  async refreshGameData() {
    // Refresh data without reinitializing
    if (!this.initialized) {
      await this.initialize();
      return;
    }

    await this.loadGameData();
  }

  reset() {
    this.systems = [];
    this.fleets = [];
    this.trades = [];
    this.treaties = [];
    this.buildings = [];
    this.mapData = null;
    this.selectedSystem = null;
    this.currentTick = 1;
    this.ticksPerMinute = 6;
    this.playerResources = {
      credits: 0,
      food: 0,
      ore: 0,
      goods: 0,
      fuel: 0,
    };
    this.initialized = false;
    this.notifyCallbacks();
  }

  async loadMapData() {
    try {
      console.log("GameState: Loading map data...");
      // Load map data (systems) even without authentication
      const mapData = await gameData.getMap();
      console.log("GameState: Received map data", mapData);
      if (mapData && mapData.systems) {
        console.log(
          "GameState: Setting systems",
          mapData.systems.length,
          "systems",
        );
        this.systems = mapData.systems;
        this.mapData = mapData;
        this.notifyCallbacks();
      }
    } catch (error) {
      console.error("Failed to load map data:", error);
    }
  }

  subscribe(callback) {
    this.callbacks.push(callback);
    callback(this); // Call immediately with current state
  }

  unsubscribe(callback) {
    this.callbacks = this.callbacks.filter((cb) => cb !== callback);
  }

  notifyCallbacks() {
    this.callbacks.forEach((callback) => callback(this));
  }

  updateSystems(systemsData) {
    if (Array.isArray(systemsData)) {
      this.systems = systemsData;
    } else {
      // Single system update
      const index = this.systems.findIndex((s) => s.id === systemsData.id);
      if (index >= 0) {
        this.systems[index] = systemsData;
      } else {
        this.systems.push(systemsData);
      }
    }
    this.updatePlayerResources();
    this.notifyCallbacks();
  }

  updateFleets(fleetsData) {
    if (Array.isArray(fleetsData)) {
      this.fleets = fleetsData;
    } else {
      const index = this.fleets.findIndex((f) => f.id === fleetsData.id);
      if (index >= 0) {
        this.fleets[index] = fleetsData;
      } else {
        this.fleets.push(fleetsData);
      }
    }
    this.notifyCallbacks();
  }

  updateTrades(tradesData) {
    if (Array.isArray(tradesData)) {
      this.trades = tradesData;
    } else {
      const index = this.trades.findIndex((t) => t.id === tradesData.id);
      if (index >= 0) {
        this.trades[index] = tradesData;
      } else {
        this.trades.push(tradesData);
      }
    }
    this.notifyCallbacks();
  }

  handleTick(tickData) {
    this.currentTick = tickData.tick || this.currentTick + 1;
    console.log(
      "Tick update received:",
      tickData,
      "Current tick:",
      this.currentTick,
    );

    // Notify UI of tick update immediately
    this.notifyCallbacks();

    // Optional: Refresh data periodically, but not every tick
    if (this.currentTick % 6 === 0) {
      // Refresh every minute (6 ticks)
      this.refreshGameData();
    }
  }

  async updatePlayerResources() {
    const user = await gameData.getPlayer(authManager.getUser().id);
    if (!user) return;

    // Start with user's global credits
    const userCredits = user.credits;

    // Calculate income from buildings
    const userBuildings = this.getPlayerBuildings();
    const creditsPerTick = userBuildings
      .filter(
        (building) => building.type === "bank" && building.active !== false,
      )
      .reduce((sum, building) => sum + (building.credits_per_tick || 1), 0);

    // Calculate total resources from owned systems
    const ownedSystems = this.systems.filter((s) => s.owner_id === user.id);
    this.playerResources = ownedSystems.reduce(
      (total, system) => ({
        credits: total.credits + (system.credits || 0),
        food: total.food + (system.food || 0),
        ore: total.ore + (system.ore || 0),
        goods: total.goods + (system.goods || 0),
        fuel: total.fuel + (system.fuel || 0),
      }),
      {
        credits: userCredits, // Start with user's global credits
        food: 0,
        ore: 0,
        goods: 0,
        fuel: 0,
      },
    );

    // Store building income for UI display
    this.creditIncome = creditsPerTick;
  }

  selectSystem(systemId) {
    this.selectedSystem = this.systems.find((s) => s.id === systemId) || null;
    this.notifyCallbacks();
  }

  getSelectedSystem() {
    return this.selectedSystem;
  }

  getOwnedSystems() {
    const user = authManager.getUser();
    if (!user) return [];
    return this.systems.filter((s) => s.owner_id === user.id);
  }

  getPlayerFleets() {
    const user = authManager.getUser();
    if (!user) return [];
    return this.fleets.filter((f) => f.owner_id === user.id);
  }

  getPlayerTrades() {
    const user = authManager.getUser();
    if (!user) return [];
    return this.trades.filter((t) => t.owner_id === user.id);
  }

  // Action methods (delegates to gameData)
  async sendFleet(fromId, toId, strength) {
    return await gameData.sendFleet(fromId, toId, strength);
  }

  async queueBuilding(systemId, buildingType) {
    return await gameData.queueBuilding(systemId, buildingType);
  }

  async createTradeRoute(fromId, toId, cargo, capacity) {
    return await gameData.createTradeRoute(fromId, toId, cargo, capacity);
  }

  async proposeTreaty(playerId, type, terms) {
    return await gameData.proposeTreaty(playerId, type, terms);
  }

  getPlayerBuildings() {
    const user = authManager.getUser();
    if (!user) return [];
    return (
      this.buildings?.filter((building) => building.owner_id === user.id) || []
    );
  }

  getPlayerBuildingsByType(type) {
    return this.getPlayerBuildings().filter(
      (building) => building.type === type,
    );
  }
}

// Create singleton game state
export const gameState = new GameState();
