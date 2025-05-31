// Main application entry point
import './styles.css';
import { authManager } from './lib/pocketbase.js';
import { gameState } from './stores/gameState.js';
import { MapRenderer } from './components/mapRenderer.js';
import { UIController } from './components/uiController.js';

class XanNationApp {
  constructor() {
    this.mapRenderer = null;
    this.uiController = null;
    
    this.init();
  }

  async init() {
    console.log('Initializing Xan Nation...');

    // Initialize UI controller
    this.uiController = new UIController();

    // Initialize map renderer
    this.mapRenderer = new MapRenderer('game-canvas');

    // Subscribe to auth changes
    authManager.subscribe((user) => {
      this.handleAuthChange(user);
    });

    // Subscribe to game state changes
    gameState.subscribe((state) => {
      this.handleGameStateChange(state);
    });

    // Set up event listeners
    this.setupEventListeners();

    console.log('Xan Nation initialized');
  }

  handleAuthChange(user) {
    this.uiController.updateAuthUI(user);
    
    if (user) {
      console.log('User logged in:', user.username);
    } else {
      console.log('User logged out');
    }
  }

  handleGameStateChange(state) {
    // Update map renderer with new data
    if (this.mapRenderer) {
      this.mapRenderer.setPlanets(state.planets); // Changed setSystems to setPlanets, state.systems to state.planets
      this.mapRenderer.setFleets(state.fleets);
      this.mapRenderer.setSelectedPlanet(state.selectedPlanet); // Changed setSelectedSystem to setSelectedPlanet, state.selectedSystem to state.selectedPlanet
      
      // Set lanes if available
      if (state.mapData && state.mapData.lanes) {
        this.mapRenderer.setLanes(state.mapData.lanes);
      }
      
      // If this is the first load, fit to planets
      if (state.planets.length > 0 && !this.mapRenderer.hasInitialFit) { // Changed state.systems to state.planets
        this.mapRenderer.fitToPlanets(); // Changed fitToSystems to fitToPlanets
        this.mapRenderer.hasInitialFit = true;
      }
    }

    // Update UI with new state
    this.uiController.updateGameUI(state);
  }

  setupEventListeners() {
    // Canvas events
    const canvas = document.getElementById('game-canvas');
    
    canvas.addEventListener('planetSelected', (e) => { // Changed systemSelected to planetSelected
      gameState.selectPlanet(e.detail.planet.id); // Changed selectSystem to selectPlanet, e.detail.system.id to e.detail.planet.id
    });

    // Context menu actions
    const contextMenu = document.getElementById('context-menu');
    contextMenu.addEventListener('click', (e) => {
      const action = e.target.dataset.action;
      const planetId = contextMenu.dataset.planetId; // Changed systemId to planetId
      
      if (action && planetId) { // Changed systemId to planetId
        this.handleContextMenuAction(action, planetId); // Changed systemId to planetId
        contextMenu.classList.add('hidden');
      }
    });

    // Hide tooltip when canvas loses focus
    canvas.addEventListener('mouseleave', () => {
      document.getElementById('tooltip').classList.add('hidden');
    });

    // Navigation buttons
    document.getElementById('fleet-btn').addEventListener('click', () => {
      this.uiController.showFleetPanel();
    });

    document.getElementById('trade-btn').addEventListener('click', () => {
      this.uiController.showTradePanel();
    });

    document.getElementById('diplo-btn').addEventListener('click', () => {
      this.uiController.showDiplomacyPanel();
    });

    document.getElementById('banking-btn').addEventListener('click', () => {
      this.uiController.showBankingPanel();
    });

    // Auth buttons
    document.getElementById('login-btn').addEventListener('click', () => {
      this.handleLogin();
    });

    document.getElementById('logout-btn').addEventListener('click', () => {
      this.handleLogout();
    });

    // Action buttons
    document.getElementById('build-btn').addEventListener('click', () => {
      this.handleBuildAction();
    });

    document.getElementById('send-fleet-btn').addEventListener('click', () => {
      this.handleSendFleetAction();
    });

    document.getElementById('trade-route-btn').addEventListener('click', () => {
      this.handleTradeRouteAction();
    });

    // Keyboard shortcuts
    document.addEventListener('keydown', (e) => {
      this.handleKeyboardInput(e);
    });

    // Modal handling
    const modalOverlay = document.getElementById('modal-overlay');
    modalOverlay.addEventListener('click', (e) => {
      if (e.target === modalOverlay) {
        this.uiController.hideModal();
      }
    });
  }

  async handleLogin() {
    try {
      await authManager.loginWithDiscord();
    } catch (error) {
      console.error('Login failed:', error);
      this.uiController.showError('Login failed. Please try again.');
    }
  }

  handleLogout() {
    authManager.logout();
  }

  handleContextMenuAction(action, planetId) { // Changed systemId to planetId
    const planet = gameState.planets.find(p => p.id === planetId); // Changed system to planet, s to p
    if (!planet) return;

    switch (action) {
      case 'view':
        gameState.selectPlanet(planetId); // Changed selectSystem to selectPlanet
        this.mapRenderer.centerOnPlanet(planetId); // Changed centerOnSystem to centerOnPlanet
        break;
      case 'fleet':
        this.uiController.showSendFleetModal(planet); // Changed system to planet
        break;
      case 'trade':
        this.uiController.showTradeRouteModal(planet); // Changed system to planet
        break;
    }
  }

  handleBuildAction() {
    const selectedPlanet = gameState.getSelectedPlanet(); // Changed getSelectedSystem to getSelectedPlanet
    if (!selectedPlanet) {
      this.uiController.showError('Please select a planet first'); // Changed system to planet
      return;
    }

    if (!authManager.isLoggedIn()) {
      this.uiController.showError('Please log in first');
      return;
    }

    this.uiController.showBuildModal(selectedPlanet); // Changed selectedSystem to selectedPlanet
  }

  handleSendFleetAction() {
    const selectedPlanet = gameState.getSelectedPlanet(); // Changed getSelectedSystem to getSelectedPlanet
    if (!selectedPlanet) {
      this.uiController.showError('Please select a planet first'); // Changed system to planet
      return;
    }

    if (!authManager.isLoggedIn()) {
      this.uiController.showError('Please log in first');
      return;
    }

    this.uiController.showSendFleetModal(selectedPlanet); // Changed selectedSystem to selectedPlanet
  }

  handleTradeRouteAction() {
    const selectedPlanet = gameState.getSelectedPlanet(); // Changed getSelectedSystem to getSelectedPlanet
    if (!selectedPlanet) {
      this.uiController.showError('Please select a planet first'); // Changed system to planet
      return;
    }

    if (!authManager.isLoggedIn()) {
      this.uiController.showError('Please log in first');
      return;
    }

    this.uiController.showTradeRouteModal(selectedPlanet); // Changed selectedSystem to selectedPlanet
  }

  handleKeyboardInput(e) {
    // Only handle keyboard shortcuts when not in input fields
    if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') {
      return;
    }

    switch (e.key.toLowerCase()) {
      case 'escape':
        this.uiController.hideModal();
        document.getElementById('context-menu').classList.add('hidden');
        break;
      case 'f':
        this.handleSendFleetAction();
        break;
      case 't':
        this.handleTradeRouteAction();
        break;
      case 'b':
        this.handleBuildAction();
        break;
      case 'c':
        if (gameState.getSelectedPlanet()) { // Changed getSelectedSystem to getSelectedPlanet
          this.mapRenderer.centerOnPlanet(gameState.getSelectedPlanet().id); // Changed centerOnSystem to centerOnPlanet
        }
        break;
      case 'h':
        this.mapRenderer.fitToPlanets(); // Changed fitToSystems to fitToPlanets
        break;
    }
  }
}

// Start the application
new XanNationApp();