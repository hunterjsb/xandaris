export class FleetComponent {
  constructor(uiController, gameState) {
    this.uiController = uiController;
    this.gameState = gameState;
    this.currentUser = this.uiController.currentUser;
    this.fleet = null;
  }

  setFleet(fleetId) {
    this.fleet = this.gameState.fleets?.find(f => f.id === fleetId);
    return this.fleet;
  }

  render(fleetId) {
    if (!this.setFleet(fleetId)) {
      return '<div class="text-red-400">Fleet not found.</div>';
    }

    const fleetName = this.fleet.name || `Fleet ${this.fleet.id.slice(-4)}`;
    const systems = this.gameState.systems || [];
    const currentSystem = systems.find(s => s.id === this.fleet.current_system);
    const systemName = currentSystem ? currentSystem.name : "Deep Space";

    const isMoving = this.isFleetMoving();
    const movementInfo = isMoving ? this.getMovementInfo() : null;

    return `
      <div class="fleet-details-container">
        ${this.renderFleetHeader(fleetName, systemName, currentSystem)}
        ${isMoving ? this.renderMovementStatus(movementInfo) : ''}
        ${this.renderShipsList()}
        ${this.renderFleetActions(currentSystem)}
      </div>
    `;
  }

  renderFleetHeader(fleetName, systemName, currentSystem) {
    const fleetGradient = this.getFleetGradient();
    
    return `
      <div class="fleet-header panel-header flex justify-between items-center p-3 cursor-move border-b border-space-700/50 bg-gradient-to-r ${fleetGradient}" draggable="false">
        <div class="flex items-center gap-2">
          <span class="material-icons text-space-400 drag-handle">drag_indicator</span>
          <span id="fleet-name" class="text-xl font-bold text-nebula-200">${fleetName}</span>
        </div>
        <div class="flex items-center gap-4">
          <div class="text-right">
            <div id="fleet-seed" class="font-semibold text-nebula-200 text-sm">ID: ${this.fleet.id.slice(-8)}</div>
            <div id="fleet-coords" class="font-mono text-xs text-gray-500">${currentSystem ? `${currentSystem.x}, ${currentSystem.y}` : 'Unknown'}</div>
          </div>
          <div class="flex items-center gap-2">
            <button onclick="window.uiController.togglePinPanel(this.closest('.floating-panel'))" 
                    class="pin-button text-space-400 hover:text-white transition-colors" 
                    title="Pin">
              <span class="material-icons text-sm">push_pin</span>
            </button>
            <button onclick="window.uiController.clearExpandedView()" 
                    class="text-space-400 hover:text-white transition-colors"
                    title="Close">
              <span class="material-icons">close</span>
            </button>
          </div>
        </div>
      </div>
    `;
  }

  getFleetGradient() {
    const isMoving = this.isFleetMoving();
    if (isMoving) {
      return "from-plasma-900/30 to-cyan-900/30";
    } else {
      return "from-nebula-900/30 to-space-900/30";
    }
  }

  renderMovementStatus(movementInfo) {
    if (!movementInfo) return '';

    const { order, originName, destName, etaDisplay, progressPercent, statusDisplay } = movementInfo;

    return `
      <div class="movement-status mb-4 p-4 bg-plasma-900/20 border border-plasma-600/30 rounded-lg">
        <h3 class="text-lg font-semibold mb-3 text-plasma-200 flex items-center gap-2">
          <span class="material-icons text-sm">flight</span>
          Movement Status
        </h3>
        
        <div class="grid grid-cols-2 gap-4 text-sm mb-3">
          <div>
            <div class="text-space-400">From:</div>
            <div class="text-white font-medium">${originName}</div>
          </div>
          <div>
            <div class="text-space-400">To:</div>
            <div class="text-white font-medium">${destName}</div>
          </div>
          <div>
            <div class="text-space-400">ETA:</div>
            <div class="text-yellow-400 font-medium">${etaDisplay}</div>
          </div>
          <div>
            <div class="text-space-400">Status:</div>
            <div class="text-cyan-400 font-medium">${statusDisplay}</div>
          </div>
        </div>

        <div class="mb-2">
          <div class="flex justify-between text-xs mb-1">
            <span class="text-space-400">Progress</span>
            <span class="text-white">${progressPercent}%</span>
          </div>
          <div class="w-full bg-space-600 rounded-full h-3">
            <div class="bg-gradient-to-r from-plasma-500 to-cyan-500 h-3 rounded-full transition-all duration-300" 
                 style="width: ${progressPercent}%"></div>
          </div>
        </div>

        ${this.renderRouteInfo(order)}
      </div>
    `;
  }

  renderRouteInfo(order) {
    if (!order.route_path || order.route_path.length <= 2) return '';

    const currentHop = order.current_hop || 0;
    const totalHops = order.route_path.length - 1;
    const remainingHops = totalHops - currentHop;

    const systems = this.gameState.systems || [];
    const finalDestSystem = systems.find(s => s.id === order.final_destination_id);
    const finalDestName = finalDestSystem ? finalDestSystem.name || `System ${finalDestSystem.id.slice(-4)}` : "Unknown";

    return `
      <div class="route-info border-t border-plasma-600/30 mt-3 pt-3">
        <div class="text-sm font-medium text-purple-300 mb-2">Multi-Hop Route</div>
        <div class="grid grid-cols-2 gap-3 text-xs">
          <div><span class="text-space-400">Final Destination:</span> <span class="text-white">${finalDestName}</span></div>
          <div><span class="text-space-400">Current Hop:</span> <span class="text-cyan-400">${currentHop + 1}/${totalHops}</span></div>
          <div><span class="text-space-400">Remaining Hops:</span> <span class="text-yellow-400">${remainingHops}</span></div>
        </div>
      </div>
    `;
  }

  renderShipsList() {
    if (!this.fleet.ships || this.fleet.ships.length === 0) {
      return `
        <div class="ships-list mb-4 p-4 bg-space-700 rounded-lg border border-space-600">
          <h3 class="text-lg font-semibold mb-3 text-nebula-200 flex items-center gap-2">
            <span class="material-icons text-sm">rocket</span>
            Ships (0)
          </h3>
          <div class="text-space-400 text-center py-4">No ships in this fleet</div>
        </div>
      `;
    }

    // Separate settlers from other ships
    const settlers = this.fleet.ships.filter(ship => 
      ship.ship_type_name && ship.ship_type_name.toLowerCase().includes('settler')
    );
    const otherShips = this.fleet.ships.filter(ship => 
      !ship.ship_type_name || !ship.ship_type_name.toLowerCase().includes('settler')
    );

    const settlersHtml = settlers.length > 0 ? 
      settlers.map(ship => this.renderShipItem(ship, true)).join('') : '';
    
    const otherShipsHtml = otherShips.length > 0 ? 
      otherShips.map(ship => this.renderShipItem(ship, false)).join('') : '';

    return `
      <div class="ships-list mb-4 p-4 bg-space-700 rounded-lg border border-space-600">
        <h3 class="text-lg font-semibold mb-3 text-nebula-200 flex items-center gap-2">
          <span class="material-icons text-sm">rocket</span>
          Ships (${this.fleet.ships.length})
        </h3>
        
        ${settlers.length > 0 ? `
          <div class="settlers-section mb-4">
            <h4 class="text-md font-medium mb-2 text-green-300 flex items-center gap-2">
              <span class="material-icons text-sm">groups</span>
              Settler Ships (${settlers.length})
            </h4>
            <div class="space-y-2">
              ${settlersHtml}
            </div>
          </div>
        ` : ''}
        
        ${otherShips.length > 0 ? `
          <div class="other-ships-section">
            <h4 class="text-md font-medium mb-2 text-cyan-300 flex items-center gap-2">
              <span class="material-icons text-sm">rocket_launch</span>
              Other Ships (${otherShips.length})
            </h4>
            <div class="space-y-2">
              ${otherShipsHtml}
            </div>
          </div>
        ` : ''}
      </div>
    `;
  }

  renderShipItem(ship, isSettler) {
    const shipTypeName = ship.ship_type_name || 'Unknown';
    const shipCount = ship.count || 1;
    const healthPercent = ship.health || 100;
    const healthColor = healthPercent > 75 ? 'text-green-400' : 
                       healthPercent > 50 ? 'text-yellow-400' : 'text-red-400';

    const shipType = this.getShipTypeData(ship);
    const cargoCapacity = shipType?.cargo_capacity || 0;
    const hasCargoCapacity = cargoCapacity > 0;

    const accentColor = isSettler ? 'text-green-400' : 'text-cyan-400';
    const iconName = isSettler ? 'groups' : (hasCargoCapacity ? 'local_shipping' : 'rocket_launch');

    return `
      <div class="ship-item p-3 bg-space-800 rounded border border-space-500 cursor-pointer hover:bg-space-750 transition-colors"
           onclick="window.fleetComponents.showShipDetails('${ship.id}')">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3">
            <span class="material-icons ${accentColor}">${iconName}</span>
            <div>
              <div class="font-medium text-white">${shipCount}x ${shipTypeName}</div>
              <div class="text-xs text-space-300">Ship ID: ${ship.id.slice(-8)}</div>
              ${hasCargoCapacity ? `<div class="text-xs text-orange-400">Cargo: ${cargoCapacity} per ship</div>` : ''}
            </div>
          </div>
          <div class="text-right">
            <div class="text-xs text-space-400">Health</div>
            <div class="text-sm font-medium ${healthColor}">${healthPercent}%</div>
            ${hasCargoCapacity ? `
              <button class="btn btn-xs btn-info mt-1" 
                      onclick="event.stopPropagation(); window.fleetComponents.showShipCargo('${ship.id}')">
                <span class="material-icons text-xs">inventory_2</span>
              </button>
            ` : ''}
          </div>
        </div>
        
        <div class="mt-2">
          <div class="w-full bg-space-600 rounded-full h-1.5">
            <div class="${healthPercent > 75 ? 'bg-green-500' : healthPercent > 50 ? 'bg-yellow-500' : 'bg-red-500'} h-1.5 rounded-full transition-all duration-300" 
                 style="width: ${healthPercent}%"></div>
          </div>
        </div>
      </div>
    `;
  }

  getShipTypeData(ship) {
    const shipTypes = this.gameState.shipTypes || [];
    return shipTypes.find(st => st.id === ship.ship_type || st.name === ship.ship_type_name);
  }

  renderFleetActions(currentSystem) {
    const isMoving = this.isFleetMoving();
    const hasSettlers = this.fleet.ships && this.fleet.ships.some(ship => 
      ship.ship_type_name && ship.ship_type_name.toLowerCase().includes('settler')
    );
    
    return `
      <div class="fleet-actions p-4 bg-space-700 rounded-lg border border-space-600">
        <h3 class="text-lg font-semibold mb-3 text-nebula-200 flex items-center gap-2">
          <span class="material-icons text-sm">settings</span>
          Fleet Actions
        </h3>
        
        <div class="grid grid-cols-2 gap-3">
          <button class="btn btn-primary flex items-center justify-center gap-2" 
                  onclick="window.fleetComponents.sendFleet('${this.fleet.id}')"
                  ${isMoving ? 'disabled' : ''}>
            <span class="material-icons text-sm">send</span>
            Move Fleet
          </button>
          
          ${currentSystem ? `
            <button class="btn btn-info flex items-center justify-center gap-2"
                    onclick="window.fleetComponents.transferCargo('${this.fleet.id}', '${currentSystem.id}')"
                    ${isMoving ? 'disabled' : ''}>
              <span class="material-icons text-sm">swap_horiz</span>
              Transfer Cargo
            </button>
          ` : `
            <button class="btn btn-secondary flex items-center justify-center gap-2" disabled>
              <span class="material-icons text-sm">block</span>
              No System
            </button>
          `}
          
          ${hasSettlers && currentSystem ? `
            <button class="btn btn-success flex items-center justify-center gap-2 col-span-2"
                    onclick="window.fleetComponents.colonize('${currentSystem.id}')"
                    ${isMoving ? 'disabled' : ''}>
              <span class="material-icons text-sm">home</span>
              Colonize Planet
            </button>
          ` : ''}
        </div>
      </div>
    `;
  }

  isFleetMoving() {
    const allFleetOrders = this.gameState.fleetOrders || [];
    return allFleetOrders.some(order => 
      order.fleet_id === this.fleet.id && 
      (order.status === "pending" || order.status === "processing")
    );
  }

  getMovementInfo() {
    const allFleetOrders = this.gameState.fleetOrders || [];
    const order = allFleetOrders.find(order => 
      order.fleet_id === this.fleet.id && 
      (order.status === "pending" || order.status === "processing")
    );

    if (!order) return null;

    const systems = this.gameState.systems || [];
    const currentTick = this.gameState.currentTick || 0;
    const TICKS_PER_MINUTE = this.gameState.ticksPerMinute || 6;
    const SECONDS_PER_TICK = 60 / TICKS_PER_MINUTE;

    const originSystem = systems.find(s => s.id === this.fleet.current_system);
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

    return {
      order,
      originName,
      destName,
      etaDisplay,
      progressPercent,
      statusDisplay
    };
  }
}