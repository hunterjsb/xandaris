// Canvas-based map renderer for the 4X game
export class MapRenderer {
  constructor(canvasId) {
    this.canvas = document.getElementById(canvasId);
    this.ctx = this.canvas.getContext('2d');
    this.systems = [];
    this.lanes = [];
    this.fleets = [];
    this.selectedSystem = null;
    this.hoveredSystem = null;
    this.hoveredTradeRoutes = [];
    this.trades = [];
    this.currentUserId = null; // Added to store current player's ID
    
    // Cache for territorial contours
    this.cachedTerritorialContours = null;
    this.territorialCacheKey = null;
    this.connectedSystems = new Map(); // Track systems connected to selected system
    this.fleetRoutes = []; // Track temporary fleet routes for visualization
    
    // View settings - adjusted for larger galaxy (6000x4500)
    this.viewX = 0;
    this.viewY = 0;
    this.zoom = 0.15;  // Start zoomed out to see full galaxy
    this.maxZoom = 2.0; // Reduced max zoom for better performance
    this.minZoom = 0.05; // Much lower min zoom to see entire large galaxy
    
    // Smooth camera movement
    this.targetViewX = 0;
    this.targetViewY = 0;
    this.cameraSpeed = 0.15; // How fast camera moves (0-1)
    
    // Deep Space Colors
    this.colors = {
      background: '#000508',
      starUnowned: '#4080ff',        // Nebula blue for unowned (original 'star')
      starPlayerOwned: '#00ff66', // Bright green for player-owned colonized planets
      starOtherOwned: '#f1a9ff',   // Plasma pink for other AI/players (original 'starOwned')
      starEnemy: '#ff6b6b',   // Bright red for enemies (already distinct)
      // starNeutral: '#8cb3ff', // Lighter blue for neutral - can be an alias for starUnowned or specific if needed
      lane: 'rgba(64, 128, 255, 0.2)',  // Faint blue lanes
      laneActive: 'rgba(241, 169, 255, 0.6)', // Active plasma lanes
      fleet: '#8b5cf6',       // Void purple for fleets
      selection: '#f1a9ff',   // Plasma selection
      grid: 'rgba(255, 255, 255, 0.02)', // Very faint grid
      nebula: 'rgba(139, 92, 246, 0.1)',  // Background nebula effect
      starGlow: 'rgba(64, 128, 255, 0.3)' // Star glow effect
    };
    
    // Animation
    this.animationFrame = null;
    this.lastTime = 0;
    
    this.setupCanvas();
    this.setupEventListeners();
    this.startRenderLoop();
    
    // Center the galaxy view on initial load
    this.initialViewSet = false;
    
    // Planet count cache for system scaling
    this.systemPlanetCounts = new Map();
  }

  setupCanvas() {
    this.resizeCanvas();
    window.addEventListener('resize', () => this.resizeCanvas());
  }

  resizeCanvas() {
    const rect = this.canvas.getBoundingClientRect();
    this.canvas.width = rect.width;
    this.canvas.height = rect.height;
  }

  setupEventListeners() {
    let isPanning = false;
    let lastMouseX = 0;
    let lastMouseY = 0;

    // Mouse events
    this.canvas.addEventListener('mousedown', (e) => {
      if (e.button === 0) { // Left click
        const worldPos = this.screenToWorld(e.offsetX, e.offsetY);
        const clickedSystem = this.getSystemAt(worldPos.x, worldPos.y);
        
        if (clickedSystem) {
          this.selectSystem(clickedSystem); // Selects and centers
          // Calculate where the system will appear after centering (offset slightly from center)
          const centeredScreenPos = {
            x: this.canvas.width / 2 + 30, // Offset right to avoid covering system icon
            y: this.canvas.height / 2 - 20  // Offset up slightly
          };
          const planetsInSystem = window.gameState.getSystemPlanets(clickedSystem.id);
          this.canvas.dispatchEvent(new CustomEvent('systemSelected', {
            detail: { system: clickedSystem, planets: planetsInSystem, screenX: centeredScreenPos.x, screenY: centeredScreenPos.y },
            bubbles: true
          }));
        } else {
          this.selectSystem(null); // Deselect system visually on map
          // Dispatch event for UI to hide panel
          this.canvas.dispatchEvent(new CustomEvent('mapClickedEmpty', { bubbles: true }));
          isPanning = true;
          lastMouseX = e.offsetX;
          lastMouseY = e.offsetY;
          this.canvas.style.cursor = 'grabbing';
        }
      }
    });

    this.canvas.addEventListener('mousemove', (e) => {
      if (isPanning) {
        const deltaX = e.offsetX - lastMouseX;
        const deltaY = e.offsetY - lastMouseY;
        this.viewX += deltaX / this.zoom;
        this.viewY += deltaY / this.zoom;
        // Update target to current position during manual panning
        this.targetViewX = this.viewX;
        this.targetViewY = this.viewY;
        lastMouseX = e.offsetX;
        lastMouseY = e.offsetY;
      } else {
        const worldPos = this.screenToWorld(e.offsetX, e.offsetY);
        // Update this.hoveredSystem. This will be used by drawSystems in the next render frame.
        this.hoveredSystem = this.getSystemAt(worldPos.x, worldPos.y);
        // Tooltip should still be shown based on the now updated this.hoveredSystem
        this.showTooltip(this.hoveredSystem, e.offsetX, e.offsetY);
        if (this.hoveredSystem && this.trades && this.trades.length > 0) {
          this.hoveredTradeRoutes = this.trades.filter(trade =>
            trade.from_id === this.hoveredSystem.id || trade.to_id === this.hoveredSystem.id
          );
        } else {
          this.hoveredTradeRoutes = [];
        }
      }
    });

    this.canvas.addEventListener('mouseup', (e) => {
      if (e.button === 0) {
        isPanning = false;
        this.canvas.style.cursor = 'crosshair';
      }
    });

    this.canvas.addEventListener('contextmenu', (e) => {
      e.preventDefault();
      const worldPos = this.screenToWorld(e.offsetX, e.offsetY);
      const clickedSystem = this.getSystemAt(worldPos.x, worldPos.y);
      
      if (clickedSystem) {
        this.showContextMenu(clickedSystem, e.offsetX, e.offsetY);
      }
    });

    // Zoom with mouse wheel
    this.canvas.addEventListener('wheel', (e) => {
      e.preventDefault();
      const zoomFactor = e.deltaY > 0 ? 0.9 : 1.1;
      const newZoom = Math.max(this.minZoom, Math.min(this.maxZoom, this.zoom * zoomFactor));
      
      if (newZoom !== this.zoom) {
        // Zoom towards mouse position
        const rect = this.canvas.getBoundingClientRect();
        const mouseX = e.clientX - rect.left;
        const mouseY = e.clientY - rect.top;
        
        const worldBefore = this.screenToWorld(mouseX, mouseY);
        this.zoom = newZoom;
        const worldAfter = this.screenToWorld(mouseX, mouseY);
        
        this.viewX += worldBefore.x - worldAfter.x;
        this.viewY += worldBefore.y - worldAfter.y;
        // Update target to current position during zoom
        this.targetViewX = this.viewX;
        this.targetViewY = this.viewY;
      }
    });
  }

  screenToWorld(screenX, screenY) {
    return {
      x: (screenX - this.canvas.width / 2) / this.zoom - this.viewX,
      y: (screenY - this.canvas.height / 2) / this.zoom - this.viewY
    };
  }

  worldToScreen(worldX, worldY) {
    return {
      x: (worldX + this.viewX) * this.zoom + this.canvas.width / 2,
      y: (worldY + this.viewY) * this.zoom + this.canvas.height / 2
    };
  }

  getSystemAt(worldX, worldY) {
    const radius = 20; // Hit detection radius
    return this.systems.find(system => {
      const dx = system.x - worldX;
      const dy = system.y - worldY;
      return Math.sqrt(dx * dx + dy * dy) <= radius;
    });
  }

  selectSystem(system) {
    // If the same system is clicked, and it's already selected, do nothing.
    // Allows re-triggering panel if it was closed by other means by handling event in mousedown.
    if (this.selectedSystem && system && this.selectedSystem.id === system.id) {
      // Still center, as focus might have shifted.
      this.centerOnSystem(system.id);
      return;
    }
    
    this.selectedSystem = system; // Set or clear selected system
    if (system) {
      this.centerOnSystem(system.id); // Center map on newly selected system
    }
    // Event dispatch with coordinates is now handled in mousedown
  }

  updateConnectedSystems() {
    this.connectedSystems.clear();
    if (!this.selectedSystem) return;

    const currentX = this.selectedSystem.x;
    const currentY = this.selectedSystem.y;

    this.systems.forEach(system => {
      if (system.id === this.selectedSystem.id) return;

      const deltaX = system.x - currentX;
      const deltaY = system.y - currentY;
      const distance = Math.sqrt(deltaX * deltaX + deltaY * deltaY);

      // Only consider reasonably close systems (within ~800 units)
      if (distance > 800) return;

      // Determine primary direction
      const angle = Math.atan2(deltaY, deltaX) * 180 / Math.PI;
      
      let direction;
      if (angle >= -45 && angle <= 45) {
        direction = 'right';
      } else if (angle >= 45 && angle <= 135) {
        direction = 'down';
      } else if (angle >= 135 || angle <= -135) {
        direction = 'left';
      } else {
        direction = 'up';
      }

      if (!this.connectedSystems.has(direction) || distance < this.connectedSystems.get(direction).distance) {
        this.connectedSystems.set(direction, { system, distance, direction });
      }
    });
  }

  showTooltip(system, screenX, screenY) {
    const tooltip = document.getElementById('tooltip');
    if (system && window.gameState) { // Ensure gameState is available
      const planets = window.gameState.getSystemPlanets(system.id);
      const totalSystemPop = planets.reduce((sum, p) => sum + (p.Pop || 0), 0);
      
      // Check if player owns any planets in this system
      const isPlayerOwned = planets.some(planet => planet.colonized_by === this.currentUserId);
      const ownerText = isPlayerOwned ? 'You' : 'Uncolonized';

      tooltip.innerHTML = `
        <div class="font-semibold">${system.name || `System ${system.id.slice(-4)}`}</div>
        <div class="text-xs">
          <div>Position: ${system.x}, ${system.y}</div>
          <div>Population: ${totalSystemPop.toLocaleString()}</div>
          <div>Owner: ${ownerText}</div>
          <div>Planets: ${planets.length}</div>
        </div>
      `;
      tooltip.style.left = `${screenX + 10}px`;
      tooltip.style.top = `${screenY - 10}px`;
      tooltip.classList.remove('hidden');
    } else {
      tooltip.classList.add('hidden');
    }
  }

  showContextMenu(system, screenX, screenY) {
    const menu = document.getElementById('context-menu');
    menu.style.left = `${screenX}px`;
    menu.style.top = `${screenY}px`;
    menu.classList.remove('hidden');
    
    // Store system reference for menu actions
    menu.dataset.systemId = system.id;
    
    // Hide menu when clicking elsewhere
    const hideMenu = (e) => {
      if (!menu.contains(e.target)) {
        menu.classList.add('hidden');
        document.removeEventListener('click', hideMenu);
      }
    };
    setTimeout(() => document.addEventListener('click', hideMenu), 0);
  }

  startRenderLoop() {
    const render = (currentTime) => {
      const deltaTime = currentTime - this.lastTime;
      this.lastTime = currentTime;
      
      // Smooth camera movement
      this.updateCamera(deltaTime);
      
      // Call draw methods in order
      this.clear();
      this.drawBackground();
      this.drawLanes();
      this.drawFleetRoutes();
      this.drawCachedTerritorialBorders();
      this.drawSystems();
      this.drawFleets(deltaTime);
      this.drawUI();
      
      this.animationFrame = requestAnimationFrame(render);
    };
    
    this.animationFrame = requestAnimationFrame(render);
  }

  updateCamera(deltaTime) {
    // Smooth camera interpolation
    const speed = this.cameraSpeed * (deltaTime / 16); // Normalize for 60fps
    const dx = this.targetViewX - this.viewX;
    const dy = this.targetViewY - this.viewY;
    
    // Only move if we're not close enough
    if (Math.abs(dx) > 1 || Math.abs(dy) > 1) {
      this.viewX += dx * speed;
      this.viewY += dy * speed;
    } else {
      // Snap to target when close enough
      this.viewX = this.targetViewX;
      this.viewY = this.targetViewY;
    }
  }

  clear() {
    this.ctx.fillStyle = this.colors.background;
    this.ctx.fillRect(0, 0, this.canvas.width, this.canvas.height);
  }

  drawBackground() {
    // Draw nebula clouds
    this.ctx.globalAlpha = 0.1;
    const time = Date.now() * 0.0001;
    
    // Draw several overlapping nebula clouds
    for (let i = 0; i < 3; i++) {
      const offset = i * 150;
      const x = Math.sin(time + i) * 200 + offset;
      const y = Math.cos(time * 0.7 + i) * 150 + offset;
      const size = 300 + Math.sin(time * 0.5 + i) * 50;
      
      const gradient = this.ctx.createRadialGradient(x, y, 0, x, y, size);
      gradient.addColorStop(0, this.colors.nebula);
      gradient.addColorStop(0.5, 'rgba(64, 128, 255, 0.05)');
      gradient.addColorStop(1, 'rgba(0, 0, 0, 0)');
      
      this.ctx.fillStyle = gradient;
      this.ctx.fillRect(0, 0, this.canvas.width, this.canvas.height);
    }
    
    this.ctx.globalAlpha = 1;

    // Draw subtle grid
    this.ctx.strokeStyle = this.colors.grid;
    this.ctx.lineWidth = 1;
    this.ctx.globalAlpha = 0.15;

    const gridSize = 100;
    const startX = Math.floor((-this.viewX - this.canvas.width / 2 / this.zoom) / gridSize) * gridSize;
    const endX = Math.ceil((-this.viewX + this.canvas.width / 2 / this.zoom) / gridSize) * gridSize;
    const startY = Math.floor((-this.viewY - this.canvas.height / 2 / this.zoom) / gridSize) * gridSize;
    const endY = Math.ceil((-this.viewY + this.canvas.height / 2 / this.zoom) / gridSize) * gridSize;

    this.ctx.beginPath();
    for (let x = startX; x <= endX; x += gridSize) {
      const screenPos = this.worldToScreen(x, 0);
      this.ctx.moveTo(screenPos.x, 0);
      this.ctx.lineTo(screenPos.x, this.canvas.height);
    }
    for (let y = startY; y <= endY; y += gridSize) {
      const screenPos = this.worldToScreen(0, y);
      this.ctx.moveTo(0, screenPos.y);
      this.ctx.lineTo(this.canvas.width, screenPos.y);
    }
    this.ctx.stroke();
    this.ctx.globalAlpha = 1;
  }

          drawLanes() {
            // Draw navigation hyperlanes first (based on our navigation logic)
            this.drawNavigationLanes();

            if (!this.trades || this.trades.length === 0) {
              // If there are no trades, or trades data isn't loaded, try to draw old lanes if they exist
              // This is a fallback for existing functionality if setTrades isn't called yet.
              // Ideally, this.lanes would be deprecated if this.trades is reliably populated.
              if (this.lanes && this.lanes.length > 0) {
                  this.ctx.strokeStyle = this.colors.lane;
                  this.ctx.lineWidth = 2;
                  this.ctx.globalAlpha = 0.6;

                  this.lanes.forEach(lane => {
                      const fromPos = this.worldToScreen(lane.fromX, lane.fromY);
                      const toPos = this.worldToScreen(lane.toX, lane.toY);

                      this.ctx.beginPath();
                      this.ctx.moveTo(fromPos.x, fromPos.y);
                      this.ctx.lineTo(toPos.x, toPos.y);
                      this.ctx.stroke();
                  });
                  this.ctx.globalAlpha = 1;
              }
              return;
            }

            this.ctx.globalAlpha = 0.7; // Default alpha for lanes

            this.trades.forEach(trade => {
              const fromSystem = this.systems.find(s => s.id === trade.from_id);
              const toSystem = this.systems.find(s => s.id === trade.to_id);

              if (!fromSystem || !toSystem) {
                return; // Skip if systems not found
              }

              const fromPos = this.worldToScreen(fromSystem.x, fromSystem.y);
              const toPos = this.worldToScreen(toSystem.x, toSystem.y);

              // Color based on hover state
              let color = this.colors.lane;
              if (this.hoveredTradeRoutes && this.hoveredTradeRoutes.some(route => route.id === trade.id)) {
                color = this.colors.laneActive;
                this.ctx.lineWidth = 3;
              } else {
                this.ctx.lineWidth = 2;
              }

              this.ctx.strokeStyle = color;
              this.ctx.beginPath();
              this.ctx.moveTo(fromPos.x, fromPos.y);
              this.ctx.lineTo(toPos.x, toPos.y);
              this.ctx.stroke();
            });

            this.ctx.globalAlpha = 1; // Reset globalAlpha
          }

  drawNavigationLanes() {
    if (!this.selectedSystem || this.systems.length === 0) return;

    this.ctx.strokeStyle = 'rgba(64, 128, 255, 0.15)';
    this.ctx.lineWidth = 1;
    this.ctx.globalAlpha = 0.8;
    this.ctx.setLineDash([3, 6]);

    const currentX = this.selectedSystem.x;
    const currentY = this.selectedSystem.y;

    // Draw lanes to all systems within navigation range
    this.systems.forEach(system => {
      if (system.id === this.selectedSystem.id) return;

      const deltaX = system.x - currentX;
      const deltaY = system.y - currentY;
      const distance = Math.sqrt(deltaX * deltaX + deltaY * deltaY);

      // Only draw lanes within navigation range
      if (distance > 800) return;

      const fromPos = this.worldToScreen(currentX, currentY);
      const toPos = this.worldToScreen(system.x, system.y);

      this.ctx.beginPath();
      this.ctx.moveTo(fromPos.x, fromPos.y);
      this.ctx.lineTo(toPos.x, toPos.y);
      this.ctx.stroke();
    });

    this.ctx.setLineDash([]);
    this.ctx.globalAlpha = 1;
  }

  drawFleetRoutes() {
    if (this.fleetRoutes.length === 0) return;

    this.ctx.strokeStyle = '#8b5cf6';
    this.ctx.lineWidth = 3;
    this.ctx.globalAlpha = 0.8;

    this.fleetRoutes.forEach(route => {
      const fromPos = this.worldToScreen(route.from.x, route.from.y);
      const toPos = this.worldToScreen(route.to.x, route.to.y);

      // Draw animated line
      this.ctx.setLineDash([10, 5]);
      this.ctx.lineDashOffset = -Date.now() / 50;
      
      this.ctx.beginPath();
      this.ctx.moveTo(fromPos.x, fromPos.y);
      this.ctx.lineTo(toPos.x, toPos.y);
      this.ctx.stroke();

      // Draw arrow at destination
      const angle = Math.atan2(toPos.y - fromPos.y, toPos.x - fromPos.x);
      const arrowLength = 15;
      
      this.ctx.setLineDash([]);
      this.ctx.fillStyle = '#8b5cf6';
      this.ctx.beginPath();
      this.ctx.moveTo(toPos.x, toPos.y);
      this.ctx.lineTo(
        toPos.x - arrowLength * Math.cos(angle - Math.PI / 6),
        toPos.y - arrowLength * Math.sin(angle - Math.PI / 6)
      );
      this.ctx.lineTo(
        toPos.x - arrowLength * Math.cos(angle + Math.PI / 6),
        toPos.y - arrowLength * Math.sin(angle + Math.PI / 6)
      );
      this.ctx.closePath();
      this.ctx.fill();
    });

    this.ctx.setLineDash([]);
    this.ctx.globalAlpha = 1;
  }

  showFleetRoute(fromSystem, toSystem) {
    // Add temporary route visualization
    this.fleetRoutes.push({
      from: fromSystem,
      to: toSystem,
      timestamp: Date.now()
    });

    // Remove route after 3 seconds
    setTimeout(() => {
      this.fleetRoutes = this.fleetRoutes.filter(route => 
        route.timestamp !== this.fleetRoutes[this.fleetRoutes.length - 1].timestamp
      );
    }, 3000);
  }

drawSystems() {
          this.systems.forEach(system => {
            const screenPos = this.worldToScreen(system.x, system.y);

            if (screenPos.x < -50 || screenPos.x > this.canvas.width + 50 ||
                screenPos.y < -50 || screenPos.y > this.canvas.height + 50) {
              return;
            }

            let scaleFactor = 1;
            if (this.hoveredSystem && this.hoveredSystem.id === system.id) {
              scaleFactor = 1.2;
            }

            let color;
            let isConnected = false;
            let isPlayerOwned = false;
            
            // Check if this system is connected to selected system
            for (const [direction, connectedData] of this.connectedSystems) {
              if (connectedData.system.id === system.id) {
                isConnected = true;
                break;
              }
            }

            // Check if player owns any planets in this system
            if (window.gameState) {
              const planets = window.gameState.getSystemPlanets(system.id);
              isPlayerOwned = planets.some(planet => planet.colonized_by === this.currentUserId);
            }

            if (isPlayerOwned) {
              color = this.colors.starPlayerOwned;
            } else if (!system.owner_id) {
              color = this.colors.starUnowned;
            } else {
              // Here, you might distinguish between neutral AI and enemy AI if data allows.
              // For now, any other owner uses starOtherOwned.
              // if (system.isEnemy) color = this.colors.starEnemy; else
              color = this.colors.starOtherOwned;
            }

            // Scale system size based on planet count
            const planetCount = this.systemPlanetCounts.get(system.id) || 1;
            const planetScale = Math.min(1 + (planetCount - 1) * 0.2, 2.0); // 1x to 2x scale based on planets
            
            const baseSystemDrawRadius = 6 * this.zoom * planetScale;
            const systemDrawRadius = baseSystemDrawRadius * scaleFactor;
            const glowDrawRadius = (20 * this.zoom) * scaleFactor * planetScale;

            const gradient = this.ctx.createRadialGradient(
              screenPos.x, screenPos.y, 0,
              screenPos.x, screenPos.y, glowDrawRadius
            );
            gradient.addColorStop(0, color + '40');
            gradient.addColorStop(0.3, color + '20');
            gradient.addColorStop(1, color + '00');

            this.ctx.fillStyle = gradient;
            this.ctx.beginPath();
            this.ctx.arc(screenPos.x, screenPos.y, glowDrawRadius, 0, Math.PI * 2);
            this.ctx.fill();

            // Add navigation hint for connected systems
            if (isConnected && this.selectedSystem) {
              this.ctx.strokeStyle = '#ffffff80';
              this.ctx.lineWidth = 2;
              this.ctx.setLineDash([5, 5]);
              this.ctx.beginPath();
              this.ctx.arc(screenPos.x, screenPos.y, systemDrawRadius + 4, 0, Math.PI * 2);
              this.ctx.stroke();
              this.ctx.setLineDash([]);
            }

            const innerGradient = this.ctx.createRadialGradient(
              screenPos.x, screenPos.y, 0,
              screenPos.x, screenPos.y, systemDrawRadius
            );
            innerGradient.addColorStop(0, '#ffffff');
            innerGradient.addColorStop(0.7, color);
            innerGradient.addColorStop(1, color + 'cc');

            this.ctx.fillStyle = innerGradient;
            this.ctx.beginPath();
            this.ctx.arc(screenPos.x, screenPos.y, systemDrawRadius, 0, Math.PI * 2);
            this.ctx.fill();

            // Add border for colonized planets (player-owned)
            if (isPlayerOwned) {
              this.ctx.strokeStyle = '#00ff88'; // Slightly brighter green for border
              this.ctx.lineWidth = 2 * this.zoom * scaleFactor;
              this.ctx.globalAlpha = 0.9;
              this.ctx.beginPath();
              this.ctx.arc(screenPos.x, screenPos.y, systemDrawRadius + 2, 0, Math.PI * 2);
              this.ctx.stroke();
              this.ctx.globalAlpha = 1;
            }

            if (isPlayerOwned && this.zoom > 0.6) {
              this.ctx.strokeStyle = this.colors.starPlayerOwned;
              this.ctx.lineWidth = 1;
              this.ctx.globalAlpha = 0.8;

              this.ctx.beginPath();
              this.ctx.moveTo(screenPos.x - baseSystemDrawRadius * 1.5, screenPos.y);
              this.ctx.lineTo(screenPos.x + baseSystemDrawRadius * 1.5, screenPos.y);
              this.ctx.moveTo(screenPos.x, screenPos.y - baseSystemDrawRadius * 1.5);
              this.ctx.lineTo(screenPos.x, screenPos.y + baseSystemDrawRadius * 1.5);
              this.ctx.stroke();

              this.ctx.globalAlpha = 1;
            }

            if (this.selectedSystem && this.selectedSystem.id === system.id) {
              const time = Date.now() * 0.005; // Keep time for pulse

              // Enhanced: Persistent thicker border for selected system
              const selectedBorderRadius = (baseSystemDrawRadius + 8 * this.zoom) * scaleFactor; // Slightly larger than pulse
              this.ctx.strokeStyle = this.colors.selection; // Use a distinct selection color
              this.ctx.lineWidth = 3 * this.zoom * scaleFactor; // Thicker line
              this.ctx.globalAlpha = 0.9; // Strong alpha
              this.ctx.beginPath();
              this.ctx.arc(screenPos.x, screenPos.y, selectedBorderRadius, 0, Math.PI * 2);
              this.ctx.stroke();

              // Pulse animation (slightly increased radius and opacity)
              const pulseRadius = (baseSystemDrawRadius + (6 + Math.sin(time) * 3) * this.zoom) * scaleFactor; // Increased pulse range
              this.ctx.strokeStyle = this.colors.selection;
              this.ctx.lineWidth = 2 * this.zoom * scaleFactor; // Standard pulse line width
              this.ctx.globalAlpha = 0.7 + Math.sin(time) * 0.3; // More variance in opacity
              this.ctx.beginPath();
              this.ctx.arc(screenPos.x, screenPos.y, pulseRadius, 0, Math.PI * 2);
              this.ctx.stroke();
              this.ctx.globalAlpha = 1; // Reset alpha
            }

            if (this.zoom > 0.8) {
              const fontSize = Math.floor(11 * this.zoom);
              this.ctx.font = `${fontSize}px monospace`;
              this.ctx.textAlign = 'center';

              const textYOffset = systemDrawRadius + (5 * this.zoom);

              this.ctx.fillStyle = 'rgba(0, 0, 0, 0.8)';
              this.ctx.fillText(
                system.name || `S${system.id.slice(-3)}`,
                screenPos.x + 1,
                screenPos.y - textYOffset + 1
              );

              this.ctx.fillStyle = 'rgba(255, 255, 255, 0.95)';
              this.ctx.fillText(
                system.name || `S${system.id.slice(-3)}`,
                screenPos.x,
                screenPos.y - textYOffset
              );
            }

            // Calculate total population for the system
            let totalSystemPop = 0;
            if (window.gameState) { // Check if gameState is available
                const planets = window.gameState.getSystemPlanets(system.id);
                totalSystemPop = planets.reduce((sum, p) => sum + (p.Pop || 0), 0);
            }

            if (totalSystemPop > 0 && this.zoom > 0.5) {
              const popFontSize = Math.floor(9 * this.zoom);
              this.ctx.font = `${popFontSize}px monospace`;
              this.ctx.textAlign = 'center';

              const popRectBaseWidth = 16 * this.zoom;
              const popRectBaseHeight = 12 * this.zoom;

              const popRectWidth = popRectBaseWidth * scaleFactor;
              const popRectHeight = popRectBaseHeight * scaleFactor;

              const popRectY = screenPos.y + systemDrawRadius + (2 * this.zoom);
              const popTextY = popRectY + popRectHeight - (3 * this.zoom * scaleFactor);

              this.ctx.fillStyle = 'rgba(241, 169, 255, 0.2)';
              this.ctx.fillRect(
                screenPos.x - popRectWidth / 2,
                popRectY,
                popRectWidth,
                popRectHeight
              );

              this.ctx.fillStyle = '#f1a9ff';
              this.ctx.fillText(
                totalSystemPop.toLocaleString(),
                screenPos.x,
                popTextY
              );
            }
          });
        }

  drawCachedTerritorialBorders() {
    if (!this.currentUserId) return;

    // Get all player-owned systems
    const playerSystems = this.systems.filter(system => {
      if (window.gameState) {
        const planets = window.gameState.getSystemPlanets(system.id);
        return planets.some(planet => planet.colonized_by === this.currentUserId);
      }
      return false;
    });

    if (playerSystems.length < 1) return;

    // Create cache key based on player systems and zoom/view
    const cacheKey = this.createTerritorialCacheKey(playerSystems);
    
    // Only recompute if cache is invalid
    if (this.territorialCacheKey !== cacheKey) {
      this.cachedTerritorialContours = this.computeTerritorialContours(playerSystems);
      this.territorialCacheKey = cacheKey;
    }

    // Draw cached contours
    if (this.cachedTerritorialContours) {
      this.drawTerritorialContours(this.cachedTerritorialContours);
    }
  }

  createTerritorialCacheKey(playerSystems) {
    // Create a simple cache key based on system positions and current view
    const systemKey = playerSystems.map(s => `${s.id}:${s.x}:${s.y}`).sort().join('|');
    const viewKey = `${Math.floor(this.viewX/50)}:${Math.floor(this.viewY/50)}:${Math.floor(this.zoom*10)}`;
    return `${systemKey}@${viewKey}`;
  }

  computeTerritorialContours(playerSystems) {
    // Create influence field across visible area
    const bounds = this.getVisibleWorldBounds();
    const gridSize = 40; // Larger grid for better performance
    const influenceField = this.calculateInfluenceField(playerSystems, bounds, gridSize);
    
    // Extract contours from influence field
    return this.extractTerritorialContours(influenceField, bounds, gridSize);
  }

  drawUnifiedTerritories(playerSystems) {
    // Much simpler approach - just draw extended borders around player systems
    const maxDistance = 200; // Reduced from 500
    
    playerSystems.forEach(system => {
      this.drawSimpleInfluenceBorder(system, playerSystems, maxDistance);
    });
  }

  getVisibleWorldBounds() {
    const padding = 200;
    const topLeft = this.screenToWorld(-padding, -padding);
    const bottomRight = this.screenToWorld(this.canvas.width + padding, this.canvas.height + padding);
    
    return {
      minX: topLeft.x,
      minY: topLeft.y,
      maxX: bottomRight.x,
      maxY: bottomRight.y
    };
  }

  calculateInfluenceField(playerSystems, bounds, gridSize) {
    const width = Math.ceil((bounds.maxX - bounds.minX) / gridSize);
    const height = Math.ceil((bounds.maxY - bounds.minY) / gridSize);
    const field = new Array(height).fill(null).map(() => new Array(width).fill(0));
    
    // Smaller influence range for performance and more reasonable borders
    const maxInfluenceRange = 150;
    
    // Calculate influence at each grid point
    for (let y = 0; y < height; y++) {
      for (let x = 0; x < width; x++) {
        const worldX = bounds.minX + x * gridSize;
        const worldY = bounds.minY + y * gridSize;
        
        let playerInfluence = 0;
        let otherInfluence = 0;
        
        // Only check nearby systems for performance
        const nearbySystems = this.systems.filter(system => {
          const deltaX = system.x - worldX;
          const deltaY = system.y - worldY;
          return Math.abs(deltaX) < maxInfluenceRange && Math.abs(deltaY) < maxInfluenceRange;
        });
        
        nearbySystems.forEach(system => {
          const deltaX = system.x - worldX;
          const deltaY = system.y - worldY;
          const distance = Math.sqrt(deltaX * deltaX + deltaY * deltaY);
          
          if (distance < maxInfluenceRange) {
            const influence = Math.max(0, 1 - (distance / maxInfluenceRange));
            const isPlayerOwned = playerSystems.some(ps => ps.id === system.id);
            
            if (isPlayerOwned) {
              playerInfluence += influence * influence;
            } else {
              // Check if it's an enemy system (has colonies)
              if (window.gameState) {
                const planets = window.gameState.getSystemPlanets(system.id);
                const hasColonies = planets.some(planet => planet.colonized_by && planet.colonized_by !== this.currentUserId);
                if (hasColonies) {
                  otherInfluence += influence * influence * 0.6;
                }
              }
            }
          }
        });
        
        // Store net player influence
        field[y][x] = playerInfluence - otherInfluence;
      }
    }
    
    return field;
  }

  extractTerritorialContours(field, bounds, gridSize) {
    const contours = [];
    const height = field.length;
    const width = field[0].length;
    const visited = new Array(height).fill(null).map(() => new Array(width).fill(false));
    
    // Find contour lines where player influence > threshold
    const threshold = 0.2;
    
    for (let y = 0; y < height - 1; y++) {
      for (let x = 0; x < width - 1; x++) {
        if (visited[y][x]) continue;
        
        const current = field[y][x];
        if (current > threshold) {
          // Start tracing a contour
          const contour = this.traceContour(field, x, y, threshold, bounds, gridSize, visited);
          if (contour.length > 4) { // Only keep substantial contours
            contours.push(contour);
          }
        }
      }
    }
    
    return contours;
  }

  traceContour(field, startX, startY, threshold, bounds, gridSize, visited) {
    const contour = [];
    const height = field.length;
    const width = field[0].length;
    
    // Simple flood fill to find territory boundary
    const queue = [{x: startX, y: startY}];
    const territoryPoints = new Set();
    
    while (queue.length > 0) {
      const {x, y} = queue.shift();
      
      if (x < 0 || x >= width || y < 0 || y >= height) continue;
      if (visited[y][x] || field[y][x] <= threshold) continue;
      
      visited[y][x] = true;
      territoryPoints.add(`${x},${y}`);
      
      // Add neighbors
      queue.push({x: x + 1, y}, {x: x - 1, y}, {x, y: y + 1}, {x, y: y - 1});
    }
    
    // Find boundary points
    territoryPoints.forEach(pointStr => {
      const [x, y] = pointStr.split(',').map(Number);
      
      // Check if this point is on the boundary
      const neighbors = [
        {x: x + 1, y}, {x: x - 1, y}, {x, y: y + 1}, {x, y: y - 1}
      ];
      
      const isBoundary = neighbors.some(neighbor => {
        if (neighbor.x < 0 || neighbor.x >= width || neighbor.y < 0 || neighbor.y >= height) {
          return true; // Edge of grid
        }
        return field[neighbor.y][neighbor.x] <= threshold; // Adjacent to non-territory
      });
      
      if (isBoundary) {
        const worldX = bounds.minX + x * gridSize;
        const worldY = bounds.minY + y * gridSize;
        const screenPos = this.worldToScreen(worldX, worldY);
        contour.push({x: screenPos.x, y: screenPos.y, worldX, worldY});
      }
    });
    
    // Sort boundary points to form a proper contour
    return this.orderContourPoints(contour);
  }

  orderContourPoints(points) {
    if (points.length < 3) return points;
    
    // Find center point
    const centerX = points.reduce((sum, p) => sum + p.x, 0) / points.length;
    const centerY = points.reduce((sum, p) => sum + p.y, 0) / points.length;
    
    // Sort by angle from center
    return points.sort((a, b) => {
      const angleA = Math.atan2(a.y - centerY, a.x - centerX);
      const angleB = Math.atan2(b.y - centerY, b.x - centerX);
      return angleA - angleB;
    });
  }

  drawTerritorialContours(contours) {
    if (contours.length === 0) return;
    
    this.ctx.save();
    this.ctx.globalCompositeOperation = 'screen';
    
    contours.forEach(contour => {
      if (contour.length < 3) return;
      
      // Create smooth path
      this.ctx.beginPath();
      this.ctx.moveTo(contour[0].x, contour[0].y);
      
      // Use smooth curves to connect points
      for (let i = 1; i < contour.length; i++) {
        const current = contour[i];
        const next = contour[(i + 1) % contour.length];
        
        // Calculate control point for smooth curve
        const cpX = current.x + (next.x - current.x) * 0.5;
        const cpY = current.y + (next.y - current.y) * 0.5;
        
        this.ctx.quadraticCurveTo(current.x, current.y, cpX, cpY);
      }
      
      // Close the path
      this.ctx.closePath();
      
      // Create gradient fill for border effect
      const bounds = this.getContourBounds(contour);
      const gradient = this.ctx.createRadialGradient(
        bounds.centerX, bounds.centerY, bounds.radius * 0.7,
        bounds.centerX, bounds.centerY, bounds.radius
      );
      
      const baseColor = '34, 197, 94';
      const alpha = Math.max(0.1, 0.25 * this.zoom);
      
      gradient.addColorStop(0, `rgba(${baseColor}, ${alpha * 0.05})`);
      gradient.addColorStop(0.7, `rgba(${baseColor}, ${alpha * 0.3})`);
      gradient.addColorStop(0.9, `rgba(${baseColor}, ${alpha * 0.6})`);
      gradient.addColorStop(1, `rgba(${baseColor}, 0)`);
      
      this.ctx.fillStyle = gradient;
      this.ctx.fill();
      
      // Add border stroke
      this.ctx.strokeStyle = `rgba(${baseColor}, ${alpha * 0.8})`;
      this.ctx.lineWidth = 2 * this.zoom;
      this.ctx.stroke();
    });
    
    this.ctx.restore();
  }

  getContourBounds(contour) {
    let minX = contour[0].x, maxX = contour[0].x;
    let minY = contour[0].y, maxY = contour[0].y;
    
    contour.forEach(point => {
      minX = Math.min(minX, point.x);
      maxX = Math.max(maxX, point.x);
      minY = Math.min(minY, point.y);
      maxY = Math.max(maxY, point.y);
    });
    
    const centerX = (minX + maxX) / 2;
    const centerY = (minY + maxY) / 2;
    const radius = Math.max(maxX - minX, maxY - minY) / 2;
    
    return { centerX, centerY, radius };
  }



  drawFleets(deltaTime) {
    this.fleets.forEach(fleet => {
      let worldX, worldY;
      let isMoving = false;
      let movementAngle = 0;
      
      // Check if fleet is in transit (has both current_system and destination_system)
      if (fleet.destination_system && fleet.destination_system !== "" && 
          fleet.current_system && fleet.current_system !== "" && 
          fleet.destination_system !== fleet.current_system) {
        // Fleet is moving between systems
        const fromSystem = this.systems.find(s => s.id === fleet.current_system);
        const toSystem = this.systems.find(s => s.id === fleet.destination_system);
        
        if (!fromSystem || !toSystem) return;

        // Validate hyperlane range (same as navigation system)
        const deltaX = toSystem.x - fromSystem.x;
        const deltaY = toSystem.y - fromSystem.y;
        const distance = Math.sqrt(deltaX * deltaX + deltaY * deltaY);
        
        // Only allow movement within hyperlane range
        if (distance > 800) return;

        isMoving = true;
        
        // Calculate progress based on ETA
        let progress = 0.5; // Default fallback
        if (fleet.eta) {
          const now = new Date();
          const eta = new Date(fleet.eta);
          const departureTime = new Date(eta.getTime() - (2 * 60 * 1000)); // 2 minute journey
          const totalTime = eta.getTime() - departureTime.getTime();
          const elapsed = now.getTime() - departureTime.getTime();
          progress = Math.max(0, Math.min(1, elapsed / totalTime));
        }

        // Interpolate position along the hyperlane
        worldX = fromSystem.x + (toSystem.x - fromSystem.x) * progress;
        worldY = fromSystem.y + (toSystem.y - fromSystem.y) * progress;
        
        // Calculate movement angle for directional arrow
        movementAngle = Math.atan2(toSystem.y - fromSystem.y, toSystem.x - fromSystem.x);
      } else if (fleet.current_system && fleet.current_system !== "") {
        // Fleet is stationary at current system
        const currentSystem = this.systems.find(s => s.id === fleet.current_system);
        if (!currentSystem) return;
        
        // Offset fleet position slightly from system center to make it visible
        worldX = currentSystem.x + 15;
        worldY = currentSystem.y + 15;
      } else {
        // Fleet has no valid position, skip rendering
        return;
      }

      const screenPos = this.worldToScreen(worldX, worldY);

      if (isMoving) {
        // Draw moving fleet as directional arrow along hyperlane
        this.ctx.fillStyle = '#8b5cf6';
        this.ctx.strokeStyle = '#ffffff';
        this.ctx.lineWidth = 2;

        const size = 12 * this.zoom;
        
        this.ctx.save();
        this.ctx.translate(screenPos.x, screenPos.y);
        this.ctx.rotate(movementAngle);
        
        // Draw arrow pointing in direction of movement
        this.ctx.beginPath();
        this.ctx.moveTo(size, 0);
        this.ctx.lineTo(-size/2, -size/2);
        this.ctx.lineTo(-size/4, 0);
        this.ctx.lineTo(-size/2, size/2);
        this.ctx.closePath();
        this.ctx.fill();
        this.ctx.stroke();
        
        this.ctx.restore();
        
        // Draw pulsing glow effect for moving fleets
        const glowRadius = (15 + Math.sin(Date.now() / 200) * 5) * this.zoom;
        const gradient = this.ctx.createRadialGradient(
          screenPos.x, screenPos.y, 0,
          screenPos.x, screenPos.y, glowRadius
        );
        gradient.addColorStop(0, 'rgba(139, 92, 246, 0.3)');
        gradient.addColorStop(1, 'rgba(139, 92, 246, 0)');
        
        this.ctx.fillStyle = gradient;
        this.ctx.beginPath();
        this.ctx.arc(screenPos.x, screenPos.y, glowRadius, 0, Math.PI * 2);
        this.ctx.fill();
      } else {
        // Draw stationary fleet as triangle
        this.ctx.fillStyle = this.colors.fleet;
        this.ctx.strokeStyle = '#ffffff';
        this.ctx.lineWidth = 1;

        const size = 8 * this.zoom;
        this.ctx.beginPath();
        this.ctx.moveTo(screenPos.x, screenPos.y - size);
        this.ctx.lineTo(screenPos.x - size, screenPos.y + size);
        this.ctx.lineTo(screenPos.x + size, screenPos.y + size);
        this.ctx.closePath();
        this.ctx.fill();
        this.ctx.stroke();
      }

      // Draw fleet name/identifier
      if (this.zoom > 0.5) {
        this.ctx.fillStyle = '#ffffff';
        this.ctx.font = `${10 * this.zoom}px Arial`;
        this.ctx.textAlign = 'center';
        this.ctx.fillText(
          fleet.name || 'Fleet',
          screenPos.x,
          screenPos.y - 16 * this.zoom
        );
      }
    });
  }

  drawUI() {
    // Draw zoom level indicator
    this.ctx.fillStyle = '#ffffff';
    this.ctx.font = '12px monospace';
    this.ctx.textAlign = 'left';
    this.ctx.fillText(`Zoom: ${(this.zoom * 100).toFixed(0)}%`, 10, 25);
    
    // Draw coordinate indicator
    const centerWorld = this.screenToWorld(this.canvas.width / 2, this.canvas.height / 2);
    this.ctx.fillText(
      `Center: ${centerWorld.x.toFixed(0)}, ${centerWorld.y.toFixed(0)}`,
      10,
      45
    );
  }

  // Public methods for updating data
  setSystems(systems) {
    console.log('MapRenderer: Setting systems', systems.length, 'systems');
    this.systems = systems;
    
    // Update planet count cache for system scaling
    this.updateSystemPlanetCounts();
    
    // Auto-center view on first load of systems
    if (!this.initialViewSet && systems.length > 0) {
      this.fitToSystems();
      this.initialViewSet = true;
    }
  }

  setTrades(trades) {
    this.trades = trades || []; // Ensure trades is an array
  }

  // Update planet count cache for system scaling
  updateSystemPlanetCounts() {
    this.systemPlanetCounts.clear();
    if (window.gameState && window.gameState.planets) {
      // Count planets per system
      for (const planet of window.gameState.planets) {
        const systemId = planet.system_id;
        const currentCount = this.systemPlanetCounts.get(systemId) || 0;
        this.systemPlanetCounts.set(systemId, currentCount + 1);
      }
    }
  }

  setLanes(lanes) {
    this.lanes = lanes;
    // Update planet counts when data is refreshed
    this.updateSystemPlanetCounts();
  }

  setFleets(fleets) {
    this.fleets = fleets;
  }

  setCurrentUserId(userId) { // New method
    if (this.currentUserId !== userId) {
      this.currentUserId = userId;
      // Invalidate territorial cache when user changes
      this.territorialCacheKey = null;
    }
  }

  setSelectedSystem(system) {
    this.selectedSystem = system;
    this.updateConnectedSystems();
  }

  // Center view on a specific system with smooth movement
  centerOnSystem(systemId) {
    const system = this.systems.find(s => s.id === systemId);
    if (system) {
      // Set target for smooth camera movement
      this.targetViewX = -system.x;
      this.targetViewY = -system.y;
    }
  }

  // Fit all systems in view
  fitToSystems() {
    if (this.systems.length === 0) return;

    const minX = Math.min(...this.systems.map(s => s.x));
    const maxX = Math.max(...this.systems.map(s => s.x));
    const minY = Math.min(...this.systems.map(s => s.y));
    const maxY = Math.max(...this.systems.map(s => s.y));

    const centerX = (minX + maxX) / 2;
    const centerY = (minY + maxY) / 2;

    // Adjust zoom to fit with generous padding for larger galaxy
    const width = maxX - minX + 500; // More padding for larger galaxy
    const height = maxY - minY + 500;
    const zoomX = this.canvas.width / width;
    const zoomY = this.canvas.height / height;
    this.zoom = Math.min(zoomX, zoomY, this.maxZoom);
    
    // Ensure we don't zoom in too much on initial load
    if (this.zoom > 0.25) {
      this.zoom = 0.25;
    }

    // Center the galaxy in the viewport (calculate after zoom is set)
    this.viewX = -centerX;
    this.viewY = -centerY;
    this.targetViewX = this.viewX;
    this.targetViewY = this.viewY;
  }

  destroy() {
    if (this.animationFrame) {
      cancelAnimationFrame(this.animationFrame);
    }
  }
}