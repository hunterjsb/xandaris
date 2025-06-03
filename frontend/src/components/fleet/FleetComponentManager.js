import { FleetListComponent } from './FleetListComponent.js';
import { FleetComponent } from './FleetComponent.js';
import { ShipComponent } from './ShipComponent.js';
import { ShipCargoComponent } from './ShipCargoComponent.js';

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
    
    const content = this.fleetListComponent.render();
    const title = this.getFleetCount();
    
    this.uiController.showModal(title, content);
  }

  // Show detailed view of a specific fleet
  showFleetDetails(fleetId) {
    this.currentView = 'fleet';
    this.currentFleetId = fleetId;
    this.currentShipId = null;
    
    const content = this.fleetComponent.render(fleetId);
    const fleet = this.gameState.fleets?.find(f => f.id === fleetId);
    const fleetName = fleet ? (fleet.name || `Fleet ${fleet.id.slice(-4)}`) : 'Unknown Fleet';
    
    this.uiController.showModal(`Fleet Details: ${fleetName}`, content);
  }

  // Show detailed view of a specific ship
  showShipDetails(shipId) {
    this.currentView = 'ship';
    this.currentShipId = shipId;
    
    const content = this.shipComponent.render(shipId);
    const ship = this.findShip(shipId);
    const shipName = ship ? `${ship.count || 1}x ${ship.ship_type_name || 'Unknown'}` : 'Unknown Ship';
    
    this.uiController.showModal(`Ship Details: ${shipName}`, content);
  }

  // Show detailed cargo view of a specific ship
  showShipCargo(shipId) {
    this.currentView = 'cargo';
    this.currentShipId = shipId;
    
    const content = this.shipCargoComponent.render(shipId);
    const ship = this.findShip(shipId);
    const shipName = ship ? `${ship.count || 1}x ${ship.ship_type_name || 'Unknown'}` : 'Unknown Ship';
    
    this.uiController.showModal(`Cargo Management: ${shipName}`, content);
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

  transferCargo(shipId) {
    // Navigate to detailed cargo view for transfers
    this.showShipCargo(shipId);
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