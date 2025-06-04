import { gameData } from '../../lib/pocketbase.js';

export class FleetListComponent {
  constructor(uiController, gameState) {
    this.uiController = uiController;
    this.gameState = gameState;
    this.currentUser = this.uiController.currentUser;
  }

  render() {
    if (!this.gameState || !this.currentUser) {
      return '<div class="text-space-400">Game data not loaded or user not available.</div>';
    }

    const movingFleets = this.getMovingFleets();
    const stationaryFleets = this.getStationaryFleets();

    return `
      <div class="fleet-list-container">
        ${movingFleets.length > 0 ? this.renderMovingFleets(movingFleets) : ''}
        ${stationaryFleets.length > 0 ? this.renderStationaryFleets(stationaryFleets) : ''}
        ${movingFleets.length === 0 && stationaryFleets.length === 0 ? 
          this.renderNoFleets() : ''}
      </div>
    `;
  }

  getMovingFleets() {
    const allFleetOrders = this.gameState.fleetOrders || [];
    const allFleets = this.gameState.fleets || [];
    const currentUserId = this.currentUser.id;

    return allFleetOrders
      .filter(order => 
        order.user_id === currentUserId && 
        (order.status === "pending" || order.status === "processing")
      )
      .map(order => {
        const fleet = allFleets.find(f => f.id === order.fleet_id);
        return fleet ? { fleet, order } : null;
      })
      .filter(Boolean)
      .sort((a, b) => a.order.execute_at_tick - b.order.execute_at_tick);
  }

  getStationaryFleets() {
    const movingFleetIds = new Set(this.getMovingFleets().map(mf => mf.fleet.id));
    return this.gameState.getPlayerFleets().filter(fleet => !movingFleetIds.has(fleet.id));
  }

  renderMovingFleets(movingFleets) {
    const fleetsHtml = movingFleets.map(({ fleet, order }) => this.renderMovingFleet(fleet, order)).join('');
    
    return `
      <div class="mb-6">
        <h3 class="text-lg font-semibold mb-3 text-plasma-200 flex items-center gap-2">
          <span class="material-icons text-sm">flight_takeoff</span>
          Moving Fleets (${movingFleets.length})
        </h3>
        <div class="space-y-3">
          ${fleetsHtml}
        </div>
      </div>
    `;
  }

  renderStationaryFleets(stationaryFleets) {
    const fleetsHtml = stationaryFleets.map(fleet => this.renderStationaryFleet(fleet)).join('');
    
    return `
      <div>
        <h3 class="text-lg font-semibold mb-3 text-nebula-200 flex items-center gap-2">
          <span class="material-icons text-sm">anchor</span>
          Docked Fleets (${stationaryFleets.length})
        </h3>
        <div class="space-y-3">
          ${fleetsHtml}
        </div>
      </div>
    `;
  }

  renderMovingFleet(fleet, order) {
    const fleetName = fleet.name || `Fleet ${fleet.id.slice(-4)}`;
    const systems = this.gameState.systems || [];
    const currentTick = this.gameState.currentTick || 0;
    const TICKS_PER_MINUTE = this.gameState.ticksPerMinute || 6;
    const SECONDS_PER_TICK = 60 / TICKS_PER_MINUTE;

    const originSystem = systems.find(s => s.id === fleet.current_system);
    const originName = originSystem ? originSystem.name : "Deep Space";

    const destinationSystem = systems.find(s => s.id === order.destination_system_id);
    const destName = destinationSystem ? destinationSystem.name : "Unknown System";

    const ticksRemaining = Math.max(0, order.execute_at_tick - currentTick);
    const secondsRemaining = (ticksRemaining * SECONDS_PER_TICK).toFixed(0);
    let etaDisplay = `${ticksRemaining} ticks (~${secondsRemaining}s)`;
    
    if (ticksRemaining === 0 && order.status === "processing") {
      etaDisplay = "Finalizing Jump";
    } else if (ticksRemaining === 0 && order.status === "pending") {
      etaDisplay = "Initiating Jump";
    }

    const totalMovementTicks = order.travel_time_ticks || 2;
    const progressTicks = totalMovementTicks - ticksRemaining;
    const progressPercent = Math.round((progressTicks / totalMovementTicks) * 100);

    const statusDisplay = order.status.charAt(0).toUpperCase() + order.status.slice(1);

    let routeInfo = "";
    if (order.route_path && order.route_path.length > 2) {
      const currentHop = order.current_hop || 0;
      const totalHops = order.route_path.length - 1;
      const remainingHops = totalHops - currentHop;

      const finalDestSystem = systems.find(s => s.id === order.final_destination_id);
      const finalDestName = finalDestSystem ? finalDestSystem.name || `System ${finalDestSystem.id.slice(-4)}` : "Unknown";

      routeInfo = `
        <div class="border-t border-space-500 mt-2 pt-2 text-xs">
          <div class="grid grid-cols-2 gap-2">
            <div><span class="text-space-400">Route:</span> <span class="text-purple-400">Multi-hop</span></div>
            <div><span class="text-space-400">Final:</span> ${finalDestName}</div>
            <div><span class="text-space-400">Hop:</span> <span class="text-cyan-400">${currentHop + 1}/${totalHops}</span></div>
            <div><span class="text-space-400">Remaining:</span> <span class="text-yellow-400">${remainingHops} hops</span></div>
          </div>
        </div>
      `;
    }

    return `
      <div class="fleet-item bg-space-700 p-4 rounded-lg border border-plasma-600/30 cursor-pointer hover:bg-space-600 transition-all duration-200 shadow-md hover:shadow-lg"
           onclick="window.fleetComponents.showFleetDetails('${fleet.id}')">
        <div class="flex items-start justify-between mb-2">
          <div class="flex items-center gap-2">
            <span class="material-icons text-plasma-400">rocket_launch</span>
            <div>
              <div class="font-semibold text-plasma-200">${fleetName}</div>
              <div class="text-xs text-space-300">Fleet ID: ${fleet.id.slice(-8)}</div>
            </div>
          </div>
          <div class="text-right">
            <div class="text-xs text-space-400">Status</div>
            <div class="text-sm font-medium text-cyan-400">${statusDisplay}</div>
          </div>
        </div>
        
        <div class="grid grid-cols-2 gap-3 text-sm">
          <div>
            <div class="text-space-400">From:</div>
            <div class="text-white truncate">${originName}</div>
          </div>
          <div>
            <div class="text-space-400">To:</div>
            <div class="text-white truncate">${destName}</div>
          </div>
          <div>
            <div class="text-space-400">ETA:</div>
            <div class="text-yellow-400">${etaDisplay}</div>
          </div>
          <div>
            <div class="text-space-400">Progress:</div>
            <div class="text-green-400">${progressPercent}%</div>
          </div>
        </div>

        <div class="w-full bg-space-600 rounded-full h-2 mt-3">
          <div class="bg-gradient-to-r from-plasma-500 to-cyan-500 h-2 rounded-full transition-all duration-300" 
               style="width: ${progressPercent}%"></div>
        </div>

        ${routeInfo}
      </div>
    `;
  }

  renderStationaryFleet(fleet) {
    const fleetName = fleet.name || `Fleet ${fleet.id.slice(-4)}`;
    const systems = this.gameState.systems || [];
    const currentSystem = systems.find(s => s.id === fleet.current_system);
    const systemName = currentSystem ? currentSystem.name : "Deep Space";

    const cargoData = this.gameState?.getFleetCargo(fleet.id) || { cargo: {}, used_capacity: 0, total_capacity: 0 };
    const shipCount = fleet.ships ? fleet.ships.reduce((sum, ship) => sum + ship.count, 0) : 0;
    const shipTypes = fleet.ships ? fleet.ships.length : 0;

    let cargoSummary = "Empty";
    if (cargoData.cargo && Object.keys(cargoData.cargo).length > 0) {
      const totalItems = Object.values(cargoData.cargo).reduce((sum, amount) => sum + amount, 0);
      cargoSummary = `${totalItems} units`;
    }

    return `
      <div class="fleet-item bg-space-700 p-4 rounded-lg border border-nebula-600/30 cursor-pointer hover:bg-space-600 transition-all duration-200 shadow-md hover:shadow-lg"
           onclick="window.fleetComponents.showFleetDetails('${fleet.id}')">
        <div class="flex items-start justify-between mb-3">
          <div class="flex items-center gap-2">
            <span class="material-icons text-nebula-400">anchor</span>
            <div>
              <div class="font-semibold text-nebula-200">${fleetName}</div>
              <div class="text-xs text-space-300">Fleet ID: ${fleet.id.slice(-8)}</div>
            </div>
          </div>
          <div class="text-right">
            <div class="text-xs text-space-400">Status</div>
            <div class="text-sm font-medium text-green-400">Docked</div>
          </div>
        </div>
        
        <div class="grid grid-cols-2 gap-3 text-sm">
          <div>
            <div class="text-space-400">Location:</div>
            <div class="text-white truncate">${systemName}</div>
          </div>
          <div>
            <div class="text-space-400">Ships:</div>
            <div class="text-white">${shipCount} (${shipTypes} types)</div>
          </div>
          <div>
            <div class="text-space-400">Cargo:</div>
            <div class="text-white">${cargoSummary}</div>
          </div>
          <div>
            <div class="text-space-400">Capacity:</div>
            <div class="text-white">${cargoData.used_capacity}/${cargoData.total_capacity}</div>
          </div>
        </div>

        ${cargoData.total_capacity > 0 ? `
          <div class="mt-3">
            <div class="flex justify-between text-xs mb-1">
              <span class="text-space-400">Cargo Usage</span>
              <span class="text-white">${Math.round((cargoData.used_capacity / cargoData.total_capacity) * 100)}%</span>
            </div>
            <div class="w-full bg-space-600 rounded-full h-2">
              <div class="bg-gradient-to-r from-nebula-500 to-blue-500 h-2 rounded-full transition-all duration-300" 
                   style="width: ${(cargoData.used_capacity / cargoData.total_capacity) * 100}%"></div>
            </div>
          </div>
        ` : ''}
      </div>
    `;
  }

  renderNoFleets() {
    return `
      <div class="text-center py-12">
        <span class="material-icons text-6xl text-space-500 mb-4">rocket_launch</span>
        <div class="text-space-400 text-lg mb-6">No fleets found</div>
        <div class="text-space-500 text-sm mb-6">You need ships to explore the galaxy and colonize planets</div>
        
        <div class="bg-space-800 rounded-lg p-6 border border-space-600 max-w-md mx-auto">
          <h3 class="text-orange-200 font-semibold mb-3">Debug: Spawn Starter Ship</h3>
          <div class="text-space-400 text-sm mb-4">
            Get a settler ship loaded with materials:
            <br>• 50 ore • 25 food • 20 metal • 15 fuel
          </div>
          <button 
            class="btn btn-primary w-full flex items-center justify-center gap-2"
            onclick="window.fleetComponents.spawnStarterShip()"
            id="spawn-starter-btn">
            <span class="material-icons text-sm">add</span>
            Spawn Starter Ship
          </button>
        </div>
      </div>
    `;
  }

  async spawnStarterShip() {
    const button = document.getElementById('spawn-starter-btn');
    if (button) {
      button.disabled = true;
      button.innerHTML = '<span class="material-icons animate-spin text-sm">refresh</span> Spawning...';
    }

    try {
      const result = await gameData.spawnStarterShip();
      
      if (result.success) {
        this.uiController.showToast('Starter ship spawned successfully!', 'success', 4000);
        
        // Refresh the game state to show the new fleet
        await this.gameState.lightweightTickUpdate();
        
        // Refresh the fleet list
        this.uiController.refreshActiveComponent();
      }
    } catch (error) {
      console.error('Failed to spawn starter ship:', error);
      this.uiController.showToast(`Failed to spawn starter ship: ${error.message}`, 'error', 5000);
    } finally {
      if (button) {
        button.disabled = false;
        button.innerHTML = '<span class="material-icons text-sm">add</span> Spawn Starter Ship';
      }
    }
  }
}