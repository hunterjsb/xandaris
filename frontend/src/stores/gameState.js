// Game state management similar to vibe-chuck's events store
import { gameData, authManager } from "../lib/pocketbase.js";

export class GameState {
  constructor() {
    this.systems = [];
    this.fleets = [];
    this.trades = [];
    this.treaties = [];
    this.buildings = [];
    this.populations = [];
    this.fleetOrders = []; // Renamed from orders
    this.hyperlanes = [];
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
    this.shipCargo = new Map(); // Store cargo data for fleets

    this.callbacks = [];
    this.initialized = false;
    this.updatingResources = false; // Prevent overlapping resource updates

    // Debouncing and update prevention
    this.updateTimer = null;
    this.tickRefreshTimer = null; // Debounce tick refreshes
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
    gameData.subscribe("fleet_orders", (data) => this.updateFleetOrders(data)); // Renamed subscription
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
      // Load map data
      const mapData = await gameData.getMap();
      if (mapData && mapData.systems) {
        this.systems = mapData.systems;
        this.mapData = mapData;
      }

      // Load hyperlanes
      const hyperlanes = await gameData.getHyperlanes();
      if (hyperlanes) {
        this.hyperlanes = hyperlanes;
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

        this.populations = await gameData.getPopulations(userId);
        await new Promise((resolve) => setTimeout(resolve, 50));

        this.fleetOrders = await gameData.getFleetOrders(userId); // Use getFleetOrders
        await new Promise(resolve => setTimeout(resolve, 50));

        // Load ship cargo for all user fleets (only on full refresh, not every tick)
        if (this.fleets && this.fleets.length > 0 && !this.cargoLoaded) {
          for (const fleet of this.fleets) {
            try {
              const cargoData = await this.getShipCargo(fleet.id);
              await new Promise(resolve => setTimeout(resolve, 25)); // Small delay between cargo requests
            } catch (error) {
              console.warn(`Failed to load cargo for fleet ${fleet.id}:`, error);
            }
          }
          this.cargoLoaded = true;
        }

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

      // Center camera on player's first fleet if this is their first time
      if (userId && this.fleets && this.fleets.length > 0 && this.systems && this.systems.length > 0) {
        const firstFleet = this.fleets[0];
        if (firstFleet && firstFleet.current_system) {
          // Notify callbacks to trigger camera centering
          this.centerOnFleetSystem = firstFleet.current_system;
        }
      }

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

  async lightweightTickUpdate() {
    // Only update essential data that changes frequently
    const user = authManager.getUser();
    if (!user) return;

    try {
      // Update only resources and status - don't reload everything
      await this.updatePlayerResources();

      // Update status for tick counter
      const status = await gameData.getStatus();
      if (status) {
        this.currentTick = status.current_tick || this.currentTick;
        this.ticksPerMinute = status.ticks_per_minute || this.ticksPerMinute;
      }

      this.notifyCallbacks();
    } catch (error) {
      console.warn("Failed to perform lightweight tick update:", error);
    }
  }

  handleTick(tickData) {
    console.log("Received tick:", tickData);
    this.currentTick = tickData.tick || tickData.current_tick || this.currentTick;

    // Debounce rapid tick updates
    if (this.tickRefreshTimer) {
      clearTimeout(this.tickRefreshTimer);
    }

    this.tickRefreshTimer = setTimeout(() => {
      // Only do full data refresh every 10th tick to reduce API load
      // Otherwise do lightweight updates for resources/status only
      if (this.currentTick % 10 === 0) {
        this.refreshGameData();
      } else {
        this.lightweightTickUpdate();
      }
    }, 500); // Increased debounce to 500ms
  }

  reset() {
    this.systems = [];
    this.fleets = [];
    this.trades = [];
    this.treaties = [];
    this.buildings = [];
    this.populations = [];
    this.fleetOrders = []; // Reset fleetOrders
    this.hyperlanes = [];
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

        // Also load hyperlanes when map data is loaded
        const hyperlanes = await gameData.getHyperlanes();
        if (hyperlanes) {
          this.hyperlanes = hyperlanes;
        }

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
    console.log(`DEBUG: updateFleets called with:`, Array.isArray(fleetsData) ? `array of ${fleetsData.length} fleets` : 'single fleet', fleetsData);

    if (Array.isArray(fleetsData)) {
      // Check for fleet arrivals when updating with array
      const oldFleets = new Map(this.fleets.map(f => [f.id, f]));
      console.log(`DEBUG: Checking ${fleetsData.length} fleets for arrivals against ${oldFleets.size} old fleets`);

      for (const newFleet of fleetsData) {
        const oldFleet = oldFleets.get(newFleet.id);
        if (oldFleet) {
          console.log(`DEBUG: Fleet ${newFleet.id} - old dest: "${oldFleet.destination_system}", new dest: "${newFleet.destination_system}"`);
          if (oldFleet.destination_system && !newFleet.destination_system) {
            console.log(`DEBUG: Fleet arrival detected for fleet ${newFleet.id}, old destination: ${oldFleet.destination_system}, new destination: ${newFleet.destination_system}`);
            this.handleFleetArrival(newFleet.id);
          }
        }
      }

      this.fleets = fleetsData;
    } else {
      const index = this.fleets.findIndex((f) => f.id === fleetsData.id);
      if (index >= 0) {
        const oldFleet = this.fleets[index];
        this.fleets[index] = fleetsData;

        // Check for fleet arrival (had destination, now doesn't)
        if (oldFleet.destination_system && !fleetsData.destination_system) {
          console.log(`DEBUG: Fleet arrival detected for fleet ${fleetsData.id}, old destination: ${oldFleet.destination_system}, new destination: ${fleetsData.destination_system}`);
          this.handleFleetArrival(fleetsData.id);
        }
      } else {
        this.fleets.push(fleetsData);
      }
    }
    this.notifyCallbacks();
  }

  handleFleetArrival(fleetId) {
    console.log(`DEBUG: Fleet ${fleetId} arrived, checking for multi-hop continuation`);
    // Notify main app about fleet arrival for multi-hop continuation
    if (window.app && typeof window.app.onFleetArrival === 'function') {
      console.log(`DEBUG: Calling window.app.onFleetArrival for fleet ${fleetId}`);
      window.app.onFleetArrival(fleetId);
    } else {
      console.warn(`DEBUG: window.app.onFleetArrival not available for fleet ${fleetId}`);
    }
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

  updateFleetOrders(fleetOrdersData) { // Renamed from updateOrders
    if (Array.isArray(fleetOrdersData)) {
      this.fleetOrders = fleetOrdersData;
    } else {
      // Single order update
      const index = this.fleetOrders.findIndex((o) => o.id === fleetOrdersData.id);
      if (index >= 0) {
        this.fleetOrders[index] = fleetOrdersData;
      } else {
        this.fleetOrders.push(fleetOrdersData);
        // Keep orders sorted by execute_at_tick
        this.fleetOrders.sort((a, b) => a.execute_at_tick - b.execute_at_tick);
      }
    }
    this.notifyCallbacks();
  }



  async updatePlayerResources() {
    const user = authManager.getUser(); // Get current user
    if (!user) {
      this.playerResources = { credits: 0, food: 0, ore: 0, goods: 0, fuel: 0 };
      this.creditIncome = 0;
      return;
    }

    // Prevent overlapping resource updates
    if (this.updatingResources) {
      return;
    }
    this.updatingResources = true;

    try {
      // Get user resources from the new API endpoint
      const userResources = await gameData.getUserResources();

      // Handle auto-cancellation gracefully
      if (userResources === null) {
        // Request was cancelled, skip this update
        return;
      }

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

    // Calculate ship cargo resources
    let shipCargoOre = 0;
    let shipCargoFood = 0;
    let shipCargoFuel = 0;
    let shipCargoMetal = 0;
    let shipCargoTitanium = 0;
    let shipCargoXanium = 0;

    // Sum up resources from ship cargo across all user's fleets
    for (const [fleetId, cargoData] of this.shipCargo.entries()) {
      if (cargoData && cargoData.cargo) {
        shipCargoOre += cargoData.cargo.ore || 0;
        shipCargoFood += cargoData.cargo.food || 0;
        shipCargoFuel += cargoData.cargo.fuel || 0;
        shipCargoMetal += cargoData.cargo.metal || 0;
        shipCargoTitanium += cargoData.cargo.titanium || 0;
        shipCargoXanium += cargoData.cargo.xanium || 0;
      }
    }

    // Calculate building stored resources
    let buildingStoredOre = 0;
    let buildingStoredCredits = 0;

    // Sum up resources from building storage on user's planets
    if (this.buildings && Array.isArray(this.buildings)) {
      for (const building of this.buildings) {
        // Get building type to determine what's stored
        const buildingType = this.buildingTypes.find(bt => bt.id === building.building_type);
        if (buildingType) {
          const buildingName = buildingType.name?.toLowerCase();

          if (buildingName === 'mine' && building.res1_stored > 0) {
            // Mines store ore in res1_stored
            buildingStoredOre += building.res1_stored;
          } else if (buildingName === 'crypto_server' && building.res1_stored > 0) {
            // Crypto servers store credits in res1_stored (already handled by userResources.credits)
            buildingStoredCredits += building.res1_stored;
          }
          // Add other building types as needed
        }
      }
    }

    this.playerResources = {
      credits: userResources.credits, // Credits from crypto_server buildings
      food: totalFood + shipCargoFood,
      ore: totalOre + shipCargoOre + buildingStoredOre, // Include mine storage
      fuel: totalFuel + shipCargoFuel,
      metal: shipCargoMetal, // Metal only comes from ship cargo for now
      titanium: shipCargoTitanium, // Titanium only comes from ship cargo for now
      xanium: shipCargoXanium, // Xanium only comes from ship cargo for now
    };
    this.creditIncome = calculatedCreditIncome; // Store income calculated from buildings for UI display

    // No notifyCallbacks() here, it's usually called by the initiator (e.g. loadGameData, handleTick)
    // However, since refreshGameData calls this, and refreshGameData calls notifyCallbacks, it's covered.
    } finally {
      this.updatingResources = false;
    }
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

  async queueBuilding(planetId, buildingType, fleetId) { // Added fleetId parameter
    return await gameData.queueBuilding(planetId, buildingType, fleetId); // Pass fleetId
  }

  async getShipCargo(fleetId) {
    try {
      const cargoData = await gameData.getShipCargo(fleetId);
      this.shipCargo.set(fleetId, cargoData);
      this.notifyCallbacks();
      return cargoData;
    } catch (error) {
      console.error("Failed to load ship cargo:", error);
      throw error;
    }
  }

  async refreshAllShipCargo() {
    if (!this.fleets || this.fleets.length === 0) return;

    const user = authManager.getUser();
    if (!user) return;

    const userFleets = this.fleets.filter(fleet => fleet.owner_id === user.id);

    for (const fleet of userFleets) {
      try {
        await this.getShipCargo(fleet.id);
        await new Promise(resolve => setTimeout(resolve, 25)); // Small delay between requests
      } catch (error) {
        console.warn(`Failed to refresh cargo for fleet ${fleet.id}:`, error);
      }
    }
  }

  getFleetCargo(fleetId) {
    return this.shipCargo.get(fleetId) || { cargo: {}, used_capacity: 0, total_capacity: 0 };
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
