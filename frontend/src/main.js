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
      console.log('User logged in:', user.id, user.username);
      if (this.mapRenderer) {
        this.mapRenderer.setCurrentPlayer(user.id);
      }
    } else {
      console.log('User logged out');
      if (this.mapRenderer) {
        this.mapRenderer.setCurrentPlayer(null);
        // Potentially clear ally/enemy lists too
        this.mapRenderer.setAllies([]);
        this.mapRenderer.setEnemies([]);
      }
    }
  }

  handleGameStateChange(state) {
    // Update map renderer with new data
    if (this.mapRenderer) {
      this.mapRenderer.setSystems(state.systems);
      this.mapRenderer.setFleets(state.fleets);
      this.mapRenderer.setSelectedSystem(state.selectedSystem);
      
      // Set lanes if available
      if (state.mapData && state.mapData.lanes) {
        this.mapRenderer.setLanes(state.mapData.lanes);
      }

      // TODO: Update ally/enemy IDs when diplomacy data is available in gameState
      // For now, this will be empty or based on mock data if we add it.
      // Example:
      // if (state.treaties && this.mapRenderer.currentPlayerId) {
      //   const allies = [];
      //   const enemies = []; // Logic to determine enemies might be more complex
      //   state.treaties.forEach(treaty => {
      //     if (treaty.status === 'active' && (treaty.type === 'alliance' || treaty.type === 'trade_pact')) {
      //       if (treaty.a_id === this.mapRenderer.currentPlayerId) allies.push(treaty.b_id);
      //       if (treaty.b_id === this.mapRenderer.currentPlayerId) allies.push(treaty.a_id);
      //     }
      //     // Add logic for enemies if applicable from treaties or other state
      //   });
      //   this.mapRenderer.setAllies([...new Set(allies)]); // Remove duplicates
      //   this.mapRenderer.setEnemies(enemies);
      // }


      // If this is the first load, fit to systems
      if (state.systems.length > 0 && !this.mapRenderer.hasInitialFit) {
        this.mapRenderer.fitToSystems();
        this.mapRenderer.hasInitialFit = true;
      }

      // Update current tick in map renderer
      if (typeof state.currentTick !== 'undefined') {
        this.mapRenderer.setCurrentTick(state.currentTick);
      }
    }

    // Update UI with new state
    this.uiController.updateGameUI(state);
  }

  setupEventListeners() {
    // Canvas events
    const canvas = document.getElementById('game-canvas');
    
    canvas.addEventListener('systemSelected', (e) => {
      gameState.selectSystem(e.detail.system.id);
    });

    // Context menu actions
    const contextMenu = document.getElementById('context-menu');
    contextMenu.addEventListener('click', (e) => {
      const action = e.target.dataset.action;
      const systemId = contextMenu.dataset.systemId;
      
      if (action && systemId) {
        this.handleContextMenuAction(action, systemId);
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

  handleContextMenuAction(action, systemId) {
    const system = gameState.systems.find(s => s.id === systemId);
    if (!system) return;

    switch (action) {
      case 'view':
        gameState.selectSystem(systemId);
        this.mapRenderer.centerOnSystem(systemId);
        break;
      case 'fleet':
        this.uiController.showSendFleetModal(system);
        break;
      case 'trade':
        this.uiController.showTradeRouteModal(system);
        break;
    }
  }

  handleBuildAction() {
    const selectedSystem = gameState.getSelectedSystem();
    if (!selectedSystem) {
      this.uiController.showError('Please select a system first');
      return;
    }

    if (!authManager.isLoggedIn()) {
      this.uiController.showError('Please log in first');
      return;
    }

    this.uiController.showBuildModal(selectedSystem);
  }

  handleSendFleetAction() {
    const selectedSystem = gameState.getSelectedSystem();
    if (!selectedSystem) {
      this.uiController.showError('Please select a system first');
      return;
    }

    if (!authManager.isLoggedIn()) {
      this.uiController.showError('Please log in first');
      return;
    }

    this.uiController.showSendFleetModal(selectedSystem);
  }

  handleTradeRouteAction() {
    const selectedSystem = gameState.getSelectedSystem();
    if (!selectedSystem) {
      this.uiController.showError('Please select a system first');
      return;
    }

    if (!authManager.isLoggedIn()) {
      this.uiController.showError('Please log in first');
      return;
    }

    this.uiController.showTradeRouteModal(selectedSystem);
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
        if (gameState.getSelectedSystem()) {
          this.mapRenderer.centerOnSystem(gameState.getSelectedSystem().id);
        }
        break;
      case 'h':
        this.mapRenderer.fitToSystems();
        break;
    }
  }
}

// Start the application
new XanNationApp();