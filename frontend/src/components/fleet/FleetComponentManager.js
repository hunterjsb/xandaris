import { FleetListComponent } from './FleetListComponent.js';
import { FleetComponent } from './FleetComponent.js';
import { ShipComponent } from './ShipComponent.js';
import { ShipCargoComponent } from './ShipCargoComponent.js';
import { gameData } from '../../lib/pocketbase.js';

export class FleetComponentManager {
  constructor(uiController, gameState) {
    this.uiController = uiController;
    this.gameState = gameState;
    
    this.fleetListComponent = new FleetListComponent(uiController, gameState);
    this.fleetComponent = new FleetComponent(uiController, gameState);
    this.shipComponent = new ShipComponent(uiController, gameState);
    this.shipCargoComponent = new ShipCargoComponent(uiController, gameState);
    
    this.currentView = 'list'; // 'list', 'fleet', 'ship', 'cargo'
    this.currentFleetId = null;
    this.currentShipId = null;
    
    // Make manager globally accessible for onclick handlers
    window.fleetComponents = this;
  }

  // Main entry point - shows the fleet list
  showFleetPanel() {
    this.currentView = 'list';
    this.currentFleetId = null;
    this.currentShipId = null;
    
    this.uiController.showFleetPanelAsFloating();
  }

  // Show detailed view of a specific fleet
  showFleetDetails(fleetId) {
    this.currentView = 'fleet';
    this.currentFleetId = fleetId;
    this.currentShipId = null;
    
    this.uiController.showFleetDetailsAsFloating(fleetId);
  }

  // Show detailed view of a specific ship
  showShipDetails(shipId) {
    this.currentView = 'ship';
    this.currentShipId = shipId;
    
    this.uiController.showShipDetailsAsFloating(shipId);
  }

  // Show detailed cargo view of a specific ship
  showShipCargo(shipId) {
    this.currentView = 'cargo';
    this.currentShipId = shipId;
    
    this.uiController.showShipCargoAsFloating(shipId);
  }

  // Navigation methods
  back() {
    switch (this.currentView) {
      case 'cargo':
        if (this.currentShipId) {
          this.showShipDetails(this.currentShipId);
        } else {
          this.showFleetPanel();
        }
        break;
      case 'ship':
        if (this.currentFleetId) {
          this.showFleetDetails(this.currentFleetId);
        } else {
          this.showFleetPanel();
        }
        break;
      case 'fleet':
        this.showFleetPanel();
        break;
      default:
        this.uiController.hideModal();
        break;
    }
  }

  backToFleet(fleetId) {
    this.showFleetDetails(fleetId);
  }

  backToShip(shipId) {
    this.showShipDetails(shipId);
  }

  // Action handlers
  sendFleet(fleetId) {
    const fleet = this.gameState.fleets?.find(f => f.id === fleetId);
    if (fleet) {
      const systems = this.gameState.systems || [];
      const currentSystem = systems.find(s => s.id === fleet.current_system);
      if (currentSystem) {
        this.uiController.hideModal();
        this.uiController.showSendFleetModal(currentSystem);
      }
    }
  }

  manageFleet(fleetId) {
    // TODO: Implement fleet management (rename, split, merge, etc.)
    this.uiController.showToast('Fleet management coming soon!', 'info', 3000);
  }

  viewCargo(fleetId) {
    const fleet = this.gameState.fleets?.find(f => f.id === fleetId);
    if (fleet && fleet.ships && fleet.ships.length > 0) {
      // Show cargo view for first ship with cargo capacity
      const cargoShip = fleet.ships.find(ship => {
        const shipType = this.getShipTypeData(ship);
        return shipType?.cargo_capacity > 0;
      });
      
      if (cargoShip) {
        this.showShipCargo(cargoShip.id);
      } else {
        this.uiController.showToast('No cargo ships found in this fleet', 'info', 3000);
      }
    } else {
      this.uiController.showToast('Fleet has no ships', 'error', 3000);
    }
  }

  transferCargo(fleetId, systemId) {
    this.showCargoTransferModal(fleetId, systemId);
  }

  async showCargoTransferModal(fleetId, systemId) {
    const fleet = this.gameState.fleets?.find(f => f.id === fleetId);
    const system = this.gameState.systems?.find(s => s.id === systemId);
    
    if (!fleet || !system) {
      this.uiController.showToast('Fleet or system not found', 'error', 3000);
      return;
    }

    const fleetCargo = this.gameState?.getFleetCargo(fleetId) || { cargo: {}, used_capacity: 0, total_capacity: 0 };
    
    // Get building storage for this system
    let buildingStorage = { storage: {}, buildings: [] };
    try {
      console.log('Fetching building storage for system:', systemId);
      buildingStorage = await gameData.getBuildingStorage(systemId);
      console.log('Building storage response:', buildingStorage);
    } catch (error) {
      console.error('Failed to get building storage:', error);
      this.uiController.showToast('Failed to load building storage', 'error', 3000);
      // Continue with empty storage to allow unloading at least
    }
    
    // Build cargo items list
    const cargoItems = Object.entries(fleetCargo.cargo || {})
      .filter(([resource, quantity]) => quantity > 0)
      .map(([resource, quantity]) => ({ resource, quantity, type: 'fleet' }));
    
    // Build building storage items list (only show resources actually stored)
    const storageItems = Object.entries(buildingStorage.storage || {})
      .filter(([resource, quantity]) => quantity > 0)
      .map(([resource, quantity]) => ({ resource, quantity, type: 'building' }));
    
    console.log('Storage items for UI:', storageItems);
    console.log('Building details:', buildingStorage.buildings);

    const content = this.renderCargoTransferModal(fleet, system, cargoItems, storageItems, fleetCargo, buildingStorage);
    
    this.uiController.showModal(
      `Cargo Transfer: ${fleet.name || `Fleet ${fleet.id.slice(-4)}`} ↔ ${system.name}`,
      content
    );
  }

  renderCargoTransferModal(fleet, system, cargoItems, storageItems, fleetCargo, buildingStorage) {
    const fleetName = fleet.name || `Fleet ${fleet.id.slice(-4)}`;
    const systemName = system.name || `System ${system.id.slice(-4)}`;
    
    const fleetItemsHtml = cargoItems.length > 0 ? 
      cargoItems.map(item => this.renderCargoItem(item, 'unload', fleet.id)).join('') :
      '<div class="text-space-400 text-center py-4">Fleet cargo is empty</div>';
    
    const storageItemsHtml = storageItems.length > 0 ? 
      storageItems.map(item => this.renderCargoItem(item, 'load', fleet.id)).join('') :
      '<div class="text-space-400 text-center py-4">No resources stored in buildings<br><small>Build storage buildings and produce resources first</small></div>';

    const buildingDetailsHtml = buildingStorage.buildings?.length > 0 ?
      buildingStorage.buildings.map(building => 
        `<div class="text-xs text-space-400 mb-1">
          ${building.name} (L${building.level}) on ${building.planet_name}
          ${building.res1_type ? `- ${building.res1_stored}/${building.res1_capacity} ${building.res1_type}` : ''}
          ${building.res2_type ? `- ${building.res2_stored}/${building.res2_capacity} ${building.res2_type}` : ''}
        </div>`
      ).join('') : '';

    return `
      <div class="cargo-transfer-modal space-y-4">
        <div class="flex justify-between items-center mb-4">
          <div class="text-sm text-space-300">
            <div><strong>Fleet:</strong> ${fleetName}</div>
            <div><strong>System:</strong> ${systemName}</div>
          </div>
          <div class="text-right text-sm">
            <div class="text-space-400">Fleet Capacity</div>
            <div class="text-white">${fleetCargo.used_capacity}/${fleetCargo.total_capacity}</div>
          </div>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <!-- Fleet Cargo (Unload) -->
          <div class="cargo-section">
            <h3 class="text-lg font-semibold mb-3 text-plasma-200 flex items-center gap-2">
              <span class="material-icons text-sm">rocket_launch</span>
              Fleet Cargo
              <span class="text-xs text-space-400 ml-auto">Click to unload</span>
            </h3>
            <div class="space-y-2 max-h-64 overflow-y-auto custom-scrollbar">
              ${fleetItemsHtml}
            </div>
          </div>

          <!-- Building Storage (Load) -->
          <div class="cargo-section">
            <h3 class="text-lg font-semibold mb-3 text-nebula-200 flex items-center gap-2">
              <span class="material-icons text-sm">warehouse</span>
              Building Storage
              <span class="text-xs text-space-400 ml-auto">Click to load</span>
            </h3>
            <div class="space-y-2 max-h-64 overflow-y-auto custom-scrollbar">
              ${storageItemsHtml}
            </div>
          </div>
        </div>

        ${buildingDetailsHtml ? `
        <div class="mt-4 pt-4 border-t border-space-600">
          <div class="text-sm text-space-300 mb-2">Storage Buildings:</div>
          ${buildingDetailsHtml}
        </div>
        ` : ''}

        <div class="mt-6 pt-4 border-t border-space-600">
          <div class="text-xs text-space-400 mb-3">
            <span class="material-icons text-xs">info</span>
            Transfer resources between fleet cargo and building storage. Buildings must have available capacity to store unloaded cargo.<br>
            <strong>Debug:</strong> System ID: ${system.id} | Fleet ID: ${fleet.id.slice(-4)} | Buildings found: ${buildingStorage.buildings?.length || 0}
          </div>
          <button class="w-full btn btn-secondary" onclick="window.uiController.hideModal()">
            Close
          </button>
        </div>
      </div>
    `;
  }

  renderCargoItem(item, direction, fleetId) {
    const resourceDef = this.uiController.getResourceDefinition(item.resource);
    const isUnload = direction === 'unload';
    const buttonColor = isUnload ? 'btn-warning' : 'btn-info';
    const buttonIcon = isUnload ? 'upload' : 'download';
    const buttonText = isUnload ? 'Unload' : 'Load';
    
    return `
      <div class="cargo-item p-3 bg-space-800 rounded border border-space-600 hover:border-space-500 transition-colors">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3">
            <span class="material-icons text-lg ${resourceDef.color}">${resourceDef.icon}</span>
            <div>
              <div class="font-medium text-white">${item.resource}</div>
              <div class="text-sm text-space-300">${item.quantity} units</div>
            </div>
          </div>
          <button class="btn btn-sm ${buttonColor} flex items-center gap-1"
                  onclick="window.fleetComponents.performCargoTransfer('${fleetId}', '${item.resource}', ${item.quantity}, '${direction}')">
            <span class="material-icons text-xs">${buttonIcon}</span>
            ${buttonText}
          </button>
        </div>
      </div>
    `;
  }

  async performCargoTransfer(fleetId, resourceType, maxQuantity, direction) {
    try {
      // Calculate smart transfer quantity based on fleet capacity
      let quantity = maxQuantity;
      
      if (direction === 'load') {
        // When loading, check fleet capacity
        const fleetCargo = this.gameState?.getFleetCargo(fleetId) || { used_capacity: 0, total_capacity: 0 };
        const availableSpace = fleetCargo.total_capacity - fleetCargo.used_capacity;
        
        if (availableSpace <= 0) {
          this.uiController.showToast('Fleet cargo is full', 'error', 3000);
          return;
        }
        
        // Transfer only what fits
        quantity = Math.min(maxQuantity, availableSpace);
        
        if (quantity < maxQuantity) {
          this.uiController.showToast(`Loading ${quantity} of ${maxQuantity} ${resourceType} (fleet capacity limit)`, 'warning', 4000);
        }
      }
      
      const result = await gameData.transferCargo(fleetId, resourceType, quantity, direction);
      
      if (result.success) {
        this.uiController.showToast(result.message, 'success', 3000);
        
        // Refresh game state to show updated cargo
        await this.gameState.lightweightTickUpdate();
        
        // Close modal and refresh current view
        this.uiController.hideModal();
        this.refresh();
      } else {
        this.uiController.showToast('Transfer failed', 'error', 3000);
      }
    } catch (error) {
      console.error('Cargo transfer failed:', error);
      this.uiController.showToast(`Transfer failed: ${error.message}`, 'error', 5000);
    }
  }

  transferCargoShip(shipId) {
    // Navigate to detailed cargo view for transfers
    this.showShipCargo(shipId);
  }

  async spawnStarterShip() {
    return await this.fleetListComponent.spawnStarterShip();
  }

  repairShip(shipId) {
    // TODO: Implement ship repair functionality
    this.uiController.showToast('Ship repair coming soon!', 'info', 3000);
  }

  scuttleShip(shipId) {
    const ship = this.findShip(shipId);
    if (ship) {
      const shipName = `${ship.count || 1}x ${ship.ship_type_name || 'Unknown'}`;
      const confirmed = confirm(`Are you sure you want to scuttle ${shipName}? This action cannot be undone.`);
      if (confirmed) {
        // TODO: Implement ship scuttling
        this.uiController.showToast('Ship scuttling coming soon!', 'info', 3000);
      }
    }
  }

  // Refresh current view with updated data
  refresh() {
    switch (this.currentView) {
      case 'list':
        this.showFleetPanel();
        break;
      case 'fleet':
        if (this.currentFleetId) {
          this.showFleetDetails(this.currentFleetId);
        }
        break;
      case 'ship':
        if (this.currentShipId) {
          this.showShipDetails(this.currentShipId);
        }
        break;
      case 'cargo':
        if (this.currentShipId) {
          this.showShipCargo(this.currentShipId);
        }
        break;
    }
  }

  // Helper methods
  getFleetCount() {
    const playerFleets = this.gameState.getPlayerFleets?.() || [];
    const fleetCount = playerFleets.length;
    const shipCount = playerFleets.reduce((sum, fleet) => {
      return sum + (fleet.ships ? fleet.ships.reduce((shipSum, ship) => shipSum + (ship.count || 1), 0) : 0);
    }, 0);
    
    return `Your Fleets (${fleetCount} fleets, ${shipCount} ships)`;
  }

  findShip(shipId) {
    const allFleets = this.gameState.fleets || [];
    for (const fleet of allFleets) {
      if (fleet.ships) {
        const ship = fleet.ships.find(s => s.id === shipId);
        if (ship) {
          return ship;
        }
      }
    }
    return null;
  }

  // New cargo operation methods
  loadCargo(shipId) {
    this.uiController.showToast('Load cargo functionality coming soon!', 'info', 3000);
  }

  unloadCargo(shipId) {
    this.uiController.showToast('Unload cargo functionality coming soon!', 'info', 3000);
  }

  transferAllCargo(shipId) {
    this.uiController.showToast('Transfer all cargo functionality coming soon!', 'info', 3000);
  }

  transferCargoType(shipId, resourceType) {
    this.uiController.showToast('Transfer specific cargo type functionality coming soon!', 'info', 3000);
  }

  jettison(shipId, resourceType) {
    const ship = this.findShip(shipId);
    if (ship) {
      const confirmed = confirm(`Are you sure you want to jettison this cargo? This action cannot be undone.`);
      if (confirmed) {
        this.uiController.showToast('Jettison cargo functionality coming soon!', 'info', 3000);
      }
    }
  }

  jettisonAll(shipId) {
    const ship = this.findShip(shipId);
    if (ship) {
      const confirmed = confirm(`Are you sure you want to jettison ALL cargo? This action cannot be undone.`);
      if (confirmed) {
        this.uiController.showToast('Jettison all cargo functionality coming soon!', 'info', 3000);
      }
    }
  }

  upgradeShip(shipId) {
    this.uiController.showToast('Ship upgrade functionality coming soon!', 'info', 3000);
  }

  colonize(systemId) {
    // Use existing colonization modal
    const system = this.gameState.systems?.find(s => s.id === systemId);
    if (system) {
      this.uiController.showColonizeModal(system);
    } else {
      this.uiController.showToast('System not found', 'error', 3000);
    }
  }

  getShipTypeData(ship) {
    const shipTypes = this.gameState.shipTypes || [];
    return shipTypes.find(st => st.id === ship.ship_type || st.name === ship.ship_type_name);
  }

  // Update game state reference (called when game state changes)
  updateGameState(gameState) {
    this.gameState = gameState;
    this.fleetListComponent.gameState = gameState;
    this.fleetComponent.gameState = gameState;
    this.shipComponent.gameState = gameState;
    this.shipCargoComponent.gameState = gameState;
  }
}