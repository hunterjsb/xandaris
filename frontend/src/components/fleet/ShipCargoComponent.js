export class ShipCargoComponent {
  constructor(uiController, gameState) {
    this.uiController = uiController;
    this.gameState = gameState;
    this.currentUser = this.uiController.currentUser;
    this.ship = null;
    this.fleet = null;
  }

  setShip(shipId) {
    // Find the ship across all fleets
    const allFleets = this.gameState.fleets || [];
    for (const fleet of allFleets) {
      if (fleet.ships) {
        const ship = fleet.ships.find(s => s.id === shipId);
        if (ship) {
          this.ship = ship;
          this.fleet = fleet;
          return ship;
        }
      }
    }
    return null;
  }

  render(shipId) {
    if (!this.setShip(shipId)) {
      return '<div class="text-red-400">Ship not found.</div>';
    }

    const shipTypeName = this.ship.ship_type_name || 'Unknown Ship';
    const shipCount = this.ship.count || 1;
    const fleetName = this.fleet.name || `Fleet ${this.fleet.id.slice(-4)}`;

    return `
      <div class="ship-cargo-container">
        ${this.renderShipHeader(shipTypeName, shipCount, fleetName)}
        ${this.renderCargoOverview()}
        ${this.renderCargoDetails()}
        ${this.renderCargoActions()}
      </div>
    `;
  }

  renderShipHeader(shipTypeName, shipCount, fleetName) {
    const systems = this.gameState.systems || [];
    const currentSystem = systems.find(s => s.id === this.fleet.current_system);
    const systemName = currentSystem ? currentSystem.name : "Deep Space";

    return `
      <div class="ship-header mb-4 p-4 bg-gradient-to-r from-space-800 to-space-700 rounded-lg border border-space-600">
        <div class="flex items-start justify-between">
          <div class="flex items-center gap-3">
            <span class="material-icons text-2xl text-orange-400">inventory_2</span>
            <div>
              <h2 class="text-xl font-bold text-white">${shipCount}x ${shipTypeName}</h2>
              <div class="text-sm text-space-300">Ship ID: ${this.ship.id.slice(-8)}</div>
              <div class="text-sm text-space-300">Fleet: ${fleetName}</div>
              <div class="text-sm text-space-300">Location: ${systemName}</div>
            </div>
          </div>
          <div class="text-right">
            <div class="text-xs text-space-400">Cargo Management</div>
            <div class="text-sm font-semibold text-orange-400">Detailed View</div>
          </div>
        </div>
      </div>
    `;
  }

  renderCargoOverview() {
    const shipType = this.getShipTypeData();
    const cargoCapacity = shipType?.cargo_capacity || 0;
    const totalCapacity = cargoCapacity * (this.ship.count || 1);
    const cargoData = this.getShipCargo();
    const usedCapacity = cargoData.reduce((sum, cargo) => sum + cargo.quantity, 0);
    const usagePercent = totalCapacity > 0 ? Math.round((usedCapacity / totalCapacity) * 100) : 0;
    const availableCapacity = totalCapacity - usedCapacity;

    if (totalCapacity === 0) {
      return `
        <div class="cargo-overview mb-4 p-4 bg-space-700 rounded-lg border border-space-600">
          <h3 class="text-lg font-semibold mb-3 text-orange-200 flex items-center gap-2">
            <span class="material-icons text-sm">info</span>
            Cargo Overview
          </h3>
          <div class="text-center py-6">
            <span class="material-icons text-4xl text-space-500 mb-2">block</span>
            <div class="text-space-400">This ship type cannot carry cargo</div>
            <div class="text-xs text-space-500 mt-1">Combat vessels have no cargo capacity</div>
          </div>
        </div>
      `;
    }

    return `
      <div class="cargo-overview mb-4 p-4 bg-space-700 rounded-lg border border-space-600">
        <h3 class="text-lg font-semibold mb-3 text-orange-200 flex items-center gap-2">
          <span class="material-icons text-sm">assessment</span>
          Cargo Overview
        </h3>
        
        <div class="grid grid-cols-3 gap-4 mb-4">
          <div class="text-center p-3 bg-space-800 rounded">
            <div class="text-2xl font-bold text-blue-400">${totalCapacity}</div>
            <div class="text-space-400 text-sm">Total Capacity</div>
          </div>
          <div class="text-center p-3 bg-space-800 rounded">
            <div class="text-2xl font-bold text-green-400">${usedCapacity}</div>
            <div class="text-space-400 text-sm">Used Space</div>
          </div>
          <div class="text-center p-3 bg-space-800 rounded">
            <div class="text-2xl font-bold text-purple-400">${availableCapacity}</div>
            <div class="text-space-400 text-sm">Available</div>
          </div>
        </div>

        <div class="cargo-bar">
          <div class="flex justify-between text-sm mb-2">
            <span class="text-space-400">Capacity Usage</span>
            <span class="text-white">${usagePercent}% Full</span>
          </div>
          <div class="w-full bg-space-600 rounded-full h-4 relative overflow-hidden">
            <div class="bg-gradient-to-r from-orange-500 to-yellow-500 h-4 rounded-full transition-all duration-500" 
                 style="width: ${usagePercent}%"></div>
            ${usagePercent > 95 ? '<div class="absolute inset-0 bg-red-500/20 animate-pulse"></div>' : ''}
          </div>
          <div class="text-center mt-2">
            <span class="text-xs ${usagePercent > 95 ? 'text-red-400 font-bold' : usagePercent > 85 ? 'text-yellow-400' : 'text-green-400'}">
              ${usagePercent > 95 ? 'OVERLOADED' : usagePercent > 85 ? 'Nearly Full' : 'Good Capacity'}
            </span>
          </div>
        </div>
      </div>
    `;
  }

  renderCargoDetails() {
    const cargoData = this.getShipCargo();
    const shipType = this.getShipTypeData();
    const cargoCapacity = shipType?.cargo_capacity || 0;

    if (cargoCapacity === 0) {
      return '';
    }

    if (cargoData.length === 0) {
      return `
        <div class="cargo-details mb-4 p-4 bg-space-700 rounded-lg border border-space-600">
          <h3 class="text-lg font-semibold mb-3 text-orange-200 flex items-center gap-2">
            <span class="material-icons text-sm">inventory</span>
            Cargo Hold Contents
          </h3>
          <div class="text-center py-8">
            <span class="material-icons text-6xl text-space-500 mb-3">inventory_2</span>
            <div class="text-space-400 text-lg">Cargo hold is empty</div>
            <div class="text-xs text-space-500 mt-2">Use transfer or loading operations to add cargo</div>
          </div>
        </div>
      `;
    }

    const cargoItemsHtml = cargoData.map(cargo => this.renderDetailedCargoItem(cargo)).join('');

    return `
      <div class="cargo-details mb-4 p-4 bg-space-700 rounded-lg border border-space-600">
        <h3 class="text-lg font-semibold mb-3 text-orange-200 flex items-center gap-2">
          <span class="material-icons text-sm">inventory</span>
          Cargo Hold Contents (${cargoData.length} types)
        </h3>
        
        <div class="cargo-items space-y-3">
          ${cargoItemsHtml}
        </div>
      </div>
    `;
  }

  renderDetailedCargoItem(cargo) {
    const resourceDef = this.uiController.getResourceDefinition(cargo.resource_name || 'unknown');
    const density = this.getResourceDensity(cargo.resource_name);
    const value = this.getResourceValue(cargo.resource_name);
    const totalValue = value * cargo.quantity;

    return `
      <div class="cargo-item p-4 bg-space-800 rounded-lg border border-space-600 hover:border-space-500 transition-colors">
        <div class="flex items-start justify-between">
          <div class="flex items-center gap-4">
            <div class="resource-icon p-3 bg-space-700 rounded-lg">
              <span class="material-icons text-2xl ${resourceDef.color}">${resourceDef.icon}</span>
            </div>
            <div class="flex-1">
              <div class="flex items-center gap-2 mb-1">
                <h4 class="text-lg font-semibold text-white">${cargo.resource_name || 'Unknown'}</h4>
                <span class="px-2 py-1 bg-space-600 rounded text-xs text-space-300">${cargo.resource_type.slice(-4)}</span>
              </div>
              <div class="grid grid-cols-2 gap-4 text-sm">
                <div>
                  <span class="text-space-400">Quantity:</span>
                  <span class="text-white font-medium ml-2">${cargo.quantity} units</span>
                </div>
                <div>
                  <span class="text-space-400">Density:</span>
                  <span class="text-white font-medium ml-2">${density} kg/unit</span>
                </div>
                <div>
                  <span class="text-space-400">Unit Value:</span>
                  <span class="text-green-400 font-medium ml-2">${value} credits</span>
                </div>
                <div>
                  <span class="text-space-400">Total Value:</span>
                  <span class="text-green-400 font-bold ml-2">${totalValue} credits</span>
                </div>
              </div>
            </div>
          </div>
          <div class="flex flex-col gap-2">
            <button class="btn btn-sm btn-info flex items-center gap-1" 
                    onclick="window.fleetComponents.transferCargoType('${this.ship.id}', '${cargo.resource_type}')"
                    ${this.isFleetMoving() ? 'disabled' : ''}>
              <span class="material-icons text-xs">swap_horiz</span>
              Transfer
            </button>
            <button class="btn btn-sm btn-warning flex items-center gap-1"
                    onclick="window.fleetComponents.jettison('${this.ship.id}', '${cargo.resource_type}')"
                    ${this.isFleetMoving() ? 'disabled' : ''}>
              <span class="material-icons text-xs">launch</span>
              Jettison
            </button>
          </div>
        </div>
      </div>
    `;
  }

  renderCargoActions() {
    const isFleetMoving = this.isFleetMoving();
    const shipType = this.getShipTypeData();
    const cargoCapacity = shipType?.cargo_capacity || 0;

    if (cargoCapacity === 0) {
      return `
        <div class="cargo-actions p-4 bg-space-700 rounded-lg border border-space-600">
          <h3 class="text-lg font-semibold mb-3 text-orange-200 flex items-center gap-2">
            <span class="material-icons text-sm">build</span>
            Ship Actions
          </h3>
          
          <div class="text-center py-4">
            <div class="text-space-400 mb-4">No cargo operations available for this ship type</div>
            <button class="btn btn-secondary flex items-center justify-center gap-2 mx-auto"
                    onclick="window.fleetComponents.backToShip('${this.ship.id}')">
              <span class="material-icons text-sm">arrow_back</span>
              Back to Ship Details
            </button>
          </div>
        </div>
      `;
    }

    return `
      <div class="cargo-actions p-4 bg-space-700 rounded-lg border border-space-600">
        <h3 class="text-lg font-semibold mb-3 text-orange-200 flex items-center gap-2">
          <span class="material-icons text-sm">build</span>
          Cargo Operations
        </h3>
        
        <div class="grid grid-cols-2 gap-3">
          <button class="btn btn-primary flex items-center justify-center gap-2"
                  onclick="window.fleetComponents.loadCargo('${this.ship.id}')"
                  ${isFleetMoving ? 'disabled' : ''}>
            <span class="material-icons text-sm">download</span>
            Load Cargo
          </button>
          
          <button class="btn btn-secondary flex items-center justify-center gap-2"
                  onclick="window.fleetComponents.unloadCargo('${this.ship.id}')"
                  ${isFleetMoving ? 'disabled' : ''}>
            <span class="material-icons text-sm">upload</span>
            Unload Cargo
          </button>
          
          <button class="btn btn-info flex items-center justify-center gap-2"
                  onclick="window.fleetComponents.transferAllCargo('${this.ship.id}')"
                  ${isFleetMoving ? 'disabled' : ''}>
            <span class="material-icons text-sm">compare_arrows</span>
            Transfer All
          </button>
          
          <button class="btn btn-warning flex items-center justify-center gap-2"
                  onclick="window.fleetComponents.jettisonAll('${this.ship.id}')"
                  ${isFleetMoving ? 'disabled' : ''}>
            <span class="material-icons text-sm">delete_sweep</span>
            Jettison All
          </button>
        </div>

        <div class="mt-4 pt-3 border-t border-space-600">
          <button class="btn btn-secondary flex items-center justify-center gap-2 w-full"
                  onclick="window.fleetComponents.backToShip('${this.ship.id}')">
            <span class="material-icons text-sm">arrow_back</span>
            Back to Ship Details
          </button>
        </div>
      </div>
    `;
  }

  getShipTypeData() {
    const shipTypes = this.gameState.shipTypes || [];
    return shipTypes.find(st => st.id === this.ship.ship_type || st.name === this.ship.ship_type_name);
  }

  getShipCargo() {
    // Get cargo for this specific ship from the game state
    const allCargo = this.gameState.shipCargo || [];
    return allCargo.filter(cargo => cargo.ship_id === this.ship.id);
  }

  getResourceDensity(resourceName) {
    // Mock data - would come from game state resource definitions
    const densities = {
      'ore': 2.5,
      'metal': 7.8,
      'fuel': 0.8,
      'food': 1.2,
      'water': 1.0,
      'machinery': 3.2,
      'electronics': 1.5
    };
    return densities[resourceName?.toLowerCase()] || 1.0;
  }

  getResourceValue(resourceName) {
    // Mock data - would come from game state market prices
    const values = {
      'ore': 10,
      'metal': 25,
      'fuel': 15,
      'food': 8,
      'water': 5,
      'machinery': 100,
      'electronics': 200
    };
    return values[resourceName?.toLowerCase()] || 1;
  }

  isFleetMoving() {
    const allFleetOrders = this.gameState.fleetOrders || [];
    return allFleetOrders.some(order => 
      order.fleet_id === this.fleet.id && 
      (order.status === "pending" || order.status === "processing")
    );
  }
}