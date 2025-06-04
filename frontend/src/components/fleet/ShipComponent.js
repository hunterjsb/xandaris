import { gameData } from '../../lib/pocketbase.js';

export class ShipComponent {
  constructor(uiController, gameState) {
    this.uiController = uiController;
    this.gameState = gameState;
    this.ship = null;
    this.fleet = null;
  }

  setShip(shipId) {
    // Find the ship across all fleets
    const allFleets = this.gameState.fleets || [];
    for (const fleet of allFleets) {
      if (fleet.ships) {
        const ship = fleet.ships.find((s) => s.id === shipId);
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

    const shipTypeName = this.ship.ship_type_name || "Unknown Ship";
    const shipCount = this.ship.count || 1;

    // Render with loading state, then load cargo asynchronously
    const containerId = `ship-cargo-${shipId}`;
    setTimeout(() => this.loadCargoData(shipId, containerId), 100);

    return `
      <div class="ship-details-container">
        <div id="${containerId}">
          ${this.renderLoadingCargo(shipTypeName, shipCount)}
        </div>
        ${this.renderShipActions()}
      </div>
    `;
  }



  renderLoadingCargo(shipTypeName, shipCount) {
    const fleetName = this.fleet.name || `Fleet ${this.fleet.id.slice(-4)}`;
    
    return `
      <div class="ship-cargo mb-4 p-4 bg-space-700 rounded-lg border border-space-600">
        <div class="mb-4">
          <h2 class="text-xl font-bold text-white mb-1">${shipCount}x ${shipTypeName}</h2>
          <div class="text-sm text-space-300">Fleet: ${fleetName}</div>
        </div>

        <div class="mb-4 p-3 bg-space-800 rounded">
          <div class="flex justify-between text-sm mb-2">
            <span class="text-space-400">Cargo Capacity</span>
            <span class="text-white">Loading...</span>
          </div>
          <div class="w-full bg-space-600 rounded-full h-2">
            <div class="bg-gradient-to-r from-space-500 to-space-400 h-2 rounded-full animate-pulse"></div>
          </div>
        </div>

        <h3 class="text-lg font-semibold mb-3 text-orange-200">Cargo</h3>
        <div class="text-space-400 text-center py-4">
          <span class="material-icons animate-spin">refresh</span>
          Loading cargo data...
        </div>
      </div>
    `;
  }

  async loadCargoData(shipId, containerId) {
    try {
      const shipCargo = await gameData.getIndividualShipCargo(shipId);
      const cargoHtml = this.renderCargoData(shipCargo);
      
      const container = document.getElementById(containerId);
      if (container) {
        container.innerHTML = cargoHtml;
      }
    } catch (error) {
      console.error('Failed to load ship cargo:', error);
      const container = document.getElementById(containerId);
      if (container) {
        container.innerHTML = this.renderCargoError();
      }
    }
  }

  renderCargoData(shipCargo) {
    const shipTypeName = this.ship.ship_type_name || "Unknown Ship";
    const shipCount = this.ship.count || 1;
    const fleetName = this.fleet.name || `Fleet ${this.fleet.id.slice(-4)}`;
    
    const cargoEntries = Object.entries(shipCargo.cargo || {}).filter(([resource, quantity]) => quantity > 0);
    const usagePercent = shipCargo.total_capacity > 0 ? Math.round((shipCargo.used_capacity / shipCargo.total_capacity) * 100) : 0;

    const cargoItemsHtml = cargoEntries.length > 0 
      ? cargoEntries.map(([resource, quantity]) => {
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
              <button class="btn btn-sm btn-warning" onclick="window.fleetComponents.showShipCargo('${this.ship.id}')">
                Manage
              </button>
            </div>
          `;
        }).join('')
      : '<div class="text-space-400 text-center py-4">No cargo loaded</div>';

    return `
      <div class="ship-cargo mb-4 p-4 bg-space-700 rounded-lg border border-space-600">
        <div class="mb-4">
          <h2 class="text-xl font-bold text-white mb-1">${shipCount}x ${shipTypeName}</h2>
          <div class="text-sm text-space-300">Fleet: ${fleetName}</div>
        </div>

        <div class="mb-4 p-3 bg-space-800 rounded">
          <div class="flex justify-between text-sm mb-2">
            <span class="text-space-400">Cargo Capacity</span>
            <span class="text-white">${shipCargo.used_capacity} / ${shipCargo.total_capacity} units</span>
          </div>
          <div class="w-full bg-space-600 rounded-full h-2">
            <div class="bg-gradient-to-r from-orange-500 to-yellow-500 h-2 rounded-full transition-all duration-300"
                 style="width: ${usagePercent}%"></div>
          </div>
        </div>

        <h3 class="text-lg font-semibold mb-3 text-orange-200">Cargo (${cargoEntries.length} types)</h3>
        <div class="space-y-2">
          ${cargoItemsHtml}
        </div>
      </div>
    `;
  }

  renderCargoError() {
    const shipTypeName = this.ship.ship_type_name || "Unknown Ship";
    const shipCount = this.ship.count || 1;
    const fleetName = this.fleet.name || `Fleet ${this.fleet.id.slice(-4)}`;
    
    return `
      <div class="ship-cargo mb-4 p-4 bg-space-700 rounded-lg border border-space-600">
        <div class="mb-4">
          <h2 class="text-xl font-bold text-white mb-1">${shipCount}x ${shipTypeName}</h2>
          <div class="text-sm text-space-300">Fleet: ${fleetName}</div>
        </div>
        <div class="text-red-400 text-center py-4">
          <span class="material-icons">error</span>
          Failed to load cargo data
        </div>
      </div>
    `;
  }



  renderShipActions() {
    const isFleetMoving = this.isFleetMoving();
    const cargoCapacity = this.getShipTypeData()?.cargo_capacity || 0;

    return `
      <div class="ship-actions p-4 bg-space-700 rounded-lg border border-space-600">
        <h3 class="text-lg font-semibold mb-3 text-cyan-200 flex items-center gap-2">
          <span class="material-icons text-sm">settings</span>
          Ship Actions
        </h3>

        <div class="grid grid-cols-2 gap-3">
          ${
            cargoCapacity > 0
              ? `
            <button class="btn btn-primary flex items-center justify-center gap-2"
                    onclick="window.fleetComponents.showShipCargo('${this.ship.id}')"
                    ${isFleetMoving ? "disabled" : ""}>
              <span class="material-icons text-sm">inventory_2</span>
              Manage Cargo
            </button>
          `
              : `
            <button class="btn btn-secondary flex items-center justify-center gap-2" disabled>
              <span class="material-icons text-sm">block</span>
              No Cargo Bay
            </button>
          `
          }

          <button class="btn btn-secondary flex items-center justify-center gap-2"
                  onclick="window.fleetComponents.repairShip('${this.ship.id}')"
                  ${isFleetMoving ? "disabled" : ""}>
            <span class="material-icons text-sm">build</span>
            Repair Ship
          </button>

          <button class="btn btn-info flex items-center justify-center gap-2"
                  onclick="window.fleetComponents.upgradeShip('${this.ship.id}')"
                  ${isFleetMoving ? "disabled" : ""}>
            <span class="material-icons text-sm">upgrade</span>
            Upgrade Ship
          </button>

          <button class="btn btn-warning flex items-center justify-center gap-2"
                  onclick="window.fleetComponents.scuttleShip('${this.ship.id}')"
                  ${isFleetMoving ? "disabled" : ""}>
            <span class="material-icons text-sm">delete_forever</span>
            Scuttle Ship
          </button>
        </div>

        <div class="mt-3 pt-3 border-t border-space-600">
          <button class="btn btn-secondary flex items-center justify-center gap-2 w-full"
                  onclick="window.fleetComponents.backToFleet('${this.fleet.id}')">
            <span class="material-icons text-sm">arrow_back</span>
            Back to Fleet
          </button>
        </div>
      </div>
    `;
  }



  getShipTypeData() {
    const shipTypes = this.gameState.shipTypes || [];
    return shipTypes.find(
      (st) =>
        st.id === this.ship.ship_type || st.name === this.ship.ship_type_name,
    );
  }



  isFleetMoving() {
    const allFleetOrders = this.gameState.fleetOrders || [];
    return allFleetOrders.some(
      (order) =>
        order.fleet_id === this.fleet.id &&
        (order.status === "pending" || order.status === "processing"),
    );
  }
}
