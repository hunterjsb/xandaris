// Canvas-based map renderer for the 4X game
export class MapRenderer {
  constructor(canvasId) {
    this.canvas = document.getElementById(canvasId);
    this.ctx = this.canvas.getContext('2d');
    this.systems = [];
    this.lanes = [];
    this.fleets = [];
    this.selectedSystem = null;

    // Player and diplomacy info
    this.currentPlayerId = null;
    this.allyIds = [];
    this.enemyIds = [];
    
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
      starOwned: '#f1a9ff',   // Plasma pink for player-owned
      starAlly: '#50fa7b',    // Friendly green for allies
      starEnemy: '#ff6b6b',   // Bright red for enemies
      starNeutral: '#8cb3ff', // Lighter blue for neutral/other players
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
    this.currentTick = 0; // Initialize currentTick
    
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
        // Show tooltip for systems
        const worldPos = this.screenToWorld(e.offsetX, e.offsetY);
        const hoveredSystem = this.getSystemAt(worldPos.x, worldPos.y);
        this.showTooltip(hoveredSystem, e.offsetX, e.offsetY);
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
    // Emit custom event for UI to handle
    this.canvas.dispatchEvent(new CustomEvent('systemSelected', {
      detail: { system }
    }));
  }

  showTooltip(system, screenX, screenY) {
    const tooltip = document.getElementById('tooltip');
    if (system) {
      tooltip.innerHTML = `
        <div class="font-semibold">${system.name || `System ${system.id}`}</div>
        <div class="text-xs">
          <div>Position: ${system.x}, ${system.y}</div>
          <div>Population: ${system.pop || 0}</div>
          <div>Owner: ${system.owner_name || 'Uncolonized'}</div>
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

  drawSystems() {
    this.systems.forEach(system => {
      const screenPos = this.worldToScreen(system.x, system.y);
      
      // Skip if outside visible area
      if (screenPos.x < -50 || screenPos.x > this.canvas.width + 50 ||
          screenPos.y < -50 || screenPos.y > this.canvas.height + 50) {
        return;
      }

      // Determine system color based on ownership
      let color = this.colors.star; // Default to unowned
      const ownerId = system.owner_id;

      if (ownerId) {
        if (ownerId === this.currentPlayerId) {
          color = this.colors.starOwned;
        } else if (this.allyIds.includes(ownerId)) {
          color = this.colors.starAlly;
        } else if (this.enemyIds.includes(ownerId)) {
          color = this.colors.starEnemy;
        } else {
          color = this.colors.starNeutral; // Owned by another player (not specifically ally or enemy)
        }
      }

      const radius = 6 * this.zoom;
      const glowRadius = 20 * this.zoom;

      // Draw star glow effect
      const gradient = this.ctx.createRadialGradient(
        screenPos.x, screenPos.y, 0,
        screenPos.x, screenPos.y, glowRadius
      );
      gradient.addColorStop(0, color + '40'); // 25% opacity
      gradient.addColorStop(0.3, color + '20'); // 12% opacity  
      gradient.addColorStop(1, color + '00'); // 0% opacity
      
      this.ctx.fillStyle = gradient;
      this.ctx.beginPath();
      this.ctx.arc(screenPos.x, screenPos.y, glowRadius, 0, Math.PI * 2);
      this.ctx.fill();

      // Draw main star body with inner glow
      const innerGradient = this.ctx.createRadialGradient(
        screenPos.x, screenPos.y, 0,
        screenPos.x, screenPos.y, radius
      );
      innerGradient.addColorStop(0, '#ffffff');
      innerGradient.addColorStop(0.7, color);
      innerGradient.addColorStop(1, color + 'cc'); // 80% opacity
      
      this.ctx.fillStyle = innerGradient;
      this.ctx.beginPath();
      this.ctx.arc(screenPos.x, screenPos.y, radius, 0, Math.PI * 2);
      this.ctx.fill();

      // Add sparkle effect for owned systems
      if (system.owner_id && this.zoom > 0.6) {
        this.ctx.strokeStyle = this.colors.starOwned;
        this.ctx.lineWidth = 1;
        this.ctx.globalAlpha = 0.8;
        
        // Draw cross sparkles
        this.ctx.beginPath();
        this.ctx.moveTo(screenPos.x - radius * 1.5, screenPos.y);
        this.ctx.lineTo(screenPos.x + radius * 1.5, screenPos.y);
        this.ctx.moveTo(screenPos.x, screenPos.y - radius * 1.5);
        this.ctx.lineTo(screenPos.x, screenPos.y + radius * 1.5);
        this.ctx.stroke();
        
        this.ctx.globalAlpha = 1;
      }

      // Draw selection ring with pulsing effect
      if (this.selectedSystem && this.selectedSystem.id === system.id) {
        const time = Date.now() * 0.005;
        const pulseRadius = (12 + Math.sin(time) * 2) * this.zoom;
        
        this.ctx.strokeStyle = this.colors.selection;
        this.ctx.lineWidth = 2;
        this.ctx.globalAlpha = 0.8 + Math.sin(time) * 0.2;
        this.ctx.beginPath();
        this.ctx.arc(screenPos.x, screenPos.y, pulseRadius, 0, Math.PI * 2);
        this.ctx.stroke();
        this.ctx.globalAlpha = 1;
      }

      // Draw system name with glow
      if (this.zoom > 0.8) {
        const fontSize = Math.floor(11 * this.zoom);
        this.ctx.font = `${fontSize}px monospace`;
        this.ctx.textAlign = 'center';
        
        // Text shadow/glow
        this.ctx.fillStyle = 'rgba(0, 0, 0, 0.8)';
        this.ctx.fillText(
          system.name || `S${system.id.slice(-3)}`,
          screenPos.x + 1,
          screenPos.y - 15 * this.zoom + 1
        );
        
        // Main text
        this.ctx.fillStyle = 'rgba(255, 255, 255, 0.95)';
        this.ctx.fillText(
          system.name || `S${system.id.slice(-3)}`,
          screenPos.x,
          screenPos.y - 15 * this.zoom
        );
      }

      // Draw population indicator with plasma styling
      if (system.pop > 0 && this.zoom > 0.5) {
        const popFontSize = Math.floor(9 * this.zoom);
        this.ctx.font = `${popFontSize}px monospace`;
        this.ctx.textAlign = 'center';
        
        // Population background
        this.ctx.fillStyle = 'rgba(241, 169, 255, 0.2)';
        this.ctx.fillRect(
          screenPos.x - 8 * this.zoom,
          screenPos.y + 12 * this.zoom,
          16 * this.zoom,
          12 * this.zoom
        );
        
        // Population text
        this.ctx.fillStyle = '#f1a9ff';
        this.ctx.fillText(
          system.pop.toString(),
          screenPos.x,
          screenPos.y + 21 * this.zoom
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

      let progress = 0.5; // Default placeholder
      const { departure_tick, eta_tick } = fleet;

      if (typeof departure_tick === 'undefined' || departure_tick === null) {
        // console.warn(`Fleet ${fleet.id} missing departure_tick, using placeholder progress.`);
        // This warning can be noisy, enable if specifically debugging this.
        progress = 0.5;
      } else if (eta_tick <= departure_tick) {
        progress = 0.5; // Or 0 if it's an invalid state (arrived before departure)
      } else if (this.currentTick >= eta_tick) {
        progress = 1.0; // Arrived or overdue
      } else if (this.currentTick < departure_tick) {
        progress = 0.0; // Not yet departed
      } else {
        const totalJourneyTicks = eta_tick - departure_tick;
        const ticksElapsed = this.currentTick - departure_tick;
        progress = ticksElapsed / totalJourneyTicks;
      }

      progress = Math.max(0, Math.min(1, progress)); // Clamp progress between 0 and 1

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

  // Methods to update player/diplomacy info
  setCurrentPlayer(playerId) {
    this.currentPlayerId = playerId;
  }

  setAllies(allyIdsArray) {
    this.allyIds = allyIdsArray;
  }

  setEnemies(enemyIdsArray) {
    this.enemyIds = enemyIdsArray;
  }

  setCurrentTick(tick) {
    this.currentTick = tick;
  }
}