export class ShipComponent {
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
    const fleetName = this.fleet.name || `Fleet ${this.fleet.id.slice(-4)}`;

    return `
      <div class="ship-details-container">
        ${this.renderShipHeader(shipTypeName, shipCount, fleetName)}
        ${this.renderShipStats()}
        ${this.renderShipCargo()}
        ${this.renderShipActions()}
      </div>
    `;
  }

  renderShipHeader(shipTypeName, shipCount, fleetName) {
    const healthPercent = this.ship.health || 100;
    const healthColor =
      healthPercent > 75
        ? "text-green-400"
        : healthPercent > 50
          ? "text-yellow-400"
          : "text-red-400";
    const healthBgColor =
      healthPercent > 75
        ? "bg-green-500"
        : healthPercent > 50
          ? "bg-yellow-500"
          : "bg-red-500";

    return `
      <div class="ship-header mb-4 p-4 bg-gradient-to-r from-space-800 to-space-700 rounded-lg border border-space-600">
        <div class="flex items-start justify-between">
          <div class="flex items-center gap-3">
            <span class="material-icons text-2xl text-cyan-400">rocket_launch</span>
            <div>
              <h2 class="text-xl font-bold text-white">${shipCount}x ${shipTypeName}</h2>
              <div class="text-sm text-space-300">Ship ID: ${this.ship.id}</div>
              <div class="text-sm text-space-300">Fleet: ${fleetName}</div>
            </div>
          </div>
          <div class="text-right">
            <div class="text-xs text-space-400">Hull Integrity</div>
            <div class="text-lg font-bold ${healthColor}">${healthPercent}%</div>
          </div>
        </div>

        <div class="mt-3">
          <div class="flex justify-between text-xs mb-1">
            <span class="text-space-400">Health Status</span>
            <span class="text-white">${this.getHealthStatus(healthPercent)}</span>
          </div>
          <div class="w-full bg-space-600 rounded-full h-3">
            <div class="${healthBgColor} h-3 rounded-full transition-all duration-300"
                 style="width: ${healthPercent}%"></div>
          </div>
        </div>
      </div>
    `;
  }

  renderShipStats() {
    const shipType = this.getShipTypeData();
    const cargoCapacity = shipType?.cargo_capacity || 0;
    const strength = shipType?.strength || 0;

    return `
      <div class="ship-stats mb-4 p-4 bg-space-700 rounded-lg border border-space-600">
        <h3 class="text-lg font-semibold mb-3 text-cyan-200 flex items-center gap-2">
          <span class="material-icons text-sm">assessment</span>
          Ship Specifications
        </h3>

        <div class="grid grid-cols-2 gap-4">
          <div class="stat-item p-3 bg-space-800 rounded">
            <div class="flex items-center gap-2 mb-2">
              <span class="material-icons text-sm text-blue-400">inventory</span>
              <span class="text-space-400">Cargo Capacity</span>
            </div>
            <div class="text-xl font-bold text-blue-400">${cargoCapacity}</div>
            <div class="text-xs text-space-500">units per ship</div>
          </div>

          <div class="stat-item p-3 bg-space-800 rounded">
            <div class="flex items-center gap-2 mb-2">
              <span class="material-icons text-sm text-red-400">military_tech</span>
              <span class="text-space-400">Combat Strength</span>
            </div>
            <div class="text-xl font-bold text-red-400">${strength}</div>
            <div class="text-xs text-space-500">per ship</div>
          </div>

          <div class="stat-item p-3 bg-space-800 rounded">
            <div class="flex items-center gap-2 mb-2">
              <span class="material-icons text-sm text-green-400">groups</span>
              <span class="text-space-400">Ship Count</span>
            </div>
            <div class="text-xl font-bold text-green-400">${this.ship.count || 1}</div>
            <div class="text-xs text-space-500">in formation</div>
          </div>

          <div class="stat-item p-3 bg-space-800 rounded">
            <div class="flex items-center gap-2 mb-2">
              <span class="material-icons text-sm text-purple-400">calculate</span>
              <span class="text-space-400">Total Strength</span>
            </div>
            <div class="text-xl font-bold text-purple-400">${strength * (this.ship.count || 1)}</div>
            <div class="text-xs text-space-500">combined</div>
          </div>
        </div>
      </div>
    `;
  }

  renderShipCargo() {
    const cargoData = this.getShipCargo();
    const cargoCapacity = this.getShipTypeData()?.cargo_capacity || 0;
    const totalCapacity = cargoCapacity * (this.ship.count || 1);

    if (totalCapacity === 0) {
      return `
        <div class="ship-cargo mb-4 p-4 bg-space-700 rounded-lg border border-space-600">
          <h3 class="text-lg font-semibold mb-3 text-orange-200 flex items-center gap-2">
            <span class="material-icons text-sm">inventory_2</span>
            Cargo Hold
          </h3>
          <div class="text-space-400 text-center py-4">This ship type cannot carry cargo</div>
        </div>
      `;
    }

    const usedCapacity = cargoData.reduce(
      (sum, cargo) => sum + cargo.quantity,
      0,
    );
    const usagePercent =
      totalCapacity > 0 ? Math.round((usedCapacity / totalCapacity) * 100) : 0;

    const cargoItemsHtml =
      cargoData.length > 0
        ? cargoData
            .slice(0, 3)
            .map((cargo) => this.renderCargoItem(cargo))
            .join("")
        : '<div class="text-space-400 text-center py-4">Cargo hold is empty</div>';

    const hasMoreCargo = cargoData.length > 3;

    return `
      <div class="ship-cargo mb-4 p-4 bg-space-700 rounded-lg border border-space-600">
        <div class="flex items-center justify-between mb-3">
          <h3 class="text-lg font-semibold text-orange-200 flex items-center gap-2">
            <span class="material-icons text-sm">inventory_2</span>
            Cargo Hold
          </h3>
          <button class="btn btn-sm btn-info flex items-center gap-1"
                  onclick="window.fleetComponents.showShipCargo('${this.ship.id}')">
            <span class="material-icons text-xs">open_in_new</span>
            Detailed View
          </button>
        </div>

        <div class="cargo-capacity mb-4 p-3 bg-space-800 rounded">
          <div class="flex justify-between text-sm mb-2">
            <span class="text-space-400">Capacity Usage</span>
            <span class="text-white">${usedCapacity} / ${totalCapacity} units</span>
          </div>
          <div class="w-full bg-space-600 rounded-full h-3">
            <div class="bg-gradient-to-r from-orange-500 to-yellow-500 h-3 rounded-full transition-all duration-300"
                 style="width: ${usagePercent}%"></div>
          </div>
          <div class="text-center mt-1">
            <span class="text-xs ${usagePercent > 90 ? "text-red-400" : usagePercent > 75 ? "text-yellow-400" : "text-green-400"}">
              ${usagePercent}% Full
            </span>
          </div>
        </div>

        <div class="cargo-items space-y-2">
          ${cargoItemsHtml}
          ${
            hasMoreCargo
              ? `
            <div class="text-center py-2">
              <button class="btn btn-sm btn-secondary" onclick="window.fleetComponents.showShipCargo('${this.ship.id}')">
                View ${cargoData.length - 3} more cargo types...
              </button>
            </div>
          `
              : ""
          }
        </div>
      </div>
    `;
  }

  renderCargoItem(cargo) {
    const resourceDef = this.uiController.getResourceDefinition(
      cargo.resource_name || "unknown",
    );

    return `
      <div class="cargo-item p-3 bg-space-800 rounded border border-space-600 flex items-center justify-between">
        <div class="flex items-center gap-3">
          <span class="material-icons text-xl ${resourceDef.color}">${resourceDef.icon}</span>
          <div>
            <div class="font-medium text-white">${cargo.resource_name || "Unknown"}</div>
            <div class="text-xs text-space-300">Resource ID: ${cargo.resource_type}</div>
          </div>
        </div>
        <div class="text-right">
          <div class="text-lg font-bold text-white">${cargo.quantity}</div>
          <div class="text-xs text-space-400">units</div>
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

  getHealthStatus(healthPercent) {
    if (healthPercent >= 90) return "Excellent";
    if (healthPercent >= 75) return "Good";
    if (healthPercent >= 50) return "Damaged";
    if (healthPercent >= 25) return "Heavily Damaged";
    return "Critical";
  }

  getShipTypeData() {
    const shipTypes = this.gameState.shipTypes || [];
    return shipTypes.find(
      (st) =>
        st.id === this.ship.ship_type || st.name === this.ship.ship_type_name,
    );
  }

  getShipCargo() {
    // Get cargo for this specific ship from the game state
    const allCargo = this.gameState.shipCargo || [];
    return allCargo.filter((cargo) => cargo.ship_id === this.ship.id);
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
