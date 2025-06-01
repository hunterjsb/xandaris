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
    this.selectedSystemPlanets = []; // Added to store planets of the selected system
    this.currentTick = 1;
    this.ticksPerMinute = 6;
    this.buildingTypes = [];
    this.resourceTypes = [];
    this.playerResources = {
      credits: 0,
      food: 0,
      ore: 0,
      goods: 0,
      fuel: 0,
    };
    this.creditIncome = 0; // To store calculated income per tick

    this.callbacks = [];
    this.initialized = false;
    
    // Debouncing and update prevention
    this.updateTimer = null;
    this.pendingUpdate = false;
    this.isUpdating = false;

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

        // Player resources will be loaded via updatePlayerResources()
      }

      // Load status last
      const status = await gameData.getStatus();
      if (status) {
        this.currentTick = status.current_tick || 1;
        this.ticksPerMinute = status.ticks_per_minute || 6;
      }

      // Load building and resource types
      try {
        const buildingTypesData = await gameData.getBuildingTypes();
        if (buildingTypesData) {
          this.buildingTypes = buildingTypesData;
        }
      } catch (error) {
        console.warn("Failed to load building types:", error);
        this.buildingTypes = []; // Ensure it's an array even on failure
      }

      try {
        const resourceTypesData = await gameData.getResourceTypes();
        if (resourceTypesData) {
          this.resourceTypes = resourceTypesData;
        }
      } catch (error) {
        console.warn("Failed to load resource types:", error);
        this.resourceTypes = []; // Ensure it's an array even on failure
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
    this.selectedSystemPlanets = [];
    this.currentTick = 1;
    this.ticksPerMinute = 6;
    this.buildingTypes = [];
    this.resourceTypes = [];
    this.playerResources = {
      credits: 0,
      food: 0,
      ore: 0,
      goods: 0,
      fuel: 0,
    };
    this.creditIncome = 0;
    this.initialized = false;
    this.notifyCallbacks();
  }

  async loadMapData() {
    try {
      const mapData = await gameData.getMap();
      if (mapData && mapData.systems) {
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
    // Prevent recursive updates
    if (this.isUpdating) return;
    
    // Debounce rapid updates
    if (this.updateTimer) {
      this.pendingUpdate = true;
      return;
    }
    
    this.updateTimer = setTimeout(() => {
      this.updateTimer = null;
      this.isUpdating = true;
      
      try {
        this.callbacks.forEach((callback) => callback(this));
      } finally {
        this.isUpdating = false;
      }
      
      if (this.pendingUpdate) {
        this.pendingUpdate = false;
        this.notifyCallbacks();
      }
    }, 16); // ~60fps update rate
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

    // Refresh data every tick, which includes resource calculation.
    // Consider optimizing if performance becomes an issue.
    this.refreshGameData();
    // notifyCallbacks() is called within refreshGameData and updatePlayerResources
  }

  async updatePlayerResources() {
    const user = authManager.getUser(); // Get current user
    if (!user) {
      this.playerResources = { credits: 0, food: 0, ore: 0, goods: 0, fuel: 0 };
      this.creditIncome = 0;
      return;
    }

    // Get user resources from the new API endpoint
    const userResources = await gameData.getUserResources();
    
    if (!this.mapData || !this.mapData.planets) {
      this.playerResources = userResources;
      this.creditIncome = 0;
      return;
    }

    let totalFood = 0;
    let totalOre = 0;
    let totalGoods = 0;
    let totalFuel = 0;
    let calculatedCreditIncome = 0;

    // Iterate through all planets to find those owned by the player
    for (const planet of this.mapData.planets) {
      if (planet.colonized_by === user.id) {
        totalFood += planet.Food || 0;
        totalOre += planet.Ore || 0;
        totalGoods += planet.Goods || 0;
        totalFuel += planet.Fuel || 0;
        // planet.Credits are accumulated resources on planet, not direct income to player from planet itself.
        // Income comes from buildings.

        if (planet.Buildings) {
          for (const [buildingIdOrName, level] of Object.entries(planet.Buildings)) {
            // Find building type details. The ID in planet.Buildings could be the type's ID or its name.
            const buildingType = this.buildingTypes.find(bt => bt.id === buildingIdOrName || (bt.name && bt.name.toLowerCase() === buildingIdOrName.toLowerCase()));
            if (buildingType && buildingType.name && buildingType.name.toLowerCase() === 'bank') {
              // Example: banks generate 1 credit per level per tick.
              // This should align with backend calculations or game design.
              calculatedCreditIncome += (level || 1) * 1;
            }
            // TODO: Add other resource-generating buildings here if they contribute to per-tick income
            // For example, if farms directly add to playerResources.Food per tick beyond what's stored on planet.
            // For now, assuming Food, Ore, Goods, Fuel on planet are the current stockpile.
          }
        }
      }
    }

    this.playerResources = {
      credits: userResources.credits, // Credits from crypto_server buildings
      food: totalFood,
      ore: totalOre,
      goods: totalGoods,
      fuel: totalFuel,
    };
    this.creditIncome = calculatedCreditIncome; // Store income calculated from buildings for UI display

    // No notifyCallbacks() here, it's usually called by the initiator (e.g. loadGameData, handleTick)
    // However, since refreshGameData calls this, and refreshGameData calls notifyCallbacks, it's covered.
  }

  getSystemPlanets(systemId) {
    if (!this.mapData || !this.mapData.planets) {
      return [];
    }
    // Remove excessive logging
    const filtered = this.mapData.planets.filter(planet => {
      // The backend returns system_id as a string
      return planet.system_id === systemId;
    });
    return filtered;
  }

  selectSystem(systemId) {
    // Prevent unnecessary updates if selecting the same system
    if (this.selectedSystem && this.selectedSystem.id === systemId) {
      return;
    }
    
    this.selectedSystem = this.systems.find((s) => s.id === systemId) || null;
    if (this.selectedSystem) {
      this.selectedSystemPlanets = this.getSystemPlanets(this.selectedSystem.id);
    } else {
      this.selectedSystemPlanets = [];
    }
    // The primary update path for UIController's expanded view is via 'systemSelected' event from mapRenderer.
    // This notifyCallbacks is for other direct subscribers to gameState if any.
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

  async queueBuilding(planetId, buildingType) { // Renamed systemId to planetId
    return await gameData.queueBuilding(planetId, buildingType); // Pass planetId
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
