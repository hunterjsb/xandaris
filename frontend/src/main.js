// Main application entry point
import "./styles.css";
import { authManager, gameData, pb } from "./lib/pocketbase.js";
import { gameState } from "./stores/gameState.js";
import { MapRenderer } from "./components/mapRenderer.js";
import { UIController } from "./components/uiController.js";

// Make gameState and uiController globally available
const uiController = new UIController();
window.gameState = gameState;
window.uiController = uiController;

class XanNationApp {
  constructor() {
    this.mapRenderer = null;
    // this.uiController = null; // Will be set from window global
    this.fleetRoutes = new Map(); // Store multi-hop fleet routes

    this.init();
  }

  async init() {
    console.log("Initializing Xan Nation...");

    // Initialize UI controller
    this.uiController = window.uiController; // Use global uiController
    this.uiController.setPocketBase(pb); // Set PocketBase instance

    // Initialize map renderer
    this.mapRenderer = new MapRenderer("game-canvas");

    // Set initial user ID if already logged in
    const currentUser = authManager.getUser();
    if (currentUser) {
      this.mapRenderer.setCurrentUserId(currentUser.id);
    }

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

    console.log("Xandaris initialized");
  }

  handleAuthChange(user) {
    this.uiController.updateAuthUI(user);

    // Set current user ID on map renderer for ownership checks
    if (this.mapRenderer) {
      this.mapRenderer.setCurrentUserId(user?.id || null);
    }

    if (user) {
      console.log("User logged in:", user.username);
    } else {
      console.log("User logged out");
    }
  }

  handleGameStateChange(state) {
    // Debug logging for fleet orders
    if (state.fleetOrders && state.fleetOrders.length > 0) {
      console.log(`Fleet orders updated: ${state.fleetOrders.length} orders`, state.fleetOrders);
    }
    
    // Update map renderer with new data
    if (this.mapRenderer) {
      this.mapRenderer.setSystems(state.systems);
      this.mapRenderer.setFleets(state.fleets);
      this.mapRenderer.setTrades(state.trades); // Added line
      this.mapRenderer.setHyperlanes(state.hyperlanes); // Add hyperlanes

      // Only update selected system if it changed
      if (this.mapRenderer.selectedSystem?.id !== state.selectedSystem?.id) {
        this.mapRenderer.setSelectedSystem(state.selectedSystem);
      }

      // Set lanes if available
      if (state.mapData && state.mapData.lanes) {
        this.mapRenderer.setLanes(state.mapData.lanes);
      }

      // Center on player's starting fleet system if specified
      if (state.centerOnFleetSystem && !this.mapRenderer.hasCenteredOnFleet) {
        this.mapRenderer.centerOnSystem(state.centerOnFleetSystem);
        this.mapRenderer.hasCenteredOnFleet = true;
        // Also zoom in a bit for better starting view
        this.mapRenderer.zoom = 0.8;
      }
      // If this is the first load and no fleet to center on, fit to systems
      else if (state.systems.length > 0 && !this.mapRenderer.hasInitialFit) {
        this.mapRenderer.fitToSystems();
        this.mapRenderer.hasInitialFit = true;
      }
    }

    // Update UI with new state
    this.uiController.updateGameUI(state);
  }

  setupEventListeners() {
    // Canvas events
    const canvas = document.getElementById("game-canvas");

    canvas.addEventListener("systemSelected", (e) => {
      const system = e.detail.system;
      const planets = e.detail.planets;
      const screenX = e.detail.screenX;
      const screenY = e.detail.screenY;

      // Only select if it's a different system
      if (
        !gameState.selectedSystem ||
        gameState.selectedSystem.id !== system.id
      ) {
        gameState.selectSystem(system.id);
      }

      // Update UI directly to avoid circular updates
      this.uiController.displaySystemView(system, planets, screenX, screenY);
    });

    // Fleet movement via shift+click
    canvas.addEventListener("fleetMoveRequested", (e) => {
      const fromFleet = e.detail.fromFleet;
      const toSystem = e.detail.toSystem;
      
      this.handleMultiMoveFleet(fromFleet, toSystem);
    });

    // Fleet selection
    canvas.addEventListener("fleetSelected", (e) => {
      const fleet = e.detail.fleet;
      this.displaySelectedFleetInfo(fleet);
    });

    // Context menu actions
    const contextMenu = document.getElementById("context-menu");
    contextMenu.addEventListener("click", (e) => {
      const action = e.target.dataset.action;
      const systemId = contextMenu.dataset.systemId;

      if (action && systemId) {
        this.handleContextMenuAction(action, systemId);
        contextMenu.classList.add("hidden");
      }
    });

    // Hide tooltip when canvas loses focus
    canvas.addEventListener("mouseleave", () => {
      document.getElementById("tooltip").classList.add("hidden");
    });

    // Navigation buttons
    document.getElementById("fleet-btn").addEventListener("click", () => {
      this.uiController.showFleetPanel();
    });

    document.getElementById("trade-btn").addEventListener("click", () => {
      this.uiController.showTradePanel();
    });

    document.getElementById("diplo-btn").addEventListener("click", () => {
      this.uiController.showDiplomacyPanel();
    });

    document.getElementById("buildings-btn").addEventListener("click", () => {
      this.uiController.showBuildingsPanel();
    });

    // Auth buttons
    document.getElementById("login-btn").addEventListener("click", () => {
      this.handleLogin();
    });

    document.getElementById("logout-btn").addEventListener("click", () => {
      this.handleLogout();
    });

    // Action buttons (Listeners removed as these buttons were part of the old static sidebar)
    // document.getElementById("build-btn").addEventListener("click", () => {
    //   this.handleBuildAction();
    // });
    //
    // document.getElementById("send-fleet-btn").addEventListener("click", () => {
    //   this.handleSendFleetAction();
    // });
    //
    // document.getElementById("trade-route-btn").addEventListener("click", () => {
    //   this.handleTradeRouteAction();
    // });
    //
    // document.getElementById("colonize-btn").addEventListener("click", () => {
    //   this.handleColonizeAction();
    // });

    // Keyboard shortcuts
    document.addEventListener("keydown", (e) => {
      this.handleKeyboardInput(e);
    });

    // Modal handling
    const modalOverlay = document.getElementById("modal-overlay");
    modalOverlay.addEventListener("click", (e) => {
      if (e.target === modalOverlay) {
        this.uiController.hideModal();
      }
    });
  }

  async handleLogin() {
    try {
      await authManager.loginWithDiscord();
    } catch (error) {
      console.error("Login failed:", error);
      this.uiController.showError("Login failed. Please try again.");
    }
  }

  handleLogout() {
    authManager.logout();
  }

  handleContextMenuAction(action, systemId) {
    const system = gameState.systems.find((s) => s.id === systemId);
    if (!system) return;

    switch (action) {
      case "view":
        gameState.selectSystem(systemId);
        this.mapRenderer.centerOnSystem(systemId);
        break;
      case "fleet":
        this.uiController.showSendFleetModal(system);
        break;
      case "trade":
        this.uiController.showTradeRouteModal(system);
        break;
    }
  }

  handleBuildAction() {
    const selectedSystem = gameState.getSelectedSystem();
    if (!selectedSystem) {
      this.uiController.showError("Please select a system first");
      return;
    }

    if (!authManager.isLoggedIn()) {
      this.uiController.showError("Please log in first");
      return;
    }

    this.uiController.showBuildModal(selectedSystem);
  }

  handleSendFleetAction() {
    const selectedSystem = gameState.getSelectedSystem();
    if (!selectedSystem) {
      this.uiController.showError("Please select a system first");
      return;
    }

    if (!authManager.isLoggedIn()) {
      this.uiController.showError("Please log in first");
      return;
    }

    this.uiController.showSendFleetModal(selectedSystem);
  }

  handleTradeRouteAction() {
    const selectedSystem = gameState.getSelectedSystem();
    if (!selectedSystem) {
      this.uiController.showError("Please select a system first");
      return;
    }

    if (!authManager.isLoggedIn()) {
      this.uiController.showError("Please log in first");
      return;
    }

    this.uiController.showTradeRouteModal(selectedSystem);
  }

  handleColonizeAction() {
    const selectedSystem = gameState.getSelectedSystem();
    if (!selectedSystem) {
      this.uiController.showError("Please select a system first");
      return;
    }

    if (!authManager.isLoggedIn()) {
      this.uiController.showError("Please log in first");
      return;
    }

    this.uiController.showColonizeModal(selectedSystem);
  }

  getConnectedSystems(currentSystem) {
    if (!currentSystem || !gameState.systems) return {};

    const connected = {};
    const currentX = currentSystem.x;
    const currentY = currentSystem.y;

    // Find systems by direction relative to current system
    gameState.systems.forEach((system) => {
      if (system.id === currentSystem.id) return;

      const deltaX = system.x - currentX;
      const deltaY = system.y - currentY;
      const distance = Math.sqrt(deltaX * deltaX + deltaY * deltaY);

      // Only consider reasonably close systems (within ~800 units)
      if (distance > 800) return;

      // Determine primary direction
      const angle = (Math.atan2(deltaY, deltaX) * 180) / Math.PI;

      // Convert angle to direction (with some tolerance)
      if (angle >= -45 && angle <= 45) {
        // Right
        if (!connected.right || distance < connected.right.distance) {
          connected.right = { system, distance };
        }
      } else if (angle >= 45 && angle <= 135) {
        // Down
        if (!connected.down || distance < connected.down.distance) {
          connected.down = { system, distance };
        }
      } else if (angle >= 135 || angle <= -135) {
        // Left
        if (!connected.left || distance < connected.left.distance) {
          connected.left = { system, distance };
        }
      } else {
        // Up
        if (!connected.up || distance < connected.up.distance) {
          connected.up = { system, distance };
        }
      }
    });

    return connected;
  }

  navigateToSystem(direction) {
    const currentSystem = gameState.getSelectedSystem();
    if (!currentSystem) return;

    const connected = this.getConnectedSystems(currentSystem);
    const target = connected[direction];

    if (target && target.system) {
      // Select and center on the new system
      gameState.selectSystem(target.system.id);
      this.mapRenderer.centerOnSystem(target.system.id);

      // Get planets for the new system and show the system view
      const planetsInSystem =
        gameState.mapData?.planets?.filter(
          (p) => p.system_id === target.system.id,
        ) || [];

      this.uiController.displaySystemView(target.system, planetsInSystem);
    }
  }

  async sendFleetToSystem(direction) {
    const currentSystem = gameState.getSelectedSystem();
    if (!currentSystem) {
      this.uiController.showToast("Select a system first", "error");
      return;
    }

    if (!authManager.isLoggedIn()) {
      this.uiController.showToast("Please log in to send fleets", "error");
      return;
    }

    const connected = this.getConnectedSystems(currentSystem);
    const target = connected[direction];

    if (!target || !target.system) {
      this.uiController.showToast(
        `No system found to the ${direction}`,
        "error",
      );
      return;
    }

    // Check if player has fleets at the current system
    const playerFleets =
      gameState.fleets?.filter(
        (fleet) =>
          fleet.owner_id === authManager.getUser()?.id &&
          fleet.current_system === currentSystem.id &&
          !fleet.destination_system,
      ) || [];

    if (playerFleets.length === 0) {
      this.uiController.showToast(
        "No available fleets at this system",
        "error",
      );
      return;
    }

    try {
      // Send fleet with default strength
      const response = await gameData.sendFleet(
        currentSystem.id,
        target.system.id,
        10,
      );

      if (response) {
        this.uiController.showToast(
          `🚀 Fleet dispatched to ${target.system.name || `System ${target.system.id.slice(-4)}`}`,
        );

        // Visual feedback - draw a temporary line
        this.mapRenderer.showFleetRoute(currentSystem, target.system);
      }
    } catch (error) {
      console.error("Failed to send fleet:", error);
      this.uiController.showToast(
        `Failed to send fleet: ${error.message || "Unknown error"}`,
        "error",
      );
    }
  }

  async handleMultiMoveFleet(selectedFleet, toSystem) {
    if (!authManager.isLoggedIn()) {
      this.uiController.showToast("Please log in to send fleets", "error");
      return;
    }

    // Check if fleet is owned by current user
    if (selectedFleet.owner_id !== authManager.getUser()?.id) {
      this.uiController.showToast("You don't own this fleet", "error");
      return;
    }

    // Check if fleet already has pending orders
    const existingOrder = gameState.fleetOrders?.find(order => 
      order.fleet_id === selectedFleet.id && 
      (order.status === "pending" || order.status === "processing")
    );
    
    if (existingOrder) {
      this.uiController.showToast("Fleet already has pending orders", "error");
      return;
    }

    // Get the fleet's current system
    const fromSystem = this.mapRenderer.systems.find(s => s.id === selectedFleet.current_system);
    if (!fromSystem) {
      this.uiController.showToast("Fleet's location not found", "error");
      return;
    }

    // For now, use new fleet orders system for direct moves only
    // TODO: Implement multi-hop routing with fleet orders later
    const path = this.findFleetPath(fromSystem, toSystem);
    
    if (!path || path.length < 2) {
      this.uiController.showToast("No valid route found to target system", "error");
      return;
    }

    if (path.length === 2) {
      // Direct hop - use single fleet orders system
      try {
        console.log(`🚀 Creating single-hop fleet order: ${fromSystem.name || fromSystem.id.slice(-4)} → ${toSystem.name || toSystem.id.slice(-4)}`);
        
        const result = await gameData.sendFleet(selectedFleet.current_system, toSystem.id, null, selectedFleet.id);
        
        this.uiController.showToast(
          `Fleet order created: ${toSystem.name || `System ${toSystem.id.slice(-4)}`} (arrives in ~20s)`,
          "success"
        );
        
        console.log("Single-hop fleet order created:", result);
        
      } catch (error) {
        console.error("Failed to create fleet order:", error);
        this.uiController.showToast(
          error.message || "Failed to create fleet order",
          "error"
        );
      }
    } else {
      // Multi-hop - use new fleet route system!
      try {
        console.log(`🚀 Creating multi-hop fleet route: ${path.length - 1} hops`);
        console.log(`Route: ${path.map(s => s.name || s.id.slice(-4)).join(" → ")}`);
        
        // Convert path to system IDs
        const routePath = path.map(system => system.id);
        
        const result = await gameData.sendFleetRoute(selectedFleet.id, routePath);
        
        this.uiController.showToast(
          `Multi-hop route created: ${toSystem.name || `System ${toSystem.id.slice(-4)}`} (${path.length - 1} hops, ~${(path.length - 1) * 20}s total)`,
          "success"
        );
        
        console.log("Multi-hop fleet route created:", result);
        
        // Store route data for visualization
        this.fleetRoutes.set(selectedFleet.id, {
          fullPath: path,
          currentHop: 0,
          targetSystem: toSystem,
          lastUpdate: Date.now(),
          isMultiHop: true,
          totalHops: path.length - 1
        });
        
        // Show route visualization on map
        this.mapRenderer.showFleetRoute(path, 0);
        
      } catch (error) {
        console.error("Failed to create multi-hop fleet route:", error);
        this.uiController.showToast(
          error.message || "Failed to create multi-hop fleet route",
          "error"
        );
      }
    }
  }

  async sendNextFleetHop(fleetId, path) {
    const routeData = this.fleetRoutes.get(fleetId);
    console.log(`DEBUG: sendNextFleetHop called for fleet ${fleetId}, route data:`, routeData);
    if (!routeData || routeData.currentHop >= path.length - 1) {
      // Route complete or invalid
      console.log(`DEBUG: Route complete or invalid for fleet ${fleetId}, cleaning up`);
      this.fleetRoutes.delete(fleetId);
      return;
    }

    const nextSystemIndex = routeData.currentHop + 1;
    const nextSystem = path[nextSystemIndex];
    const currentSystem = path[routeData.currentHop];

    // Calculate distance and travel time for this hop
    const deltaX = nextSystem.x - currentSystem.x;
    const deltaY = nextSystem.y - currentSystem.y;
    const distance = Math.sqrt(deltaX * deltaX + deltaY * deltaY);
    
    // Scale travel time with distance: 10-120 seconds (800 units = 2 minutes max)
    const travelSec = Math.max(10, Math.min(120, Math.round(distance / 800 * 120)));

    try {
      const response = await gameData.sendMultiMoveFleet(
        fleetId,
        nextSystem.id,
        travelSec
      );

      if (response) {
        // Immediately update fleet visually on frontend
        const fleet = gameState.fleets.find(f => f.id === fleetId);
        if (fleet) {
          // Calculate ETA for frontend display
          const etaTime = new Date();
          etaTime.setSeconds(etaTime.getSeconds() + travelSec);
          
          // Update fleet state immediately for smooth movement
          fleet.destination_system = nextSystem.id;
          fleet.eta = etaTime.toISOString();
          fleet.next_stop = nextSystem.id;
          
          // Notify UI to update
          gameState.notifyCallbacks();
        }

        // Update route progress - fleet is now moving to nextSystemIndex
        // but currentHop should only be updated when the fleet actually arrives
        // We'll update it in onFleetArrival instead
        
        const isLastHop = nextSystemIndex === path.length - 1;
        const systemName = nextSystem.name || `System ${nextSystem.id.slice(-4)}`;
        
        if (isLastHop) {
          const pathNames = path
            .map((sys) => sys.name || `System ${sys.id.slice(-4)}`)
            .join(" → ");
          this.uiController.showToast(
            `Fleet dispatched via: ${pathNames}`,
            "info",
            3000
          );
          // Visual feedback - draw the entire route
          this.mapRenderer.showMultiFleetRoute(path);
        } else {
          this.uiController.showToast(
            `Fleet en route to ${systemName} (hop ${nextSystemIndex}/${path.length - 1})`,
            "info",
            2000
          );
        }
      }
    } catch (error) {
      console.error("Failed to send fleet hop:", error);
      this.uiController.showToast(
        "Failed to dispatch fleet. Try again.",
        "error",
      );
      // Clean up route on error
      this.fleetRoutes.delete(fleetId);
    }
  }

  onFleetArrival(fleetId) {
    console.log(`DEBUG: onFleetArrival called for fleet ${fleetId}`);
    // Check if this fleet has route visualization data
    const routeData = this.fleetRoutes.get(fleetId);
    console.log(`DEBUG: Route data for fleet ${fleetId}:`, routeData);
    
    if (routeData && routeData.isMultiHop) {
      // Update visualization data based on backend fleet orders
      const activeOrder = gameState.fleetOrders?.find(order => 
        order.fleet_id === fleetId && 
        (order.status === "pending" || order.status === "processing")
      );
      
      if (activeOrder) {
        // Update our visualization to match backend state
        const currentHop = activeOrder.current_hop || 0;
        routeData.currentHop = currentHop;
        routeData.lastUpdate = Date.now();
        
        console.log(`DEBUG: Updated route visualization for fleet ${fleetId}, hop ${currentHop}/${routeData.totalHops}`);
        
        // Update map visualization
        this.mapRenderer.showFleetRoute(routeData.fullPath, currentHop);
      } else {
        // No active order found - route completed
        console.log(`DEBUG: Fleet ${fleetId} route completed, cleaning up visualization`);
        this.fleetRoutes.delete(fleetId);
        this.mapRenderer.clearFleetRoute();
      }
    } else {
      console.log(`DEBUG: No multi-hop route data found for fleet ${fleetId}`);
    }
  }



  findFleetPath(fromSystem, toSystem) {
    // Breadth-first search pathfinding using hyperlane connectivity
    const visited = new Set();
    const queue = [{ system: fromSystem, path: [fromSystem] }];
    const maxHops = 15; // Maximum number of hops to prevent infinite loops

    console.log(`🗺️ Pathfinding from ${fromSystem.name || fromSystem.id.slice(-4)} to ${toSystem.name || toSystem.id.slice(-4)}`);

    while (queue.length > 0) {
      const current = queue.shift();
      const currentSystem = current.system;

      if (currentSystem.id === toSystem.id) {
        console.log(`✅ Path found with ${current.path.length} hops:`, current.path.map(s => s.name || s.id.slice(-4)));
        return current.path;
      }

      if (visited.has(currentSystem.id) || current.path.length > maxHops) {
        continue;
      }

      visited.add(currentSystem.id);

      // Find all systems connected via hyperlanes (same logic as visual display)
      const connectedSystems =
        this.mapRenderer.systems?.filter((system) => {
          if (system.id === currentSystem.id || visited.has(system.id)) {
            return false;
          }

          return this.mapRenderer.areSystemsConnected(currentSystem, system);
        }) || [];

      console.log(`🔍 From ${currentSystem.name || currentSystem.id.slice(-4)}: found ${connectedSystems.length} connected systems (hop ${current.path.length})`);

      // Add connected systems to queue
      connectedSystems.forEach((system) => {
        if (!visited.has(system.id)) {
          queue.push({
            system: system,
            path: [...current.path, system],
          });
        }
      });
    }

    console.log(`❌ No path found from ${fromSystem.name || fromSystem.id.slice(-4)} to ${toSystem.name || toSystem.id.slice(-4)}`);
    return null; // No path found
  }

  calculatePathDistance(path) {
    let totalDistance = 0;

    for (let i = 0; i < path.length - 1; i++) {
      const from = path[i];
      const to = path[i + 1];

      const deltaX = to.x - from.x;
      const deltaY = to.y - from.y;
      const distance = Math.sqrt(deltaX * deltaX + deltaY * deltaY);

      totalDistance += distance;
    }

    return totalDistance;
  }

  displaySelectedFleetInfo(fleet) {
    const fleetName = fleet.name || `Fleet ${fleet.id.slice(-4)}`;
    const currentSystem = this.mapRenderer.systems.find(s => s.id === fleet.current_system);
    const currentName = currentSystem ? (currentSystem.name || `System ${currentSystem.id.slice(-4)}`) : "Unknown";
    
    // Check if fleet is moving
    let ticketContent = "";
    if (fleet.destination_system) {
      // Moving fleet - show transit info
      const destSystem = this.mapRenderer.systems.find(s => s.id === fleet.destination_system);
      const destName = destSystem ? (destSystem.name || `System ${destSystem.id.slice(-4)}`) : "Unknown";
      const nextSystem = this.mapRenderer.systems.find(s => s.id === fleet.next_stop);
      const nextName = nextSystem ? (nextSystem.name || `System ${nextSystem.id.slice(-4)}`) : null;
      
      // Format ETA
      let etaText = "Unknown";
      if (fleet.eta) {
        const eta = new Date(fleet.eta);
        const now = new Date();
        const diffMs = eta.getTime() - now.getTime();
        const diffMin = Math.ceil(diffMs / (1000 * 60));
        etaText = diffMin > 0 ? `${diffMin}m` : "Arriving";
      }
      
      ticketContent = `
        <div class="font-mono text-xs bg-slate-800 p-2 rounded">
          <div class="space-y-0.5">
            <div><span class="text-slate-400">FLEET:</span> ${fleetName}</div>
            <div><span class="text-slate-400">FROM:</span> ${currentName}</div>
            ${nextName && nextName !== destName ? `<div><span class="text-slate-400">NEXT:</span> ${nextName}</div>` : ''}
            <div><span class="text-slate-400">DEST:</span> ${destName}</div>
            <div><span class="text-slate-400">ETA:</span> ${etaText}</div>
          </div>
        </div>
      `;
    } else {
      // Stationary fleet - show selection info
      let shipInfo = "No ships";
      if (fleet.ships && fleet.ships.length > 0) {
        shipInfo = fleet.ships.map(ship => 
          `${ship.count}x ${ship.ship_type_name || 'Unknown'}`
        ).join(', ');
      }
      
      ticketContent = `
        <div class="font-mono text-xs bg-slate-800 p-2 rounded">
          <div class="space-y-0.5">
            <div><span class="text-slate-400">FLEET:</span> ${fleetName}</div>
            <div><span class="text-slate-400">LOCATION:</span> ${currentName}</div>
            <div><span class="text-slate-400">SHIPS:</span> ${shipInfo}</div>
            <div><span class="text-slate-400">STATUS:</span> Docked</div>
          </div>
        </div>
      `;
    }
    
    this.uiController.showToast(ticketContent, 'ticket', 0); // 0 duration = manual dismiss only
  }

  handleKeyboardInput(e) {
    // Only handle keyboard shortcuts when not in input fields
    if (e.target.tagName === "INPUT" || e.target.tagName === "TEXTAREA") {
      return;
    }

    switch (e.key.toLowerCase()) {
      case "escape":
        this.uiController.hideModal();
        document.getElementById("context-menu").classList.add("hidden");
        break;
      case "arrowup":
        e.preventDefault();
        if (e.shiftKey) {
          this.sendFleetToSystem("up");
        } else {
          this.navigateToSystem("up");
        }
        break;
      case "arrowdown":
        e.preventDefault();
        if (e.shiftKey) {
          this.sendFleetToSystem("down");
        } else {
          this.navigateToSystem("down");
        }
        break;
      case "arrowleft":
        e.preventDefault();
        if (e.shiftKey) {
          this.sendFleetToSystem("left");
        } else {
          this.navigateToSystem("left");
        }
        break;
      case "arrowright":
        e.preventDefault();
        if (e.shiftKey) {
          this.sendFleetToSystem("right");
        } else {
          this.navigateToSystem("right");
        }
        break;
      case "f":
        this.handleSendFleetAction();
        break;
      case "t":
        this.handleTradeRouteAction();
        break;
      case "b":
        this.handleBuildAction();
        break;
      case "c":
        if (gameState.getSelectedSystem()) {
          this.mapRenderer.centerOnSystem(gameState.getSelectedSystem().id);
        }
        break;
      case "o":
        this.handleColonizeAction();
        break;
      case "h":
        this.mapRenderer.fitToSystems();
        break;
    }
  }
}

// Start the application
const app = new XanNationApp();
window.app = app;
