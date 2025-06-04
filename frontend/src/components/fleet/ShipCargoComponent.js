export class ShipCargoComponent {
  constructor(uiController, gameState) {
    this.uiController = uiController;
    this.gameState = gameState;
    this.ship = null;
    this.fleet = null;
  }

  setShip(shipId) {
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
    const fleetCargo = this.gameState?.getFleetCargo(this.fleet.id) || { cargo: {}, used_capacity: 0, total_capacity: 0 };

    return `
      <div class="ship-cargo-container">
        ${this.renderHeader(shipTypeName, shipCount, fleetCargo)}
        ${this.renderCargo(fleetCargo)}
        ${this.renderActions()}
      </div>
    `;
  }

  renderHeader(shipTypeName, shipCount, fleetCargo) {
    const usagePercent = fleetCargo.total_capacity > 0 ? Math.round((fleetCargo.used_capacity / fleetCargo.total_capacity) * 100) : 0;
    
    return `
      <div class="mb-4 p-4 bg-space-800 rounded-lg border border-space-600">
        <div class="flex items-center justify-between mb-3">
          <div>
            <h2 class="text-xl font-bold text-white">${shipCount}x ${shipTypeName}</h2>
            <div class="text-sm text-space-300">Cargo Management</div>
          </div>
          <div class="text-right">
            <div class="text-lg font-bold text-white">${fleetCargo.used_capacity}/${fleetCargo.total_capacity}</div>
            <div class="text-xs text-space-400">Units Used</div>
          </div>
        </div>
        
        <div class="w-full bg-space-600 rounded-full h-2">
          <div class="bg-gradient-to-r from-orange-500 to-yellow-500 h-2 rounded-full transition-all duration-500" 
               style="width: ${usagePercent}%"></div>
        </div>
      </div>
    `;
  }

  renderCargo(fleetCargo) {
    const cargoEntries = Object.entries(fleetCargo.cargo || {}).filter(([resource, quantity]) => quantity > 0);
    
    if (cargoEntries.length === 0) {
      return `
        <div class="mb-4 p-4 bg-space-700 rounded-lg border border-space-600 text-center">
          <span class="material-icons text-4xl text-space-500 mb-2">inventory_2</span>
          <div class="text-space-400">No cargo loaded</div>
        </div>
      `;
    }

    const cargoItemsHtml = cargoEntries.map(([resource, quantity]) => {
      const resourceDef = this.uiController.getResourceDefinition(resource);
      return `
        <div class="flex items-center justify-between p-3 bg-space-800 rounded border border-space-600">
          <div class="flex items-center gap-3">
            <span class="material-icons ${resourceDef.color}">${resourceDef.icon}</span>
            <div>
              <div class="font-medium text-white">${resource}</div>
              <div class="text-sm text-space-300">${quantity} units</div>
            </div>
          </div>
          <button class="btn btn-sm btn-warning" onclick="window.fleetComponents.transferCargoType('${this.ship.id}', '${resource}')">
            Transfer
          </button>
        </div>
      `;
    }).join('');

    return `
      <div class="mb-4 p-4 bg-space-700 rounded-lg border border-space-600">
        <h3 class="text-lg font-semibold mb-3 text-orange-200">Cargo (${cargoEntries.length} types)</h3>
        <div class="space-y-2">
          ${cargoItemsHtml}
        </div>
      </div>
    `;
  }

  renderActions() {
    const isFleetMoving = this.isFleetMoving();
    
    return `
      <div class="p-4 bg-space-700 rounded-lg border border-space-600">
        <div class="grid grid-cols-2 gap-3">
          <button class="btn btn-primary" onclick="window.fleetComponents.loadCargo('${this.ship.id}')" ${isFleetMoving ? 'disabled' : ''}>
            <span class="material-icons text-sm mr-1">download</span>
            Load
          </button>
          <button class="btn btn-secondary" onclick="window.fleetComponents.unloadCargo('${this.ship.id}')" ${isFleetMoving ? 'disabled' : ''}>
            <span class="material-icons text-sm mr-1">upload</span>
            Unload
          </button>
        </div>
        <button class="btn btn-ghost w-full mt-3" onclick="window.fleetComponents.backToShip('${this.ship.id}')">
          <span class="material-icons text-sm mr-1">arrow_back</span>
          Back to Ship
        </button>
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
}