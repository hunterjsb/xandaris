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
    
    // View settings
    this.viewX = 0;
    this.viewY = 0;
    this.zoom = 1;
    this.maxZoom = 3;
    this.minZoom = 0.3;
    
    // Deep Space Colors
    this.colors = {
      background: '#000508',
      star: '#4080ff',        // Nebula blue for unowned
      starOwned: '#f1a9ff',   // Plasma pink for owned 
      starEnemy: '#ff6b6b',   // Bright red for enemies
      starNeutral: '#8cb3ff', // Lighter blue for neutral
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
          this.selectSystem(clickedSystem);
        } else {
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
    this.selectedSystem = system;
    // Emit custom event for UI to handle, now including planets in that system
    const planetsInSystem = window.gameState.getSystemPlanets(system.id);
    this.canvas.dispatchEvent(new CustomEvent('systemSelected', {
      detail: { system, planets: planetsInSystem }
    }));
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

            let color = this.colors.star;
            if (system.owner_id) {
              color = this.colors.starOwned;
            }

            const baseSystemDrawRadius = 6 * this.zoom;
            const systemDrawRadius = baseSystemDrawRadius * scaleFactor;
            const glowDrawRadius = (20 * this.zoom) * scaleFactor;

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
              const time = Date.now() * 0.005;
              const pulseRadius = (12 + Math.sin(time) * 2) * this.zoom * scaleFactor;

              this.ctx.strokeStyle = this.colors.selection;
              this.ctx.lineWidth = 2;
              this.ctx.globalAlpha = 0.8 + Math.sin(time) * 0.2;
              this.ctx.beginPath();
              this.ctx.arc(screenPos.x, screenPos.y, pulseRadius, 0, Math.PI * 2);
              this.ctx.stroke();
              this.ctx.globalAlpha = 1;
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
      // Calculate fleet position based on progress
      const fromSystem = this.systems.find(s => s.id === fleet.from_id);
      const toSystem = this.systems.find(s => s.id === fleet.to_id);
      
      if (!fromSystem || !toSystem) return;

      // TODO: Calculate actual progress based on ETA
      const progress = 0.5; // Placeholder
      const worldX = fromSystem.x + (toSystem.x - fromSystem.x) * progress;
      const worldY = fromSystem.y + (toSystem.y - fromSystem.y) * progress;
      
      const screenPos = this.worldToScreen(worldX, worldY);

      // Draw fleet
      this.ctx.fillStyle = this.colors.fleet;
      this.ctx.beginPath();
      this.ctx.arc(screenPos.x, screenPos.y, 4 * this.zoom, 0, Math.PI * 2);
      this.ctx.fill();

      // Draw strength indicator
      if (this.zoom > 0.7) {
        this.ctx.fillStyle = '#ffffff';
        this.ctx.font = `${Math.floor(8 * this.zoom)}px monospace`;
        this.ctx.textAlign = 'center';
        this.ctx.fillText(
          fleet.strength.toString(),
          screenPos.x,
          screenPos.y - 8 * this.zoom
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
  }

  setTrades(trades) {
    this.trades = trades || []; // Ensure trades is an array
  }

  setLanes(lanes) {
    this.lanes = lanes;
  }

  setFleets(fleets) {
    this.fleets = fleets;
  }

  setSelectedSystem(system) {
    this.selectedSystem = system;
  }

  // Center view on a specific system
  centerOnSystem(systemId) {
    const system = this.systems.find(s => s.id === systemId);
    if (system) {
      this.viewX = -system.x;
      this.viewY = -system.y;
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

    this.viewX = -centerX;
    this.viewY = -centerY;

    // Adjust zoom to fit
    const width = maxX - minX + 100; // Add padding
    const height = maxY - minY + 100;
    const zoomX = this.canvas.width / width;
    const zoomY = this.canvas.height / height;
    this.zoom = Math.min(zoomX, zoomY, this.maxZoom);
  }

  destroy() {
    if (this.animationFrame) {
      cancelAnimationFrame(this.animationFrame);
    }
  }
}