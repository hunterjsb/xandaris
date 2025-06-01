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
      starPlayerOwned: '#00ffea', // Bright cyan/teal for player-owned
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

  showTooltip(system, screenX, screenY) {
    const tooltip = document.getElementById('tooltip');
    if (system && window.gameState) { // Ensure gameState is available
      const planets = window.gameState.getSystemPlanets(system.id);
      const totalSystemPop = planets.reduce((sum, p) => sum + (p.Pop || 0), 0);

      tooltip.innerHTML = `
        <div class="font-semibold">${system.name || `System ${system.id.slice(-4)}`}</div>
        <div class="text-xs">
          <div>Position: ${system.x}, ${system.y}</div>
          <div>Population: ${totalSystemPop.toLocaleString()}</div>
          <div>Owner: ${system.owner_name || 'Uncolonized'}</div>
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
      
      this.clear();
      this.drawBackground();
      this.drawLanes();
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

            const isHovered = this.hoveredTradeRoutes.some(htr => htr.id === trade.id);

            if (isHovered) {
              this.ctx.strokeStyle = this.colors.laneActive; // Use a more prominent color
              this.ctx.lineWidth = 4; // Thicker line for hovered routes
              this.ctx.globalAlpha = 1; // More opaque
            } else {
              this.ctx.strokeStyle = this.colors.lane;
              this.ctx.lineWidth = 2;
              this.ctx.globalAlpha = 0.6; // Default alpha
            }

            this.ctx.beginPath();
            this.ctx.moveTo(fromPos.x, fromPos.y);
            this.ctx.lineTo(toPos.x, toPos.y);
            this.ctx.stroke();
          });

          this.ctx.globalAlpha = 1; // Reset globalAlpha
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
            if (!system.owner_id) {
              color = this.colors.starUnowned;
            } else if (system.owner_id === this.currentUserId) {
              color = this.colors.starPlayerOwned;
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

            if (system.owner_id && this.zoom > 0.6) {
              this.ctx.strokeStyle = this.colors.starOwned;
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

  drawFleets(deltaTime) {
    this.fleets.forEach(fleet => {
      let worldX, worldY;
      
      // Check if fleet is stationary (at current_system) or moving (to destination_system)
      if (fleet.destination_system && fleet.destination_system !== fleet.current_system) {
        // Fleet is moving between systems
        const fromSystem = this.systems.find(s => s.id === fleet.current_system);
        const toSystem = this.systems.find(s => s.id === fleet.destination_system);
        
        if (!fromSystem || !toSystem) return;

        // Calculate progress based on ETA
        let progress = 0.5; // Default fallback
        if (fleet.eta) {
          const now = new Date();
          const eta = new Date(fleet.eta);
          const departureTime = new Date(eta.getTime() - (10 * 60 * 1000)); // Assume 10 minute journey
          const totalTime = eta.getTime() - departureTime.getTime();
          const elapsed = now.getTime() - departureTime.getTime();
          progress = Math.max(0, Math.min(1, elapsed / totalTime));
        }
        
        worldX = fromSystem.x + (toSystem.x - fromSystem.x) * progress;
        worldY = fromSystem.y + (toSystem.y - fromSystem.y) * progress;
      } else {
        // Fleet is stationary at current system
        const currentSystem = this.systems.find(s => s.id === fleet.current_system);
        if (!currentSystem) return;
        
        // Offset fleet position slightly from system center to make it visible
        worldX = currentSystem.x + 15;
        worldY = currentSystem.y + 15;
      }
      
      const screenPos = this.worldToScreen(worldX, worldY);

      // Draw fleet icon (triangle for directional indication)
      this.ctx.fillStyle = this.colors.fleet;
      this.ctx.strokeStyle = '#ffffff';
      this.ctx.lineWidth = 1;
      
      const size = 6 * this.zoom;
      this.ctx.beginPath();
      this.ctx.moveTo(screenPos.x, screenPos.y - size);
      this.ctx.lineTo(screenPos.x - size, screenPos.y + size);
      this.ctx.lineTo(screenPos.x + size, screenPos.y + size);
      this.ctx.closePath();
      this.ctx.fill();
      this.ctx.stroke();

      // Draw fleet name/identifier
      if (this.zoom > 0.5) {
        this.ctx.fillStyle = '#ffffff';
        this.ctx.font = `${Math.floor(10 * this.zoom)}px monospace`;
        this.ctx.textAlign = 'center';
        this.ctx.fillText(
          fleet.name || 'Fleet',
          screenPos.x,
          screenPos.y - 12 * this.zoom
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
    this.currentUserId = userId;
  }

  setSelectedSystem(system) {
    this.selectedSystem = system;
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