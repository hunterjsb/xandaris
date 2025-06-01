// UI Controller for handling all UI interactions and updates
export class UIController {
  constructor() {
    this.currentUser = null;
    this.gameState = null;
    this.tickTimer = null;
    this.currentSystemId = null; // Track current system/planet ID being viewed
    this.planetTypes = new Map(); // Store planet types with their icons
    this.pb = null; // Will be set when PocketBase is available

    // Make instance available globally, for event handlers in dynamically created HTML
    window.uiController = this;
    this.expandedView = document.getElementById("expanded-view-container");
    if (this.expandedView) {
        // Ensure it starts hidden and styled as a panel (though class will be set on show)
        this.expandedView.classList.add('hidden', 'floating-panel');
        this.expandedView.style.left = '-2000px'; // Start off-screen
        this.expandedView.style.top = '-2000px';
    } else {
        console.error("#expanded-view-container not found during UIController construction");
    }
  }

  setPocketBase(pb) {
    this.pb = pb;
    this.loadPlanetTypes(); // Load planet types when PocketBase is available
  }

  async loadPlanetTypes() {
    try {
      if (!this.pb) return; // PocketBase not initialized yet
      const response = await this.pb.collection('planet_types').getFullList();
      this.planetTypes.clear();
      
      response.forEach(type => {
        // Store by both name and ID for lookup flexibility
        const typeData = {
          name: type.name,
          icon: type.icon || '' // URL to icon image
        };
        this.planetTypes.set(type.name.toLowerCase(), typeData);
        this.planetTypes.set(type.id, typeData);
      });
    } catch (error) {
      console.warn('Failed to load planet types:', error);
    }
  }

  getPlanetTypeIcon(planetTypeName) {
    if (!planetTypeName) return '<img src="/placeholder-planet.png" class="w-6 h-6" alt="Unknown planet type" />';
    
    // Try lookup by ID first, then by name (lowercase)
    let planetType = this.planetTypes.get(planetTypeName);
    if (!planetType) {
      planetType = this.planetTypes.get(planetTypeName.toLowerCase());
    }
    
    if (planetType && planetType.icon) {
      return `<img src="${planetType.icon}" class="w-6 h-6" alt="${planetType.name}" />`;
    }
    
    // Fallback if no icon URL is set
    return '<img src="/placeholder-planet.png" class="w-6 h-6" alt="Unknown planet type" />';
  }

  getPlanetTypeName(planetTypeId) {
    if (!planetTypeId) return 'Unknown';
    
    // Try lookup by ID first, then by name (lowercase)
    let planetType = this.planetTypes.get(planetTypeId);
    if (!planetType) {
      planetType = this.planetTypes.get(planetTypeId.toLowerCase());
    }
    
    return planetType ? planetType.name : 'Unknown';
  }

  getResourceIcons(planet) {
    const icons = [];
    
    // Show resource availability based on planet type and actual resources
    if (planet.Credits > 0 || planet.planet_type === 'terrestrial') {
      icons.push('<span class="material-icons text-yellow-400" title="Credits">account_balance_wallet</span>');
    }
    if (planet.Food > 0 || planet.planet_type === 'terrestrial' || planet.planet_type === 'gaia') {
      icons.push('<span class="material-icons text-green-400" title="Food">restaurant</span>');
    }
    if (planet.Ore > 0 || planet.planet_type === 'volcanic' || planet.planet_type === 'barren') {
      icons.push('<span class="material-icons text-gray-400" title="Ore">construction</span>');
    }
    if (planet.Goods > 0 || planet.planet_type === 'terrestrial') {
      icons.push('<span class="material-icons text-orange-400" title="Goods">inventory_2</span>');
    }
    
    return icons;
  }

  getResourceIcons(planet) {
    const icons = [];
    
    // Show resource availability based on planet type and actual resources
    if (planet.Credits > 0 || planet.planet_type === 'terrestrial') {
      icons.push('<span class="material-icons text-yellow-400" title="Credits">account_balance_wallet</span>');
    }
    if (planet.Food > 0 || planet.planet_type === 'terrestrial' || planet.planet_type === 'gaia') {
      icons.push('<span class="material-icons text-green-400" title="Food">restaurant</span>');
    }
    if (planet.Ore > 0 || planet.planet_type === 'volcanic' || planet.planet_type === 'barren') {
      icons.push('<span class="material-icons text-gray-400" title="Ore">construction</span>');
    }
    if (planet.Goods > 0 || planet.planet_type === 'terrestrial') {
      icons.push('<span class="material-icons text-orange-400" title="Goods">inventory_2</span>');
    }
    
    return icons;
  }

  clearExpandedView() {
    if (this.expandedView) {
      // Clean up drag event listeners if they exist
      if (this.expandedView._dragCleanup) {
        this.expandedView._dragCleanup();
        delete this.expandedView._dragCleanup;
      }
      
      // Don't clear innerHTML here, as content might be reused or fade out.
      // Content replacement will happen in displaySystemView/displayPlanetView.
      this.expandedView.classList.add("hidden");
      // Move it off-screen to prevent interaction and ensure it's visually gone
      this.expandedView.style.left = '-2000px';
      this.expandedView.style.top = '-2000px';
      this.expandedView.style.right = 'auto';
    }
    this.currentSystemId = null;
  }

  positionPanel(container, screenX, screenY) {
    // Ensure container is not hidden to get accurate dimensions
    const wasHidden = container.classList.contains('hidden');
    if (wasHidden) {
      container.classList.remove('hidden');
      // Temporarily make it visible but off-screen to measure
      container.style.left = '-9999px';
      container.style.top = '-9999px';
    }

    const panelWidth = container.offsetWidth;
    const panelHeight = container.offsetHeight;
    const viewportWidth = window.innerWidth;
    const viewportHeight = window.innerHeight;
    const margin = 15; // Margin from system icon and viewport edges

    let top = screenY + margin;
    let left = screenX + margin;

    // Adjust if too close to right edge
    if (left + panelWidth + margin > viewportWidth) {
      left = screenX - panelWidth - margin; // Position to the left of the cursor
    }
    // Adjust if too close to bottom edge
    if (top + panelHeight + margin > viewportHeight) {
      top = viewportHeight - panelHeight - margin; // Align to bottom edge
    }
    // Adjust if too close to left edge (after potential flip)
    if (left < margin) {
      left = margin; // Align to left edge
    }
    // Adjust if too close to top edge
    if (top < margin) {
      top = margin; // Align to top edge
    }

    container.style.left = `${left}px`;
    container.style.top = `${top}px`;

    if (wasHidden && (container.style.left === '-9999px')) {
      // If it was temp unhidden and not positioned, re-hide. Should not happen if positionPanel is called correctly.
      // This case is unlikely if called after content is set and ready to be shown.
      container.classList.add('hidden');
    }
  }

  displaySystemView(system, planets, screenX, screenY) {
    const container = this.expandedView;
    if (!container) {
      console.error("#expanded-view-container not found in displaySystemView");
      return;
    }

    // If it's the same system and the panel is already visible, and no new coords are given,
    // we might not need to do anything, or just update content if planets data changed.
    // For now, this simplified check allows re-rendering if called.
    if (this.currentSystemId === system.id && !container.classList.contains("hidden") && screenX === undefined) {
        // If planets data could change, we might need to proceed and update planet list.
        // For now, assume if it's the same system and visible, content is up to date unless new position is given.
        // This could be more sophisticated by checking if 'planets' data has actually changed.
      // return; // Potentially skip if no new position and system is same.
    }

    container.className = 'floating-panel'; // Ensure base class is set, removes 'hidden' implicitly if it was only 'hidden'
    

    this.currentSystemId = system.id;


    
    // Calculate system stats
    let totalPopulation = 0;
    let colonizedCount = 0;
    let ownedByPlayer = 0;
    const currentUserId = this.currentUser?.id;
    
    if (planets && planets.length > 0) {
      planets.forEach(planet => {
        if (planet.colonized_by) {
          colonizedCount++;
          if (planet.colonized_by === currentUserId) {
            ownedByPlayer++;
          }
        }
        totalPopulation += planet.Pop || 0;
      });
    }
    
    let planetsHtml = '<div class="text-sm text-space-400">No planets detected in this system.</div>';
    if (planets && planets.length > 0) {
      planetsHtml = planets.map(planet => {
        const planetName = planet.name || `Planet ${planet.id.slice(-4)}`;
        const planetTypeValue = planet.planet_type || planet.type;
        const planetIcon = this.getPlanetTypeIcon(planetTypeValue);
        const planetTypeName = this.getPlanetTypeName(planetTypeValue);
        
        const isOwned = planet.colonized_by === currentUserId;
        const population = planet.Pop || 0;
        const maxPop = planet.MaxPopulation || 'N/A';
        
        // Status indicators
        let statusHtml = '';
        if (planet.colonized_by) {
          if (isOwned) {
            statusHtml = `<span class="text-xs text-green-400 flex items-center gap-1"><span class="material-icons text-xs">check_circle</span>Your Colony</span>`;
          } else {
            statusHtml = `<span class="text-xs text-red-400 flex items-center gap-1"><span class="material-icons text-xs">block</span>${planet.colonized_by_name || 'Occupied'}</span>`;
          }
        } else {
          statusHtml = `<span class="text-xs text-gray-400 flex items-center gap-1"><span class="material-icons text-xs">radio_button_unchecked</span>Uncolonized</span>`;
        }
        
        // Resource icons preview
        const resourceIcons = this.getResourceIcons(planet);
        let resourcesPreview = '';
        if (resourceIcons.length > 0) {
          resourcesPreview = `
            <div class="mt-2 flex gap-1 items-center">
              ${resourceIcons.join('')}
            </div>
          `;
        }
        
        return `
          <li class="mb-2 p-3 bg-space-700 hover:bg-space-600 rounded-md cursor-pointer transition-all duration-200 border border-transparent hover:border-space-500"
              onclick="window.uiController.displayPlanetView(JSON.parse(decodeURIComponent('${encodeURIComponent(JSON.stringify(planet))}')))">
            <div class="flex items-start justify-between">
              <div class="flex-1">
                <div class="flex items-center gap-2">
                  <div class="flex items-center justify-center w-8 h-8">${planetIcon}</div>
                  <div>
                    <div class="font-semibold">${planetName}</div>
                    <div class="text-xs text-space-300">${planetTypeName} • Size ${planet.size || 'N/A'}</div>
                  </div>
                </div>
                ${resourcesPreview}
              </div>
              <div class="text-right">
                ${statusHtml}
              </div>
            </div>
          </li>
        `;
      }).join("");
    }

    // Determine if a full redraw of innerHTML is needed or if specific parts can be updated.
    // For simplicity in this refactor, we'll redraw the inner content structure.
    // More advanced would be to diff content if system.id is the same.
    container.innerHTML = `
      <div class="floating-panel-content">
        <div class="panel-header flex justify-between items-center p-3 cursor-move border-b border-space-700/50" draggable="false">
          <div class="flex items-center gap-2">
            <span class="material-icons text-space-400 drag-handle">drag_indicator</span>
            <h2 id="system-name" class="text-xl font-bold text-orange-300"></h2>
          </div>
          <button onclick="window.uiController.clearExpandedView()"
                  class="btn-icon hover:bg-space-700 rounded">
            <span class="material-icons text-sm">close</span>
          </button>
        </div>
        <div class="p-4">
          <div class="mb-4">
            <div id="system-coords" class="text-center p-3 bg-gradient-to-r from-nebula-900/30 to-plasma-900/30 rounded-lg border border-nebula-600/20"></div>
          </div>

          <div class="flex-1 overflow-hidden flex flex-col">
            <h3 class="text-lg font-semibold mb-2 text-nebula-200">Planets in System</h3>
            <ul id="system-planets-list" class="flex-1 overflow-y-auto pr-2 custom-scrollbar">
            </ul>
          </div>
        </div>
      </div>
      `;

    // Update dynamic content - system name now handled in coords section
    container.querySelector("#system-coords").innerHTML = `
      <div class="flex items-center justify-between">
        <span class="font-semibold text-nebula-200">${system.name || `System ${system.id.slice(-4)}`}</span>
        <span class="font-mono text-sm text-gray-500">${system.x}, ${system.y}</span>
      </div>
    `;

    const planetsListUl = container.querySelector("#system-planets-list"); // Query within the new innerHTML
    this.updatePlanetList(planetsListUl, planets, currentUserId);

    container.dataset.viewType = 'system'; // For potential future logic
    container.dataset.currentId = system.id;

    // Position the panel
    container.classList.remove('hidden');
    if (screenX !== undefined && screenY !== undefined) {
        this.positionPanel(container, screenX, screenY);
    } else if (container.style.left === '-2000px' || container.style.left === '-9999px' || !container.style.left) {
        // If it was off-screen or no position set, place it default
        container.style.top = '20px';
        container.style.left = '20px';
        container.style.right = 'auto';
    }
    
    // Always re-add drag functionality after content recreation
    this.makePanelDraggable(container);
  }

  makePanelDraggable(container) {
    const header = container.querySelector('.panel-header');
    if (!header) return;

    // Clean up any existing drag handlers first
    if (container._dragCleanup) {
      container._dragCleanup();
    }

    let isDragging = false;
    let currentX;
    let currentY;
    let initialX;
    let initialY;
    let xOffset = 0;
    let yOffset = 0;

    const dragStart = (e) => {
      // Only start drag if clicking on the header or drag handle, but not close button
      if (e.target.closest('.panel-header') && !e.target.closest('button')) {
        // Get current position from computed style
        const rect = container.getBoundingClientRect();
        currentX = rect.left;
        currentY = rect.top;
        
        if (e.type === "touchstart") {
          initialX = e.touches[0].clientX - currentX;
          initialY = e.touches[0].clientY - currentY;
        } else {
          initialX = e.clientX - currentX;
          initialY = e.clientY - currentY;
        }

        isDragging = true;
        container.style.transition = 'none';
        container.style.right = 'auto'; // Clear right positioning
        header.style.cursor = 'grabbing';
        e.preventDefault();
      }
    };

    const dragEnd = () => {
      if (isDragging) {
        isDragging = false;
        container.style.transition = '';
        header.style.cursor = 'move';
      }
    };

    const drag = (e) => {
      if (isDragging) {
        e.preventDefault();

        if (e.type === "touchmove") {
          currentX = e.touches[0].clientX - initialX;
          currentY = e.touches[0].clientY - initialY;
        } else {
          currentX = e.clientX - initialX;
          currentY = e.clientY - initialY;
        }

        // Constrain to viewport
        const rect = container.getBoundingClientRect();
        const maxX = window.innerWidth - rect.width;
        const maxY = window.innerHeight - rect.height;
        
        currentX = Math.max(0, Math.min(currentX, maxX));
        currentY = Math.max(0, Math.min(currentY, maxY));

        container.style.left = `${currentX}px`;
        container.style.top = `${currentY}px`;
      }
    };

    // Mouse events
    header.addEventListener('mousedown', dragStart);
    document.addEventListener('mousemove', drag);
    document.addEventListener('mouseup', dragEnd);

    // Touch events for mobile
    header.addEventListener('touchstart', dragStart);
    document.addEventListener('touchmove', drag);
    document.addEventListener('touchend', dragEnd);

    // Clean up on panel removal
    const cleanup = () => {
      header.removeEventListener('mousedown', dragStart);
      document.removeEventListener('mousemove', drag);
      document.removeEventListener('mouseup', dragEnd);
      header.removeEventListener('touchstart', dragStart);
      document.removeEventListener('touchmove', drag);
      document.removeEventListener('touchend', dragEnd);
    };

    // Store cleanup function for later use
    container._dragCleanup = cleanup;
  }

  updatePlanetList(ulElement, planets, currentUserId) {
    const existingPlanetElements = new Map();
    ulElement.querySelectorAll("li[data-planet-id]").forEach(li => {
      existingPlanetElements.set(li.dataset.planetId, li);
    });

    if (!planets || planets.length === 0) {
      ulElement.innerHTML = '<div class="text-sm text-space-400">No planets detected in this system.</div>';
      return;
    }

    let focusedElement = document.activeElement;
    let focusedPlanetId = focusedElement && focusedElement.closest('li[data-planet-id]') ? focusedElement.closest('li[data-planet-id]').dataset.planetId : null;
    let selectionStart, selectionEnd;
    if (focusedElement && focusedElement.tagName === 'INPUT' || focusedElement.tagName === 'TEXTAREA') {
        selectionStart = focusedElement.selectionStart;
        selectionEnd = focusedElement.selectionEnd;
    }


    planets.forEach(planet => {
      const planetId = planet.id;
      const listItem = existingPlanetElements.get(planetId) || document.createElement("li");
      listItem.dataset.planetId = planetId; // Ensure ID is set for new items

      const planetName = planet.name || `Planet ${planet.id.slice(-4)}`;
      const planetTypeValue = planet.planet_type || planet.type;
      const planetIcon = this.getPlanetTypeIcon(planetTypeValue);
      const planetTypeName = this.getPlanetTypeName(planetTypeValue);
      const isOwned = planet.colonized_by === currentUserId;
      const population = planet.Pop || 0;
      const maxPop = planet.MaxPopulation || 'N/A';

      let statusHtml = '';
      if (planet.colonized_by) {
        if (isOwned) {
          statusHtml = `<span class="text-xs text-green-400 flex items-center gap-1"><span class="material-icons text-xs">check_circle</span>Your Colony</span>`;
        } else {
          statusHtml = `<span class="text-xs text-red-400 flex items-center gap-1"><span class="material-icons text-xs">block</span>${planet.colonized_by_name || 'Occupied'}</span>`;
        }
      } else {
        statusHtml = `<span class="text-xs text-gray-400 flex items-center gap-1"><span class="material-icons text-xs">radio_button_unchecked</span>Uncolonized</span>`;
      }

      // Resource icons preview
      const resourceIcons = this.getResourceIcons(planet);
      let resourcesPreview = '';
      if (resourceIcons.length > 0) {
        resourcesPreview = `
          <div class="mt-2 flex gap-1 items-center">
            ${resourceIcons.join('')}
          </div>
        `;
      }

      // It's generally better to update specific parts of the listItem's innerHTML
      // or use more targeted DOM manipulation if performance becomes an issue here.
      listItem.className = "mb-2 p-3 bg-space-700 hover:bg-space-600 rounded-md transition-all duration-200 border border-transparent hover:border-space-500";

      let colonizeButtonHtml = '';
      
      // Check if player has any fleets with settler ships at this system
      const availableSettlerFleets = this.getAvailableSettlerFleets(planet.system_id);
      const canPlayerColonize = !planet.colonized_by &&
                                this.currentUser &&
                                availableSettlerFleets.length > 0;

      if (canPlayerColonize) {
        colonizeButtonHtml = `
          <button class="btn btn-success btn-sm py-1 px-2 text-xs mt-2"
                  onclick="event.stopPropagation(); console.log('Colonize button clicked for planet: ${planet.id}'); window.uiController.colonizePlanetWrapper('${planet.id}');">
            Colonize (Settler Ship)
          </button>
        `;
      } else if (!planet.colonized_by) {
        // Show disabled button if no settler ships available
        colonizeButtonHtml = `
          <button class="btn btn-disabled btn-sm py-1 px-2 text-xs mt-2" disabled>
            Colonize (Need Settler Ship)
          </button>
        `;
      }

      listItem.innerHTML = `
        <div class="flex items-start justify-between">
          <div class="flex-1 ${isOwned || planet.colonized_by || !canPlayerColonize ? 'cursor-pointer' : ''}"
               onclick="${isOwned || planet.colonized_by || !canPlayerColonize ? `window.uiController.displayPlanetView(JSON.parse(decodeURIComponent('${encodeURIComponent(JSON.stringify(planet))}')))` : ''}">
            <div class="flex items-center gap-2">
              <div class="flex items-center justify-center w-8 h-8">${planetIcon}</div>
              <div>
                <div class="font-semibold">${planetName}</div>
                <div class="text-xs text-space-300">${planetTypeName} • Size ${planet.size || 'N/A'}</div>
              </div>
            </div>
            ${resourcesPreview}
          </div>
          <div class="text-right flex flex-col items-end">
            ${statusHtml}
            ${colonizeButtonHtml}
          </div>
        </div>
      `;

      // If the planet is not colonized AND the player can colonize it,
      // the main list item can also be made clickable for colonization for a larger target area.
      if (!planet.colonized_by && canPlayerColonize) {
        listItem.classList.add('cursor-pointer');
        listItem.onclick = (e) => {
            // Prevent click from bubbling if the click was on the button itself
            if (e.target.closest('button')) return;
            console.log('Colonize (list item) clicked for planet: ${planet.id}');
            window.uiController.colonizePlanetWrapper(planet.id);
        };
      } else {
        // If already colonized, or cannot be colonized by player, remove general click handler
        // or ensure it only navigates (handled by the inner div's onclick now).
        listItem.onclick = null;
        // The inner div with class 'flex-1' handles the click to displayPlanetView for non-colonizable/owned planets.
      }

      if (!existingPlanetElements.has(planetId)) {
        ulElement.appendChild(listItem);
      }
      existingPlanetElements.delete(planetId); // Remove from map as it's processed
    });

    // Remove old planet elements that are no longer in the list
    existingPlanetElements.forEach(li => li.remove());

    if (focusedPlanetId) {
        const newFocusedElement = ulElement.querySelector(`li[data-planet-id="${focusedPlanetId}"]`);
        if (newFocusedElement) {
            let elementToFocus = newFocusedElement.querySelector('input, textarea, button, [tabindex="0"]') || newFocusedElement;
            elementToFocus.focus();
            if (elementToFocus.setSelectionRange && selectionStart !== undefined && selectionEnd !== undefined) {
                elementToFocus.setSelectionRange(selectionStart, selectionEnd);
            }
        }
    }
  }


  displayPlanetView(planet, screenX, screenY) { // Added screenX, screenY
    const container = this.expandedView;
    if (!container) {
      console.error("#expanded-view-container not found in displayPlanetView");
      return;
    }

    // Simplified check: if it's the same planet and no new coords, assume no major update needed.
    // Could be more sophisticated by checking if 'planet' data has changed.
    if (this.currentSystemId === planet.id && container.dataset.viewType === 'planet' && !container.classList.contains("hidden") && screenX === undefined) {
      // return;
    }

    container.className = 'floating-panel'; // Ensure base class is set

    // Use currentSystemId to track the currently displayed entity (system or planet)
    // This might be better named e.g. currentViewEntityId if it can be a planet ID too.
    // For now, reusing currentSystemId for simplicity, assuming planet IDs are distinct from system IDs
    // or that context (system view vs planet view) is managed by viewType.
    this.currentSystemId = planet.id;
    container.dataset.viewType = 'planet';
    container.dataset.currentId = planet.id;

    const planetName = planet.name || `Planet ${planet.id.slice(-4)}`;
    const planetTypeValue = planet.planet_type || planet.type;
    const planetIcon = this.getPlanetTypeIcon(planetTypeValue);
    const planetTypeName = this.getPlanetTypeName(planetTypeValue);
    const systemName = planet.system_name || (this.gameState && this.gameState.mapData.systems.find(s => s.id === planet.system_id)?.name) || planet.system_id;

    // Calculate population percentage
    const popPercentage = planet.MaxPopulation ? Math.round((planet.Pop || 0) / planet.MaxPopulation * 100) : 0;
    const popBarColor = popPercentage > 80 ? 'bg-green-500' : popPercentage > 50 ? 'bg-yellow-500' : 'bg-orange-500';

    let resourcesHtml = '<div class="text-sm text-space-400">No resource data available.</div>';
    if (planet.Credits !== undefined) {
      resourcesHtml = `
        <div class="space-y-3">
          <!-- Population Bar -->
          <div>
            <div class="flex justify-between items-center mb-1">
              <span class="text-sm flex items-center gap-1"><span class="material-icons text-sm">people</span>Population</span>
              <span class="text-sm font-semibold">${planet.Pop?.toLocaleString() || 0} / ${planet.MaxPopulation?.toLocaleString() || 'N/A'}</span>
            </div>
            <div class="w-full bg-space-700 rounded-full h-2">
              <div class="${popBarColor} h-2 rounded-full transition-all duration-300" style="width: ${popPercentage}%"></div>
            </div>
          </div>
          
          <!-- Morale Bar -->
          <div>
            <div class="flex justify-between items-center mb-1">
              <span class="text-sm flex items-center gap-1"><span class="material-icons text-sm">sentiment_satisfied</span>Morale</span>
              <span class="text-sm font-semibold">${planet.Morale || 0}%</span>
            </div>
            <div class="w-full bg-space-700 rounded-full h-2">
              <div class="bg-green-500 h-2 rounded-full transition-all duration-300" style="width: ${planet.Morale || 0}%"></div>
            </div>
          </div>
          
          <!-- Resources Grid -->
          <div class="grid grid-cols-2 gap-3 mt-4 p-3 bg-space-800 rounded-lg">
            <div class="flex items-center gap-2">
              <span class="material-icons text-xl text-yellow-300">account_balance_wallet</span>
              <div>
                <div class="text-xs text-space-400">Credits</div>
                <div class="font-semibold text-yellow-300">${planet.Credits?.toLocaleString() || 0}</div>
              </div>
            </div>
            <div class="flex items-center gap-2">
              <span class="material-icons text-xl text-lime-300">restaurant</span>
              <div>
                <div class="text-xs text-space-400">Food</div>
                <div class="font-semibold text-lime-300">${planet.Food?.toLocaleString() || 0}</div>
              </div>
            </div>
            <div class="flex items-center gap-2">
              <span class="material-icons text-xl text-gray-300">construction</span>
              <div>
                <div class="text-xs text-space-400">Ore</div>
                <div class="font-semibold text-gray-300">${planet.Ore?.toLocaleString() || 0}</div>
              </div>
            </div>
            <div class="flex items-center gap-2">
              <span class="material-icons text-xl text-orange-300">inventory_2</span>
              <div>
                <div class="text-xs text-space-400">Goods</div>
                <div class="font-semibold text-orange-300">${planet.Goods?.toLocaleString() || 0}</div>
              </div>
            </div>
            <div class="flex items-center gap-2">
              <span class="text-xl">⛽</span>
              <div>
                <div class="text-xs text-space-400">Fuel</div>
                <div class="font-semibold text-purple-300">${planet.Fuel?.toLocaleString() || 0}</div>
              </div>
            </div>
          </div>
        </div>
      `;
    }

    let buildingsHtml = '<div class="text-sm text-space-400">No buildings constructed.</div>';
    if (planet.Buildings && Object.keys(planet.Buildings).length > 0) {
      const buildingEntries = Object.entries(planet.Buildings).map(([buildingName, level]) => {
        let displayName = buildingName;
        let buildingIcon = '🏢';
        if (this.gameState && this.gameState.buildingTypes) {
            const buildingType = this.gameState.buildingTypes.find(bt => bt.id === buildingName || bt.name.toLowerCase() === buildingName.toLowerCase());
            if (buildingType) {
              displayName = buildingType.name;
              // Add icons based on building type
              if (displayName.toLowerCase().includes('farm')) buildingIcon = '🌾';
              else if (displayName.toLowerCase().includes('mine')) buildingIcon = '⛏️';
              else if (displayName.toLowerCase().includes('factory')) buildingIcon = '🏭';
              else if (displayName.toLowerCase().includes('bank')) buildingIcon = '🏦';
              else if (displayName.toLowerCase().includes('research')) buildingIcon = '🔬';
            }
        }
        return `
          <li class="p-3 bg-space-700 rounded-md flex items-center justify-between hover:bg-space-600 transition-colors">
            <div class="flex items-center gap-2">
              <span class="text-xl">${buildingIcon}</span>
              <span class="font-semibold">${displayName}</span>
            </div>
            <span class="text-sm text-space-300">Level ${level}</span>
          </li>
        `;
      }).join("");
      buildingsHtml = `<ul class="space-y-2">${buildingEntries}</ul>`;
    }

    const isColonized = planet.colonized_by && planet.colonized_by !== "";
    const isOwnedByPlayer = isColonized && planet.colonized_by === this.currentUser?.id;
    const canColonize = !isColonized && this.currentUser && this.gameState && this.gameState.playerResources && this.gameState.playerResources.credits >= 500;

    // Redraw inner content structure
    container.innerHTML = `
      <div class="floating-panel-content">
        <div class="panel-header flex justify-between items-center p-3 cursor-move border-b border-space-700/50" draggable="false">
          <div class="flex items-center gap-2">
            <span class="material-icons text-space-400 drag-handle">drag_indicator</span>
            <span id="planet-icon" class="text-2xl"></span>
            <h2 id="planet-name" class="text-xl font-bold text-orange-300"></h2>
          </div>
          <button onclick="window.uiController.clearExpandedView()"
                  class="btn-icon hover:bg-space-700 rounded">
            <span class="material-icons text-sm">close</span>
          </button>
        </div>
        <div class="p-4">
          <div class="mb-4">
            <div id="planet-type-size" class="text-sm text-space-300 mb-3"></div>
            <div id="planet-resources-icons" class="flex gap-2 justify-center p-3 bg-gradient-to-r from-space-800/50 to-space-700/50 rounded-lg"></div>
          </div>

          <div id="planet-details-scroll-container" class="flex-1 overflow-y-auto pr-2 custom-scrollbar space-y-4">
            <div id="planet-resources-container" style="display: none;">
              <h3 class="text-lg font-semibold mb-3 text-nebula-200">Resources & Stats</h3>
              <div id="planet-resources-html"></div>
            </div>
            <div id="planet-buildings-container" style="display: none;">
              <h3 class="text-lg font-semibold mb-3 text-nebula-200">Buildings</h3>
              <div id="planet-buildings-html"></div>
            </div>
          </div>

          <div id="planet-actions-container" class="mt-4 space-y-2">
            <!-- Action buttons will be dynamically added here -->
          </div>
        </div>
      </div>
      `;

    // Update header info
    container.querySelector("#planet-icon").textContent = planetIcon;
    container.querySelector("#planet-name").textContent = planetName;
    container.querySelector("#planet-type-size").innerHTML = `
      <div class="text-center">
        <div class="font-medium">${planetTypeName}</div>
        <div class="text-xs text-space-400">Size ${planet.size || 'N/A'} • ${systemName}</div>
      </div>
    `;

    // Update resource icons
    const resourceIcons = this.getResourceIcons(planet);
    container.querySelector("#planet-resources-icons").innerHTML = resourceIcons.join('');

    // Update resources and buildings
    const resourcesContainer = container.querySelector("#planet-resources-container");
    const buildingsContainer = container.querySelector("#planet-buildings-container");

    if (isOwnedByPlayer || planet.Credits !== undefined) {
      container.querySelector("#planet-resources-html").innerHTML = resourcesHtml;
      resourcesContainer.style.display = "block";
    } else {
      resourcesContainer.style.display = "none";
    }

    if (isOwnedByPlayer) {
      container.querySelector("#planet-buildings-html").innerHTML = buildingsHtml;
      buildingsContainer.style.display = "block";
    } else {
      buildingsContainer.style.display = "none";
    }

    // Update action buttons
    const actionsContainer = container.querySelector("#planet-actions-container");
    actionsContainer.innerHTML = ''; // Clear previous buttons

    if (canColonize) {
      const availableSettlerFleets = this.getAvailableSettlerFleets(planet.system_id);
      if (availableSettlerFleets.length > 0) {
        const colonizeButton = document.createElement("button");
        colonizeButton.className = "w-full btn btn-success py-3 flex items-center justify-center gap-2";
        colonizeButton.innerHTML = `<span class="material-icons">rocket_launch</span> Colonize Planet (Settler Ship)`;
        colonizeButton.onclick = () => window.uiController.colonizePlanetWrapper(planet.id);
        actionsContainer.appendChild(colonizeButton);
      }
    } else if (!isColonized && this.currentUser && this.gameState) {
      const noSettlerButton = document.createElement("button");
      noSettlerButton.className = "w-full btn btn-disabled py-3 flex items-center justify-center gap-2";
      noSettlerButton.innerHTML = `<span class="material-icons">rocket_launch</span> Colonize Planet (Need Settler Ship)`;
      noSettlerButton.disabled = true;
      actionsContainer.appendChild(noSettlerButton);
    }

    if (isOwnedByPlayer) {
      const constructButton = document.createElement("button");
      constructButton.className = "w-full btn btn-primary py-3 flex items-center justify-center gap-2";
      constructButton.innerHTML = `<span>🏗️</span> Construct Building`;
      constructButton.onclick = () => window.uiController.showPlanetBuildModal(planet);
      actionsContainer.appendChild(constructButton);
    }

    const backButton = document.createElement("button");
    backButton.className = "w-full btn btn-secondary py-3 flex items-center justify-center gap-2";
    backButton.textContent = "← Back to System";
    backButton.onclick = () => window.uiController.goBackToSystemView(planet.system_id);
    actionsContainer.appendChild(backButton);

    container.classList.remove('hidden'); // Make sure it's not hidden before positioning
    if (screenX !== undefined && screenY !== undefined) {
        this.positionPanel(container, screenX, screenY);
    } else if (container.style.left === '-2000px' || container.style.left === '-9999px' || !container.style.left) {
        container.style.top = '20px';
        container.style.left = '20px';
        container.style.right = 'auto';
    }
    
    // Always re-add drag functionality after content recreation
    this.makePanelDraggable(container);
  }

  showPlanetBuildModal(planet) {
    if (!this.currentUser) {
      this.showError("Please log in to construct buildings.");
      return;
    }
    if (!planet || !planet.id) {
      this.showError("Invalid planet data provided for construction.");
      return;
    }

    const buildingTypes = this.gameState?.buildingTypes;

    if (!buildingTypes || buildingTypes.length === 0) {
      console.warn("Building types not available or empty in gameState for showPlanetBuildModal.");
      this.showModal(
        `Construct on ${planet.name || `Planet ${planet.id.slice(-4)}`}`,
        `<div class="text-space-400">No building types available or data is still loading.</div>
         <button class="w-full mt-2 btn btn-secondary" onclick="window.uiController.hideModal()">Close</button>`
      );
      return;
    }

    const buildingOptions = buildingTypes
      .map((buildingType) => {
        let costString = "Cost: ";
        if (buildingType.cost === undefined) { // Check if cost is defined at all
            costString += "N/A (data missing)";
        } else if (typeof buildingType.cost === "number") {
          costString += `${buildingType.cost} Credits`;
        } else if (typeof buildingType.cost === "object" && buildingType.cost !== null) {
          const resourceTypesMap = (this.gameState?.resourceTypes || []).reduce((map, rt) => {
            map[rt.id] = rt.name;
            return map;
          }, {});
          costString += Object.entries(buildingType.cost)
            .map(([resourceId, amount]) => {
              const resourceName = resourceTypesMap[resourceId] || resourceId;
              return `${amount} ${resourceName}`;
            })
            .join(", ");
            if (Object.keys(buildingType.cost).length === 0) costString += "Free"; // Handle empty cost object
        } else {
          costString += "N/A"; // Fallback for null or other unexpected types
        }

        // Safely stringify planet.id and buildingType.id for the onclick handler
        const safePlanetId = planet.id.replace(/'/g, "\\'");
        const safeBuildingTypeId = buildingType.id.replace(/'/g, "\\'");

        return `
      <button class="w-full p-3 bg-space-700 hover:bg-space-600 rounded mb-2 text-left"
              onclick="window.gameState.queueBuilding('${safePlanetId}', '${safeBuildingTypeId}'); window.uiController.hideModal();">
        <div class="font-semibold">${buildingType.name || "Unknown Building"}</div>
        <div class="text-sm text-space-300">${buildingType.description || "No description available."}</div>
        <div class="text-sm text-green-400">${costString}</div>
      </button>
    `;
      })
      .join("");

    this.showModal(
      `Construct on ${planet.name || `Planet ${planet.id.slice(-4)}`}`,
      `
      <div class="space-y-2 max-h-96 overflow-y-auto">
        ${buildingOptions.length > 0 ? buildingOptions : '<div class="text-space-400">No buildings available to construct.</div>'}
      </div>
      <button class="w-full mt-4 btn btn-secondary" onclick="window.uiController.hideModal()">Cancel</button>
    `
    );
  }

  // Helper method to get fleets with settler ships at a specific system
  getAvailableSettlerFleets(systemId) {
    if (!this.gameState || !this.gameState.fleets) {
      return [];
    }

    return this.gameState.fleets.filter(fleet => {
      // Fleet must be at the target system
      if (fleet.current_system !== systemId) {
        return false;
      }

      // Fleet must be owned by the current user
      if (fleet.owner_id !== this.currentUser?.id) {
        return false;
      }

      // Check if fleet has settler ships
      return fleet.ships && fleet.ships.some(ship => 
        ship.ship_type_name === 'settler' && ship.count > 0
      );
    });
  }

  // Wrapper for colonizePlanet to fit new UI structure if needed
  colonizePlanetWrapper(planetId) {
    // Find planet data again, or ensure it's correctly passed
    // For simplicity, assuming colonizePlanet can fetch necessary data or is adapted
    if (!this.gameState || !this.gameState.mapData || !this.gameState.mapData.planets) {
        this.showError("Game data not loaded. Cannot colonize.");
        return;
    }
    const planet = this.gameState.mapData.planets.find(p => p.id === planetId);
    if (!planet) {
        this.showError("Planet data not found. Cannot colonize.");
        return;
    }
    // Check if already showing colonize modal or similar logic
    // This replaces the old showPlanetColonizeModal call path
    this.colonizePlanet(planet.id, planet.system_id); // Pass both planet ID and system ID
  }

  async colonizePlanet(planetId, systemId = null) {
    try {
      const { pb } = await import("../lib/pocketbase.js");



      if (!pb.authStore.isValid) {
        this.showError("Please log in first to colonize planets");
        return;
      }

      // Get system ID - either passed directly or from planet data
      let targetSystemId = systemId;
      if (!targetSystemId) {
        // Fallback: find system ID from game state data
        const planet = this.gameState?.mapData?.planets?.find(p => p.id === planetId);
        if (!planet) {
          this.showError("Planet not found in game data");
          return;
        }
        targetSystemId = planet.system_id;
      }
      
      const availableSettlerFleets = this.getAvailableSettlerFleets(targetSystemId);
      
      if (availableSettlerFleets.length === 0) {
        this.showError("No settler ships available at this system");
        return;
      }

      // Use the first available fleet with settler ships
      const fleetToUse = availableSettlerFleets[0];

      const response = await fetch(`${pb.baseUrl}/api/orders/colonize`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: pb.authStore.token,
        },
        body: JSON.stringify({
          planet_id: planetId,
          fleet_id: fleetToUse.id,
        }),
      });

      const result = await response.json();

      if (response.ok && result.success) {
        this.hideModal();
        
        // Immediately refresh game state to show the new colony
        const { gameState } = await import("../stores/gameState.js");
        await gameState.refreshGameData();
        
        // Force update the current system view if we're looking at the colonized system
        if (this.gameState && this.gameState.selectedSystem && this.gameState.selectedSystem.id === targetSystemId) {
          // Re-display the system view with updated data
          setTimeout(() => {
            this.displaySystemView(this.gameState.selectedSystem);
          }, 100);
        }
        
        this.showSuccessMessage(
          "Planet colonized successfully! Your settler ship has established a new colony.",
        );
      } else {
        throw new Error(result.message || "Colonization failed");
      }
    } catch (error) {
      console.error("Colonization error:", error);
      this.showError(`Failed to colonize planet: ${error.message}`);
    }
  }

  goBackToSystemView(systemId) {
    if (!this.gameState || !this.gameState.mapData || !this.gameState.mapData.systems) {
      this.showError("Game data not fully loaded.");
      this.clearExpandedView();
      return;
    }
    const system = this.gameState.mapData.systems.find(s => s.id === systemId);
    if (system) {
      let planetsInSystem = [];
      if (this.gameState.mapData.planets) {
        planetsInSystem = this.gameState.mapData.planets.filter(p => {
            if (Array.isArray(p.system_id)) return p.system_id.includes(system.id);
            return p.system_id === system.id;
        });
      }
      this.displaySystemView(system, planetsInSystem);
    } else {
      this.showError("System data not found. Cannot go back.");
      this.clearExpandedView();
    }
  }

  manageColony(planetId) {
    // Placeholder for managing colony - could open build modal for this planet
    // For now, let's try to open the general build modal, but ideally it would be context-aware
    // This will require finding the system for this planet first
     if (!this.gameState || !this.gameState.mapData || !this.gameState.mapData.planets) {
        this.showError("Game data not loaded. Cannot manage colony.");
        return;
    }
    const planet = this.gameState.mapData.planets.find(p => p.id === planetId);
    if (!planet) {
        this.showError("Planet data not found.");
        return;
    }
    const system = this.gameState.mapData.systems.find(s => s.id === planet.system_id);
    if (system) {
        this.showBuildModal(system); // This is an existing modal, might need adaptation for planet-specific context
    } else {
        this.showError("System for this planet not found.");
    }
  }


  updateAuthUI(user) {
    this.currentUser = user;
    const loginBtn = document.getElementById("login-btn");
    const userInfo = document.getElementById("user-info");
    const username = document.getElementById("username");

    if (user) {
      loginBtn.classList.add("hidden");
      userInfo.classList.remove("hidden");
      username.textContent = user.username;
    } else {
      loginBtn.classList.remove("hidden");
      userInfo.classList.add("hidden");
      username.textContent = "";
    }
  }

  updateGameUI(state) {
    this.gameState = state;
    this.updateResourcesUI(state.playerResources);
    // this.updateSystemInfoUI(state.selectedSystem); // Removed: Replaced by expanded-view-container logic
    this.updateGameStatusUI(state);
  }

  updateResourcesUI(resources) {
    document.getElementById("credits").textContent =
      resources.credits?.toLocaleString();

    // Show credit income if available
    const incomeElement = document.getElementById("credit-income");
    if (this.gameState?.creditIncome > 0) {
      incomeElement.textContent = `(+${this.gameState.creditIncome}/tick)`;
      incomeElement.style.display = "inline";
    } else {
      incomeElement.style.display = "none";
    }

    // Set up credits button click handler for breakdown modal
    const creditsBtn = document.getElementById("credits-btn");
    if (creditsBtn) {
      creditsBtn.onclick = () => this.showCreditsBreakdown();
    }
  }

  // updateSystemInfoUI and loadSystemPlanets are removed as their functionality
  // is being replaced by displaySystemView and displayPlanetView,
  // which manage the #expanded-view-container.
  // selectPlanet, showPlanetColonizeModal, and showPlanetInfo are also removed
  // as their roles are absorbed into displayPlanetView or handled by new interaction flows.
  // The actual colonizePlanet action method is kept.


  updateGameStatusUI(state) {
    const tickElement = document.getElementById("game-tick-display");
    if (tickElement) {
      const prevTick = tickElement.textContent;
      const newTick = `Tick: ${state.currentTick}`; // Label added for consistency
      tickElement.textContent = newTick;

      // Add flash animation if tick changed
      if (prevTick !== newTick && prevTick !== "Tick: 0") { // Avoid flash on initial load
        tickElement.style.animation = "none";
        tickElement.offsetHeight; // Trigger reflow
        tickElement.style.animation = "flash 0.5s ease-out";
      }
    }



    // Update tick rate display (this part of the logic might be combined with startTickTimer or be static if only countdown changes)
    const nextTickRateElement = document.getElementById("next-tick-display");
    if (nextTickRateElement && !this.tickTimer) { // Only set this if timer isn't running
        const tickRate = state.ticksPerMinute || 6;
        const secondsPerTick = Math.round(60 / tickRate);
        nextTickRateElement.textContent = `Next Tick: (${secondsPerTick}s period)`;
    }
  }

  startTickTimer(nextTickTime) {
    if (this.tickTimer) {
      clearInterval(this.tickTimer);
    }
    const nextTickDisplayElement = document.getElementById("next-tick-display");

    const updateTimer = () => {
      const now = new Date();
      const remaining = nextTickTime - now;

      if (remaining <= 0) {
        if (nextTickDisplayElement) nextTickDisplayElement.textContent = "Next Tick: Processing...";
        clearInterval(this.tickTimer);
        this.tickTimer = null; // Clear timer instance
        return;
      }

      const minutes = Math.floor(remaining / 60000);
      const seconds = Math.floor((remaining % 60000) / 1000);
      if (nextTickDisplayElement) {
        nextTickDisplayElement.textContent = `Next Tick: ${minutes}:${seconds.toString().padStart(2, "0")}`;
      }
    };

    updateTimer(); // Call immediately to set initial value
    this.tickTimer = setInterval(updateTimer, 1000);
  }

  showModal(title, content) {
    const modalOverlay = document.getElementById("modal-overlay");
    const modalContent = document.getElementById("modal-content");

    modalContent.innerHTML = `
      <div class="flex justify-between items-center mb-4">
        <h2 class="text-xl font-bold">${title}</h2>
        <button id="modal-close" class="text-space-400 hover:text-space-200">&times;</button>
      </div>
      ${content}
    `;

    modalOverlay.classList.remove("hidden");

    // Set up close button
    document.getElementById("modal-close").addEventListener("click", () => {
      this.hideModal();
    });
  }

  hideModal() {
    document.getElementById("modal-overlay").classList.add("hidden");
  }

  showError(message) {
    this.showModal(
      "Error",
      `
      <div class="text-red-400 mb-4">${message}</div>
      <button class="w-full btn btn-secondary" onclick="document.getElementById('modal-overlay').classList.add('hidden')">
        OK
      </button>
    `,
    );
  }

  showBuildModal(system) {
    const buildingTypes = this.gameState?.buildingTypes;

    if (!buildingTypes || buildingTypes.length === 0) {
      console.warn("Building types not available or empty in gameState.");
      this.showModal(
        `Build in ${system.name || `System ${system.id.slice(-3)}`}`,
        `<div class="text-space-400">No buildings available to construct or building types are still loading.</div>`,
      );
      return;
    }

    const buildingOptions = buildingTypes
      .map((buildingType) => {
        let costString = "Cost: ";
        if (typeof buildingType.cost === "number") {
          costString += `${buildingType.cost} Credits`;
        } else if (typeof buildingType.cost === "object") {
          // Assuming cost is an object like { "credits": 100, "ore": 50 }
          // And resourceTypes is an array of objects like [{ id: "ore", name: "Ore" }, ...]
          const resourceTypesMap = (this.gameState?.resourceTypes || []).reduce((map, rt) => {
            map[rt.id] = rt.name;
            return map;
          }, {});
          costString += Object.entries(buildingType.cost)
            .map(([resourceId, amount]) => {
              const resourceName = resourceTypesMap[resourceId] || resourceId;
              return `${amount} ${resourceName}`;
            })
            .join(", ");
        } else {
          costString += "N/A";
        }

        return `
      <button class="w-full p-3 bg-space-700 hover:bg-space-600 rounded mb-2 text-left"
              onclick="window.gameState.queueBuilding('${system.id}', '${buildingType.id}')">
        <div class="font-semibold">${buildingType.name || "Unknown Building"}</div>
        <div class="text-sm text-space-300">${buildingType.description || "No description available."}</div>
        <div class="text-sm text-green-400">${costString}</div>
      </button>
    `;
      })
      .join("");

    this.showModal(
      `Build in ${system.name || `System ${system.id.slice(-3)}`}`,
      `
      <div class="space-y-2">
        ${buildingOptions.length > 0 ? buildingOptions : '<div class="text-space-400">No buildings available to construct.</div>'}
      </div>
    `,
    );
  }

  showSendFleetModal(system) {
    const ownedSystems = this.gameState?.getOwnedSystems() || [];

    if (ownedSystems.length === 0) {
      this.showError("You need to own at least one system to send fleets");
      return;
    }

    const systemOptions = ownedSystems
      .map(
        (s) =>
          `<option value="${s.id}">${s.name || `System ${s.id.slice(-3)}`}</option>`,
      )
      .join("");

    this.showModal(
      "Send Fleet",
      `
      <form id="fleet-form" class="space-y-4">
        <div>
          <label class="block text-sm font-medium mb-1">From System:</label>
          <select id="from-system" class="w-full p-2 bg-space-700 border border-space-600 rounded">
            ${systemOptions}
          </select>
        </div>
        <div>
          <label class="block text-sm font-medium mb-1">To System:</label>
          <input type="text" id="to-system" value="${system.name || `System ${system.id.slice(-3)}`}"
                 class="w-full p-2 bg-space-700 border border-space-600 rounded" readonly>
          <input type="hidden" id="to-system-id" value="${system.id}">
        </div>
        <div>
          <label class="block text-sm font-medium mb-1">Fleet Strength:</label>
          <input type="number" id="fleet-strength" min="1" max="100" value="10"
                 class="w-full p-2 bg-space-700 border border-space-600 rounded">
        </div>
        <div class="flex space-x-2">
          <button type="submit" class="flex-1 btn btn-danger">
            Send Fleet
          </button>
          <button type="button" onclick="document.getElementById('modal-overlay').classList.add('hidden')"
                  class="flex-1 btn btn-secondary">
            Cancel
          </button>
        </div>
      </form>
    `,
    );

    document
      .getElementById("fleet-form")
      .addEventListener("submit", async (e) => {
        e.preventDefault();
        try {
          const fromId = document.getElementById("from-system").value;
          const toId = document.getElementById("to-system-id").value;
          const strength = parseInt(
            document.getElementById("fleet-strength").value,
          );

          await this.gameState.sendFleet(fromId, toId, strength);
          this.hideModal();
        } catch (error) {
          this.showError(`Failed to send fleet: ${error.message}`);
        }
      });
  }

  showTradeRouteModal(system) {
    const ownedSystems = this.gameState?.getOwnedSystems() || [];

    if (ownedSystems.length === 0) {
      this.showError(
        "You need to own at least one system to create trade routes",
      );
      return;
    }

    const systemOptions = ownedSystems
      .map(
        (s) =>
          `<option value="${s.id}">${s.name || `System ${s.id.slice(-3)}`}</option>`,
      )
      .join("");

    const cargoTypes = ["food", "ore", "goods", "fuel"];
    const cargoOptions = cargoTypes
      .map(
        (type) =>
          `<option value="${type}">${type.charAt(0).toUpperCase() + type.slice(1)}</option>`,
      )
      .join("");

    this.showModal(
      "Create Trade Route",
      `
      <form id="trade-form" class="space-y-4">
        <div>
          <label class="block text-sm font-medium mb-1">From System:</label>
          <select id="trade-from-system" class="w-full p-2 bg-space-700 border border-space-600 rounded">
            ${systemOptions}
          </select>
        </div>
        <div>
          <label class="block text-sm font-medium mb-1">To System:</label>
          <input type="text" value="${system.name || `System ${system.id.slice(-3)}`}"
                 class="w-full p-2 bg-space-700 border border-space-600 rounded" readonly>
          <input type="hidden" id="trade-to-system-id" value="${system.id}">
        </div>
        <div>
          <label class="block text-sm font-medium mb-1">Cargo Type:</label>
          <select id="cargo-type" class="w-full p-2 bg-space-700 border border-space-600 rounded">
            ${cargoOptions}
          </select>
        </div>
        <div>
          <label class="block text-sm font-medium mb-1">Cargo Capacity:</label>
          <input type="number" id="cargo-capacity" min="1" max="1000" value="100"
                 class="w-full p-2 bg-space-700 border border-space-600 rounded">
        </div>
        <div class="flex space-x-2">
          <button type="submit" class="flex-1 btn btn-success">
            Create Route
          </button>
          <button type="button" onclick="document.getElementById('modal-overlay').classList.add('hidden')"
                  class="flex-1 btn btn-secondary">
            Cancel
          </button>
        </div>
      </form>
    `,
    );

    document
      .getElementById("trade-form")
      .addEventListener("submit", async (e) => {
        e.preventDefault();
        try {
          const fromId = document.getElementById("trade-from-system").value;
          const toId = document.getElementById("trade-to-system-id").value;
          const cargo = document.getElementById("cargo-type").value;
          const capacity = parseInt(
            document.getElementById("cargo-capacity").value,
          );

          await this.gameState.createTradeRoute(fromId, toId, cargo, capacity);
          this.hideModal();
        } catch (error) {
          this.showError(`Failed to create trade route: ${error.message}`);
        }
      });
  }

  showFleetPanel() {
    const fleets = this.gameState?.getPlayerFleets() || [];

    const fleetList =
      fleets.length > 0
        ? fleets
            .map(
              (fleet) => `
      <div class="bg-space-700 p-3 rounded mb-2">
        <div class="font-semibold">Fleet ${fleet.id.slice(-3)}</div>
        <div class="text-sm text-space-300">
          <div>From: ${fleet.from_name || fleet.from_id}</div>
          <div>To: ${fleet.to_name || fleet.to_id}</div>
          <div>Strength: ${fleet.strength}</div>
          <div>ETA: ${fleet.eta_tick ? `Tick ${fleet.eta_tick}` : "Unknown"}</div>
        </div>
      </div>
    `,
            )
            .join("")
        : '<div class="text-space-400">No fleets in transit</div>';

    this.showModal("Your Fleets", fleetList);
  }

  showTradePanel() {
    const trades = this.gameState?.getPlayerTrades() || [];

    const tradeList =
      trades.length > 0
        ? trades
            .map(
              (trade) => `
      <div class="bg-space-700 p-3 rounded mb-2">
        <div class="font-semibold">Trade Route ${trade.id.slice(-3)}</div>
        <div class="text-sm text-space-300">
          <div>From: ${trade.from_name || trade.from_id}</div>
          <div>To: ${trade.to_name || trade.to_id}</div>
          <div>Cargo: ${trade.cargo}</div>
          <div>Capacity: ${trade.cap}</div>
          <div>ETA: ${trade.eta_tick ? `Tick ${trade.eta_tick}` : "Unknown"}</div>
        </div>
      </div>
    `,
            )
            .join("")
        : '<div class="text-space-400">No active trade routes</div>';

    this.showModal("Your Trade Routes", tradeList);
  }

  showDiplomacyPanel() {
    this.showModal(
      "Diplomacy",
      `
      <div class="text-center text-space-400 py-8">
        Diplomacy features coming soon!
      </div>
    `,
    );
  }

  showBuildingsPanel() {
    const buildings = this.gameState?.getPlayerBuildings() || [];
    // Updated to filter by credits_per_tick > 0 for income calculation
    const incomeGeneratingBuildings = buildings.filter((b) => b.credits_per_tick > 0);
    const totalIncome = incomeGeneratingBuildings.reduce(
      (sum, building) => sum + building.credits_per_tick, // Removed '|| 1'
      0,
    );

    // Group buildings by type
    const buildingsByType = buildings.reduce((acc, building) => {
      if (!acc[building.type]) acc[building.type] = [];
      acc[building.type].push(building);
      return acc;
    }, {});

    const buildingTypeNames = {};
    if (this.gameState && this.gameState.buildingTypes) {
      for (const bt of this.gameState.buildingTypes) {
        buildingTypeNames[bt.id] = bt.name || bt.id; // Fallback to ID if name is missing
      }
    } else {
      console.warn("Building types not available in gameState for building panel.");
    }

    const buildingSections = Object.entries(buildingsByType)
      .map(
        ([typeId, typeBuildings]) => `
      <div class="mb-4">
        <h3 class="text-lg font-semibold text-plasma-300 mb-2">${buildingTypeNames[typeId] || typeId} (${typeBuildings.length})</h3>
        <div class="space-y-2">
          ${typeBuildings
            .map(
              (building) => `
            <div class="bg-space-700 p-3 rounded">
              <div class="font-semibold text-nebula-300">${building.name || `${buildingTypeNames[building.type] || building.type} ${building.id.slice(-3)}`}</div>
              <div class="text-sm text-space-300">
                <div>System: ${building.system_name || building.system_id}</div>
                ${building.credits_per_tick > 0 ? `<div class="text-nebula-300">Income: ${building.credits_per_tick} credits/tick</div>` : ""}
                <div class="text-xs ${building.active !== false ? "text-green-400" : "text-red-400"}">
                  ${building.active !== false ? "Active" : "Inactive"}
                </div>
              </div>
            </div>
          `,
            )
            .join("")}
        </div>
      </div>
    `,
      )
      .join("");

    this.showModal(
      "Buildings Overview",
      `
      ${
        totalIncome > 0 // Changed condition to totalIncome > 0
          ? `
        <div class="mb-4 p-3 bg-space-800 rounded">
          <div class="text-lg font-semibold text-plasma-300">Credit Income: ${totalIncome} credits/tick</div>
          <div class="text-sm text-space-400">${totalIncome * 6} credits/minute • ${totalIncome * 360} credits/hour</div>
        </div>
      `
          : ""
      }

      ${buildingSections || '<div class="text-space-400 text-center py-8">No buildings constructed</div>'}

      <div class="mt-4 text-xs text-space-400 border-t border-space-600 pt-2">
        💡 Build structures at your systems to improve production and defense
      </div>
    `,
    );
  }

  showColonizeModal(system) {
    if (!this.currentUser) {
      this.showError("Please log in to colonize planets");
      return;
    }

    // We need to fetch planets in this system
    fetch(`http://localhost:8090/api/planets?system_id=${system.id}`)
      .then((response) => response.json())
      .then((data) => {
        const planets = data.items || [];
        if (planets.length === 0) {
          this.showError("No planets found in this system");
          return;
        }

        const planetOptions = planets
          .map((planet) => {
            const isColonized =
              planet.colonized_by != null && planet.colonized_by !== "";
            const planetTypeName = this.getPlanetTypeName(planet.type) || "Unknown";

            return `
            <div class="p-3 bg-space-700 rounded mb-2 ${isColonized ? "opacity-50" : "hover:bg-space-600 cursor-pointer"}"
                 ${!isColonized ? `onclick="window.uiController.colonizePlanet('${planet.id}')"` : ""}>
              <div class="font-semibold">${planet.name}</div>
              <div class="text-sm text-space-300">Type: ${planetTypeName}</div>
              <div class="text-sm text-space-300">Size: ${planet.size}</div>
              ${
                isColonized
                  ? `<div class="text-sm text-red-400">Already colonized</div>`
                  : `<div class="text-sm text-emerald-400">Available for colonization</div>`
              }
            </div>
          `;
          })
          .join("");

        this.showModal(
          `Colonize Planet in ${system.name || `System ${system.id.slice(-3)}`}`,
          `
          <div class="space-y-2">
            <div class="text-sm text-space-300 mb-4">
              Select a planet to establish a new colony:
            </div>
            ${planetOptions}
          </div>
        `,
        );

        // Store reference for colonization
        window.uiController = this;
      })
      .catch((err) => {
        console.error("Error fetching planets:", err);
        this.showError("Failed to load planets in this system");
      });
  }

  showSuccessMessage(message) {
    this.showModal(
      "Success",
      `
      <div class="text-emerald-400 mb-4">${message}</div>
      <button class="w-full btn btn-secondary" onclick="document.getElementById('modal-overlay').classList.add('hidden')">
        OK
      </button>
    `,
    );
    
    // Auto-close success message after 3 seconds
    setTimeout(() => {
      const modal = document.getElementById('modal-overlay');
      if (modal && !modal.classList.contains('hidden')) {
        modal.classList.add('hidden');
      }
    }, 3000);
  }

  showCreditsBreakdown() {
      if (!this.currentUser) {
        this.showError("Please log in to view credit breakdown");
        return;
      }

      // Get all crypto_server buildings for the user
      const buildings = this.gameState?.getPlayerBuildings() || [];
      const cryptoServers = buildings.filter(building => {
        const buildingTypeName = this.gameState?.buildingTypes?.find(bt => bt.id === building.type)?.name;
        return buildingTypeName === 'crypto_server';
      });

      let totalCredits = this.gameState?.playerResources?.credits || 0;
      let totalProduction = 0;

      // Calculate total production per tick
      cryptoServers.forEach(building => {
        if (building.credits_per_tick) {
          totalProduction += building.credits_per_tick;
        }
      });

      const buildingsList = cryptoServers.length > 0 ? 
        cryptoServers.map(building => {
          const systemName = building.system_name || `System ${building.system_id?.slice(-3)}`;
          const storedCredits = building.stored_credits || 'Unknown';
          const production = building.credits_per_tick || 1;
        
          return `
            <div class="bg-space-700 p-3 rounded mb-2">
              <div class="flex justify-between items-center">
                <div>
                  <div class="font-semibold text-nebula-300">Crypto Server</div>
                  <div class="text-sm text-space-300">Location: ${systemName}</div>
                </div>
                <div class="text-right">
                  <div class="text-nebula-300">+${production}/tick</div>
                  <div class="text-xs text-space-400">Level ${building.level || 1}</div>
                </div>
              </div>
            </div>
          `;
        }).join('') :
        '<div class="text-space-400 text-center py-4">No crypto servers found</div>';

      this.showModal(
        '<span class="flex items-center gap-2"><span class="material-icons">account_balance_wallet</span>Credits Breakdown</span>',
        `
          <div class="space-y-4">
            <div class="bg-space-800 p-4 rounded-lg">
              <div class="grid grid-cols-2 gap-4 text-center">
                <div>
                  <div class="text-2xl font-bold text-nebula-300">${totalCredits.toLocaleString()}</div>
                  <div class="text-sm text-space-400">Total Credits</div>
                </div>
                <div>
                  <div class="text-2xl font-bold text-plasma-300">+${totalProduction}</div>
                  <div class="text-sm text-space-400">Per Tick</div>
                </div>
              </div>
            </div>
          
            <div>
              <h3 class="text-lg font-semibold mb-3 text-nebula-200">Credit Sources</h3>
              <div class="max-h-60 overflow-y-auto custom-scrollbar">
                ${buildingsList}
              </div>
            </div>

            ${cryptoServers.length === 0 ? `
              <div class="bg-amber-900/20 border border-amber-600/30 p-3 rounded">
                <div class="text-amber-300 text-sm">
                  💡 <strong>Tip:</strong> Build Crypto Servers on your planets to generate credits over time!
                </div>
              </div>
            ` : ''}
          </div>
        
          <button class="w-full btn btn-secondary mt-4" onclick="document.getElementById('modal-overlay').classList.add('hidden')">
            Close
          </button>
        `
      );
    }
  }
