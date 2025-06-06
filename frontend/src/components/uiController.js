import { FleetComponentManager } from './fleet/FleetComponentManager.js';

// UI Controller for handling all UI interactions and updates
export class UIController {
  constructor() {
    this.currentUser = null;
    this.gameState = null;
    this.tickTimer = null;
    this.currentSystemId = null; // Track current system/planet ID being viewed
    this.planetTypes = new Map(); // Store planet types with their icons
    this.pb = null; // Will be set when PocketBase is available
    this.resourceTypes = new Map(); // Store resource types with their icons
    this.fleetComponentManager = null; // Will be initialized when game state is available
    this.displayedResources = this.loadResourcePreferences(); // Which resources to show in top bar

    // Make instance available globally, for event handlers in dynamically created HTML
    window.uiController = this;
    this.expandedView = document.getElementById("expanded-view-container");
    if (this.expandedView) {
      // Ensure it starts hidden and styled as a panel (though class will be set on show)
      this.expandedView.classList.add("hidden", "floating-panel");
      this.expandedView.style.left = "-2000px"; // Start off-screen
      this.expandedView.style.top = "-2000px";
    } else {
      console.error(
        "#expanded-view-container not found during UIController construction",
      );
    }

    // Initialize resources dropdown
    this.initializeResourcesDropdown();
  }

  setPocketBase(pb) {
    this.pb = pb;
    this.loadPlanetTypes(); // Load planet types when PocketBase is available
    this.loadResourceTypes(); // Load resource types when PocketBase is available
  }

  async loadPlanetTypes() {
    try {
      if (!this.pb) return; // PocketBase not initialized yet
      const response = await this.pb.collection("planet_types").getFullList();
      this.planetTypes.clear();

      response.forEach((type) => {
        // Store by both name and ID for lookup flexibility
        const typeData = {
          name: type.name,
          icon: type.icon || "", // URL to icon image
        };
        this.planetTypes.set(type.name.toLowerCase(), typeData);
        this.planetTypes.set(type.id, typeData);
      });
    } catch (error) {
      console.warn("Failed to load planet types:", error);
    }
  }

  async loadResourceTypes() {
    try {
      if (!this.pb) return; // PocketBase not initialized yet
      const response = await this.pb.send("/api/resource_types", {
        method: "GET",
      });
      this.resourceTypes.clear();

      response.items.forEach((type) => {
        // Store by both name and ID for lookup flexibility
        const typeData = {
          name: type.name,
          icon: type.icon || "", // URL to icon image
        };
        this.resourceTypes.set(type.name.toLowerCase(), typeData);
        this.resourceTypes.set(type.id, typeData);
      });

      // Update resources dropdown after loading types
      this.updateResourcesDropdown();
    } catch (error) {
      console.warn("Failed to load resource types:", error);
    }
  }

  getPlanetTypeIcon(planetTypeName) {
    if (!planetTypeName)
      return '<img src="/placeholder-planet.svg" class="w-6 h-6" alt="Unknown planet type" />';

    // Try lookup by ID first, then by name (lowercase)
    let planetType = this.planetTypes.get(planetTypeName);
    if (!planetType) {
      planetType = this.planetTypes.get(planetTypeName.toLowerCase());
    }

    if (planetType && planetType.icon) {
      // Color border for different planet types
      const colorMap = {
        highlands: "border-green-400",
        abundant: "border-emerald-400",
        fertile: "border-lime-400",
        mountain: "border-stone-400",
        desert: "border-yellow-400",
        volcanic: "border-red-400",
        swamp: "border-blue-400",
        barren: "border-gray-400",
        radiant: "border-purple-400",
        barred: "border-red-600",
      };

      const colorClass =
        colorMap[planetType.name.toLowerCase()] || "border-space-300";
      return `<img src="${planetType.icon}" class="w-6 h-6 rounded border-2 ${colorClass}" alt="${planetType.name}" title="${planetType.name}" />`;
    }

    // Fallback if no icon URL is set
    return '<img src="/placeholder-planet.svg" class="w-6 h-6" alt="Unknown planet type" />';
  }

  getPlanetTypeName(planetTypeId) {
    if (!planetTypeId) return "Unknown";

    // Try lookup by ID first, then by name (lowercase)
    let planetType = this.planetTypes.get(planetTypeId);
    if (!planetType) {
      planetType = this.planetTypes.get(planetTypeId.toLowerCase());
    }

    return planetType ? planetType.name : "Unknown";
  }

  getPlanetAnimatedGif(planetTypeName) {
    if (!planetTypeName) return null;

    // Default GIF to use when specific ones aren't available
    const defaultGifPath = "/planets/default.gif";

    // Map planet types to their preferred animated GIF files
    const gifMap = {
      highlands: "/planets/highlands.gif",
      abundant: "/planets/abundant.gif",
      fertile: "/planets/fertile.gif",
      mountain: "/planets/mountain.gif",
      desert: "/planets/desert.gif",
      volcanic: "/planets/volcanic.gif",
      swamp: "/planets/swamp.gif",
      barren: "/planets/barren.gif",
      radiant: "/planets/radiant.gif",
      barred: "/planets/barred.gif",
      null: "/planets/null.gif",
    };

    // Try lookup by ID first, then by name (lowercase)
    let planetType = this.planetTypes.get(planetTypeName);
    if (!planetType) {
      planetType = this.planetTypes.get(planetTypeName.toLowerCase());
    }

    const typeName = planetType
      ? planetType.name.toLowerCase()
      : planetTypeName.toLowerCase();

    // Use specific GIF if available, otherwise fall back to default
    const gifPath = gifMap[typeName] || defaultGifPath;

    return `<img src="${gifPath}" class="w-12 h-12 rounded-full border-2 border-space-400 shadow-lg" alt="${typeName} planet" title="${typeName} planet" />`;
  }

  getPlanetTypeGradient(planetTypeId) {
    if (!planetTypeId) return "from-nebula-900/30 to-plasma-900/30";

    // Try lookup by ID first, then by name (lowercase)
    let planetType = this.planetTypes.get(planetTypeId);
    if (!planetType) {
      planetType = this.planetTypes.get(planetTypeId.toLowerCase());
    }

    if (!planetType) return "from-nebula-900/30 to-plasma-900/30";

    // Planet type specific gradients
    const gradientMap = {
      highlands: "from-green-900/30 to-emerald-800/30",
      abundant: "from-green-800/30 to-lime-700/30",
      fertile: "from-green-700/30 to-green-600/30",
      mountain: "from-gray-800/30 to-slate-700/30",
      desert: "from-yellow-800/30 to-orange-700/30",
      volcanic: "from-red-900/30 to-orange-800/30",
      swamp: "from-cyan-900/30 to-teal-800/30",
      barren: "from-gray-900/30 to-gray-800/30",
      radiant: "from-yellow-600/30 to-amber-500/30",
      barred: "from-red-800/30 to-red-900/30",
    };

    return (
      gradientMap[planetType.name.toLowerCase()] ||
      "from-nebula-900/30 to-plasma-900/30"
    );
  }

  getSystemGradient(planets) {
    if (!planets || planets.length === 0)
      return "from-nebula-900/30 to-plasma-900/30";

    // Count planet types
    const typeCounts = {};
    planets.forEach((planet) => {
      const planetTypeValue = planet.planet_type || planet.type;
      if (planetTypeValue) {
        let planetType = this.planetTypes.get(planetTypeValue);
        if (!planetType) {
          planetType = this.planetTypes.get(planetTypeValue.toLowerCase());
        }
        if (planetType) {
          const typeName = planetType.name.toLowerCase();
          typeCounts[typeName] = (typeCounts[typeName] || 0) + 1;
        }
      }
    });

    // Find the most common planet type
    let dominantType = "unknown";
    let maxCount = 0;
    for (const [type, count] of Object.entries(typeCounts)) {
      if (count > maxCount) {
        maxCount = count;
        dominantType = type;
      }
    }

    // Use the dominant planet type's gradient for the system
    const gradientMap = {
      highlands: "from-green-900/30 to-emerald-800/30",
      abundant: "from-green-800/30 to-lime-700/30",
      fertile: "from-green-700/30 to-green-600/30",
      mountain: "from-gray-800/30 to-slate-700/30",
      desert: "from-yellow-800/30 to-orange-700/30",
      volcanic: "from-red-900/30 to-orange-800/30",
      swamp: "from-cyan-900/30 to-teal-800/30",
      barren: "from-gray-900/30 to-gray-800/30",
      radiant: "from-yellow-600/30 to-amber-500/30",
      barred: "from-red-800/30 to-red-900/30",
    };

    return gradientMap[dominantType] || "from-nebula-900/30 to-plasma-900/30";
  }

  async getResourceNodes(planetId) {
    if (!this.pb) return [];

    try {
      const resourceNodes = await this.pb
        .collection("resource_nodes")
        .getFullList({
          filter: `planet_id = "${planetId}"`,
          expand: "resource_type",
        });
      return resourceNodes;
    } catch (error) {
      console.warn("Failed to load resource nodes:", error);
      return [];
    }
  }

  async loadResourceTypes() {
    if (!this.pb) return [];

    try {
      const resourceTypes = await this.pb
        .collection("resource_types")
        .getFullList();
      return resourceTypes;
    } catch (error) {
      console.warn("Failed to load resource types:", error);
      return [];
    }
  }

  async getResourceIcons(planet) {
    const icons = [];

    // Show resource availability based on planet's resource nodes
    if (planet.resourceNodes && planet.resourceNodes.length > 0) {
      // Get resource types for lookup
      const resourceTypes = await this.loadResourceTypes();
      const resourceTypeMap = {};
      resourceTypes.forEach((rt) => {
        resourceTypeMap[rt.id] = rt;
      });

      const uniqueResourceTypes = new Set();
      planet.resourceNodes.forEach((node) => {
        let resourceType, resourceTypeData;

        // Check if expand worked
        if (node.expand && node.expand.resource_type) {
          resourceTypeData = node.expand.resource_type;
        } else {
          // Fallback: use the resource type ID to lookup
          resourceTypeData = resourceTypeMap[node.resource_type];
        }

        if (resourceTypeData) {
          const resourceKey = resourceTypeData.name.toLowerCase();

          if (!uniqueResourceTypes.has(resourceKey)) {
            uniqueResourceTypes.add(resourceKey);

            // Use the actual icon from the resource type
            const iconUrl = resourceTypeData.icon || "/placeholder-planet.svg";
            const resourceName = resourceTypeData.name;

            icons.push(
              `<img src="${iconUrl}" class="w-5 h-5" title="${resourceName}" alt="${resourceName}" />`,
            );
          }
        }
      });
    }

    return icons;
  }

  clearExpandedView() {
    if (this.expandedView) {
      // Don't close if panel is pinned
      if (this.expandedView._isPinned) {
        return;
      }

      // Clean up drag event listeners if they exist
      if (this.expandedView._dragCleanup) {
        this.expandedView._dragCleanup();
        delete this.expandedView._dragCleanup;
      }

      // Remove all panel classes
      this.expandedView.classList.remove("focused", "pinned-panel", "glass-panel");
      
      // Don't clear innerHTML here, as content might be reused or fade out.
      // Content replacement will happen in displaySystemView/displayPlanetView.
      this.expandedView.classList.add("hidden");
      // Move it off-screen to prevent interaction and ensure it's visually gone
      this.expandedView.style.left = "-9999px";
      this.expandedView.style.top = "-9999px";
    }
  }

  positionPanel(container, screenX, screenY) {
    // Ensure container is not hidden to get accurate dimensions
    const wasHidden = container.classList.contains("hidden");
    if (wasHidden) {
      container.classList.remove("hidden");
      // Temporarily make it visible but off-screen to measure
      container.style.left = "-9999px";
      container.style.top = "-9999px";
    }

    const panelWidth = container.offsetWidth;
    const panelHeight = container.offsetHeight;
    const viewportWidth = window.innerWidth;
    const viewportHeight = window.innerHeight;
    const margin = 20; // Base margin from viewport edges
    const systemOffset = 120; // Distance from system center to avoid covering navigation options

    // Start with a larger offset from the system to avoid covering nearby systems
    let top = screenY - systemOffset;
    let left = screenX + systemOffset;

    // Try different positions in order of preference to avoid covering the system
    const positions = [
      { left: screenX + systemOffset, top: screenY - systemOffset }, // Top-right
      {
        left: screenX - panelWidth - systemOffset,
        top: screenY - systemOffset,
      }, // Top-left
      { left: screenX + systemOffset, top: screenY + systemOffset }, // Bottom-right
      {
        left: screenX - panelWidth - systemOffset,
        top: screenY + systemOffset,
      }, // Bottom-left
      { left: screenX + systemOffset, top: screenY - panelHeight / 2 }, // Center-right
      {
        left: screenX - panelWidth - systemOffset,
        top: screenY - panelHeight / 2,
      }, // Center-left
    ];

    // Find the first position that fits within viewport
    let bestPosition = positions[0];
    for (const pos of positions) {
      if (
        pos.left >= margin &&
        pos.left + panelWidth + margin <= viewportWidth &&
        pos.top >= margin &&
        pos.top + panelHeight + margin <= viewportHeight
      ) {
        bestPosition = pos;
        break;
      }
    }

    left = bestPosition.left;
    top = bestPosition.top;

    // Final boundary adjustments if no perfect position was found
    if (left < margin) left = margin;
    if (left + panelWidth + margin > viewportWidth)
      left = viewportWidth - panelWidth - margin;
    if (top < margin) top = margin;
    if (top + panelHeight + margin > viewportHeight)
      top = viewportHeight - panelHeight - margin;

    container.style.left = `${left}px`;
    container.style.top = `${top}px`;

    if (wasHidden && container.style.left === "-9999px") {
      // If it was temp unhidden and not positioned, re-hide. Should not happen if positionPanel is called correctly.
      // This case is unlikely if called after content is set and ready to be shown.
      container.classList.add("hidden");
    }
  }

  async displaySystemView(system, planets, screenX, screenY) {
    const container = this.expandedView;
    if (!container) {
      console.error("#expanded-view-container not found in displaySystemView");
      return;
    }

    // If it's the same system and the panel is already visible, and no new coords are given,
    // we might not need to do anything, or just update content if planets data changed.
    // For now, this simplified check allows re-rendering if called.
    if (
      this.currentSystemId === system.id &&
      !container.classList.contains("hidden") &&
      screenX === undefined
    ) {
      // If planets data could change, we might need to proceed and update planet list.
      // For now, assume if it's the same system and visible, content is up to date unless new position is given.
      // This could be more sophisticated by checking if 'planets' data has actually changed.
      // return; // Potentially skip if no new position and system is same.
    }

    // Preserve existing classes (like pinned-panel) while ensuring base classes
    container.classList.remove("hidden");
    container.classList.add("floating-panel", "glass-panel");

    this.currentSystemId = system.id;

    // Calculate system stats
    let totalPopulation = 0;
    let colonizedCount = 0;
    let ownedByPlayer = 0;
    const currentUserId = this.currentUser?.id;

    if (planets && planets.length > 0) {
      planets.forEach((planet) => {
        if (planet.colonized_by) {
          colonizedCount++;
          if (planet.colonized_by === currentUserId) {
            ownedByPlayer++;
          }
        }
        totalPopulation += planet.Pop || 0;
      });
    }

    let planetsHtml =
      '<div class="text-sm text-space-400">No planets detected in this system.</div>';
    if (planets && planets.length > 0) {
      planetsHtml = planets
        .map((planet) => {
          const planetName = planet.name || `Planet ${planet.id.slice(-4)}`;
          const planetTypeValue = planet.planet_type || planet.type;
          const planetIcon = this.getPlanetTypeIcon(planetTypeValue);
          const planetTypeName = this.getPlanetTypeName(planetTypeValue);

          const isOwned = planet.colonized_by === currentUserId;
          const population = planet.Pop || 0;
          const maxPop = planet.MaxPopulation || "N/A";

          // Status indicators
          let statusHtml = "";
          if (planet.colonized_by) {
            if (isOwned) {
              statusHtml = `<span class="text-xs text-green-400 flex items-center gap-1"><span class="material-icons text-xs">check_circle</span>Your Colony</span>`;
            } else {
              statusHtml = `<span class="text-xs text-red-400 flex items-center gap-1"><span class="material-icons text-xs">block</span>${planet.colonized_by_name || "Occupied"}</span>`;
            }
          } else {
            statusHtml = `<span class="text-xs text-gray-400 flex items-center gap-1"><span class="material-icons text-xs">radio_button_unchecked</span>Uncolonized</span>`;
          }

          // Resource icons preview
          const resourceIcons = this.getResourceIcons(planet);
          let resourcesPreview = "";
          if (resourceIcons.length > 0) {
            resourcesPreview = `
            <div class="mt-2 flex gap-1 items-center">
              ${resourceIcons.join("")}
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
                    <div class="text-xs text-space-300">${planetTypeName} • Size ${planet.size || "N/A"}</div>
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
        })
        .join("");
    }

    // Determine if a full redraw of innerHTML is needed or if specific parts can be updated.
    // For simplicity in this refactor, we'll redraw the inner content structure.
    // More advanced would be to diff content if system.id is the same.
    container.innerHTML = `
      <div class="floating-panel-content">
        <div id="system-header" class="panel-header flex justify-between items-center p-3 cursor-move border-b border-space-700/50 bg-gradient-to-r from-nebula-900/30 to-plasma-900/30" draggable="false">
          <div class="flex items-center gap-2">
            <span class="material-icons text-space-400 drag-handle">drag_indicator</span>
            <span id="system-name" class="text-xl font-bold text-nebula-200"></span>
          </div>
          <div class="flex items-center gap-4">
            <div class="text-right">
              <div id="system-seed" class="font-semibold text-nebula-200 text-sm"></div>
              <div id="system-coords" class="font-mono text-xs text-gray-500"></div>
            </div>
            <div class="flex items-center gap-2">
              <button onclick="window.uiController.togglePinPanel(this.closest('.floating-panel'))" 
                      class="pin-button text-space-400 hover:text-white transition-colors" 
                      title="Pin">
                <span class="material-icons text-sm">push_pin</span>
              </button>
              <button onclick="window.uiController.clearExpandedView()"
                      class="btn-icon hover:bg-space-700 rounded">
                <span class="material-icons text-sm">close</span>
              </button>
            </div>
          </div>
        </div>
        <div class="p-4">

          <div class="flex-1 overflow-hidden flex flex-col">
            <div class="flex justify-end mb-2">
              <div class="text-xs text-space-400">
                <kbd class="px-1 py-0.5 bg-space-700 rounded text-xs">Click</kbd> Fleet to Select • <kbd class="px-1 py-0.5 bg-space-700 rounded text-xs">Shift+Click</kbd> System to Move • <kbd class="px-1 py-0.5 bg-space-700 rounded text-xs">↑↓←→</kbd> Navigate
              </div>
            </div>
            <ul id="system-planets-list" class="flex-1 overflow-y-auto pr-2 custom-scrollbar">
            </ul>
          </div>
        </div>
      </div>
      `;

    // Update dynamic content - set gradient and content for top bar
    const systemGradient = this.getSystemGradient(planets);
    const systemHeader = container.querySelector("#system-header");
    systemHeader.className = `panel-header flex justify-between items-center p-3 cursor-move border-b border-space-700/50 bg-gradient-to-r ${systemGradient}`;

    container.querySelector("#system-name").textContent =
      system.name || `System ${system.id.slice(-4)}`;
    container.querySelector("#system-seed").textContent =
      `Seed: ${system.id.slice(-8)}`;
    container.querySelector("#system-coords").textContent =
      `${system.x}, ${system.y}`;

    const planetsListUl = container.querySelector("#system-planets-list"); // Query within the new innerHTML
    await this.updatePlanetList(planetsListUl, planets, currentUserId);

    container.dataset.viewType = "system"; // For potential future logic
    container.dataset.currentId = system.id;

    // Position the panel (only if not pinned)
    container.classList.remove("hidden");
    
    // Don't reposition if panel is pinned - keep it where user placed it
    if (!container._isPinned) {
      if (screenX !== undefined && screenY !== undefined) {
        this.positionPanel(container, screenX, screenY);
      } else if (
        container.style.left === "-2000px" ||
        container.style.left === "-9999px" ||
        !container.style.left
      ) {
        // If it was off-screen or no position set, place it default
        container.style.top = "20px";
        container.style.left = "20px";
        container.style.right = "auto";
      }
    }

    // Always re-add drag functionality after content recreation
    this.makePanelDraggable(container);
    
    // Add focus effects
    this.addPanelFocusEffects(container);
  }

  makePanelDraggable(container) {
    const header = container.querySelector(".panel-header");
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
      if (e.target.closest(".panel-header") && !e.target.closest("button")) {
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
        container.style.transition = "none";
        container.style.right = "auto"; // Clear right positioning
        header.style.cursor = "grabbing";
        e.preventDefault();
      }
    };

    const dragEnd = () => {
      if (isDragging) {
        isDragging = false;
        container.style.transition = "";
        header.style.cursor = "move";
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
    header.addEventListener("mousedown", dragStart);
    document.addEventListener("mousemove", drag);
    document.addEventListener("mouseup", dragEnd);

    // Touch events for mobile
    header.addEventListener("touchstart", dragStart);
    document.addEventListener("touchmove", drag);
    document.addEventListener("touchend", dragEnd);

    // Clean up on panel removal
    const cleanup = () => {
      header.removeEventListener("mousedown", dragStart);
      document.removeEventListener("mousemove", drag);
      document.removeEventListener("mouseup", dragEnd);
      header.removeEventListener("touchstart", dragStart);
      document.removeEventListener("touchmove", drag);
      document.removeEventListener("touchend", dragEnd);
    };

    // Store cleanup function for later use
    container._dragCleanup = cleanup;
  }

  async updatePlanetList(ulElement, planets, currentUserId) {
    if (!planets || planets.length === 0) {
      ulElement.innerHTML =
        '<div class="text-sm text-space-400">No planets detected in this system.</div>';
      return;
    }

    // Clear existing content
    ulElement.innerHTML = "";

    // Create embedded planet containers
    for (const planet of planets) {
      const planetContainer = await this.createEmbeddedPlanetContainer(
        planet,
        currentUserId,
      );
      ulElement.appendChild(planetContainer);
    }
  }

  async createEmbeddedPlanetContainer(planet, currentUserId) {
    const planetId = planet.id;
    const planetName = planet.name || `Planet ${planet.id.slice(-4)}`;
    const planetTypeValue = planet.planet_type || planet.type;
    const planetTypeName = this.getPlanetTypeName(planetTypeValue);
    const planetGif = this.getPlanetAnimatedGif(planetTypeName);
    const isOwned = planet.colonized_by === currentUserId;
    const population = planet.Pop || 0;
    const maxPop = planet.MaxPopulation || "N/A";

    // Fetch resource nodes for this planet (same as planet modal)
    const resourceNodes = await this.getResourceNodes(planet.id);
    planet.resourceNodes = resourceNodes;

    // Get resource icons (await since it's async)
    const resourceIcons = await this.getResourceIcons(planet);

    // Status
    let statusHtml = "";
    if (planet.colonized_by) {
      if (isOwned) {
        statusHtml = `<span class="text-xs text-green-400 flex items-center gap-1"><span class="material-icons text-xs">check_circle</span>Your Colony</span>`;
      } else {
        statusHtml = `<span class="text-xs text-red-400 flex items-center gap-1"><span class="material-icons text-xs">block</span>${planet.colonized_by_name || "Occupied"}</span>`;
      }
    } else {
      statusHtml = `<span class="text-xs text-gray-400 flex items-center gap-1"><span class="material-icons text-xs">radio_button_unchecked</span>Uncolonized</span>`;
    }

    const container = document.createElement("div");
    container.className =
      "mb-3 p-3 bg-space-700/30 border border-space-600/50 rounded-lg hover:bg-space-650/40 transition-all duration-200 cursor-pointer";
    container.dataset.planetId = planetId;

    container.innerHTML = `
      <div class="flex items-start gap-4">
        <!-- Planet Icon -->
        <div class="flex-shrink-0">
          <div class="planet-icon-container w-16 h-16 flex items-center justify-center">
            <!-- GIF will be set via DOM manipulation -->
          </div>
        </div>

        <!-- Planet Info -->
        <div class="flex-1 min-w-0">
          <div class="flex items-start justify-between mb-2">
            <div>
              <h3 class="font-semibold text-lg text-nebula-200">${planetName}</h3>
              <div class="text-sm text-space-300">${planetTypeName} • Size ${planet.size || "N/A"}</div>
            </div>
            <div class="text-right">
              ${statusHtml}
              ${population > 0 ? `<div class="text-sm text-green-400 mt-1">${population.toLocaleString()}/${maxPop} pop</div>` : ""}
            </div>
          </div>

          <!-- Resources -->
          ${
            resourceIcons.length > 0
              ? `
            <div class="mb-2">
              <div class="text-xs text-space-400 mb-1">Resources:</div>
              <div class="flex items-center gap-1 flex-wrap">
                ${resourceIcons.join("")}
              </div>
            </div>
          `
              : '<div class="text-xs text-space-500 mb-2">No resources detected</div>'
          }


        </div>
      </div>
    `;

    // Set the planet GIF after container is created
    const iconContainer = container.querySelector(".planet-icon-container");
    if (iconContainer && planetGif) {
      iconContainer.innerHTML = planetGif;
    }

    // Make the whole container clickable
    container.onclick = () => {
      window.uiController.displayPlanetView(planet);
    };

    return container;
  }

  async displayPlanetView(planet, screenX, screenY) {
    // Added screenX, screenY
    const container = this.expandedView;
    if (!container) {
      console.error("#expanded-view-container not found in displayPlanetView");
      return;
    }

    // Fetch resource nodes for this planet
    const resourceNodes = await this.getResourceNodes(planet.id);
    planet.resourceNodes = resourceNodes;

    // Simplified check: if it's the same planet and no new coords, assume no major update needed.
    // Could be more sophisticated by checking if 'planet' data has changed.
    if (
      this.currentSystemId === planet.id &&
      container.dataset.viewType === "planet" &&
      !container.classList.contains("hidden") &&
      screenX === undefined
    ) {
      // return;
    }

    container.className = "floating-panel"; // Ensure base class is set

    // Use currentSystemId to track the currently displayed entity (system or planet)
    // This might be better named e.g. currentViewEntityId if it can be a planet ID too.
    // For now, reusing currentSystemId for simplicity, assuming planet IDs are distinct from system IDs
    // or that context (system view vs planet view) is managed by viewType.
    this.currentSystemId = planet.id;
    container.dataset.viewType = "planet";
    container.dataset.currentId = planet.id;

    const planetName = planet.name || `Planet ${planet.id.slice(-4)}`;
    const planetTypeValue = planet.planet_type || planet.type;
    const planetIcon = this.getPlanetTypeIcon(planetTypeValue);
    const planetTypeName = this.getPlanetTypeName(planetTypeValue);
    const systemName =
      planet.system_name ||
      (this.gameState &&
        this.gameState.mapData.systems.find((s) => s.id === planet.system_id)
          ?.name) ||
      planet.system_id;

    // Calculate population percentage
    const popPercentage = planet.MaxPopulation
      ? Math.round(((planet.Pop || 0) / planet.MaxPopulation) * 100)
      : 0;
    const popBarColor =
      popPercentage > 80
        ? "bg-green-500"
        : popPercentage > 50
          ? "bg-yellow-500"
          : "bg-orange-500";

    let resourcesHtml =
      '<div class="text-sm text-space-400">No resource data available.</div>';
    if (planet.Credits !== undefined) {
      resourcesHtml = `
        <div class="space-y-3">
          <!-- Population Bar -->
          <div>
            <div class="flex justify-between items-center mb-1">
              <span class="text-sm flex items-center gap-1"><span class="material-icons text-sm">people</span>Population</span>
              <span class="text-sm font-semibold">${planet.Pop?.toLocaleString() || 0} / ${planet.MaxPopulation?.toLocaleString() || "N/A"}</span>
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

    let buildingsHtml =
      '<div class="text-sm text-space-400">No buildings constructed.</div>';
    let buildingWarnings = [];
    
    if (planet.Buildings && Object.keys(planet.Buildings).length > 0) {
      // Get population assignments for this planet
      const planetPopulations = this.gameState?.populations?.filter(pop => pop.planet_id === planet.id) || [];
      const assignedBuildingIds = new Set(planetPopulations.map(pop => pop.employed_at).filter(Boolean));
      
      const buildingEntries = Object.entries(planet.Buildings)
        .map(([buildingName, level]) => {
          let displayName = buildingName;
          let buildingIcon = "🏢";
          let buildingId = buildingName;
          
          // Find the actual building record to get its ID
          const buildingRecord = this.gameState?.buildings?.find(b => 
            b.planet_id === planet.id && 
            (b.building_type === buildingName || b.id === buildingName)
          );
          
          if (buildingRecord) {
            buildingId = buildingRecord.id;
          }
          
          if (this.gameState && this.gameState.buildingTypes) {
            const buildingType = this.gameState.buildingTypes.find(
              (bt) =>
                bt.id === buildingName ||
                bt.name.toLowerCase() === buildingName.toLowerCase(),
            );
            if (buildingType) {
              displayName = buildingType.name;
              // Add icons based on building type
              if (displayName.toLowerCase().includes("farm"))
                buildingIcon = "🌾";
              else if (displayName.toLowerCase().includes("mine"))
                buildingIcon = "⛏️";
              else if (displayName.toLowerCase().includes("factory"))
                buildingIcon = "🏭";
              else if (displayName.toLowerCase().includes("bank"))
                buildingIcon = "🏦";
              else if (displayName.toLowerCase().includes("research"))
                buildingIcon = "🔬";
            }
          }
          
          // Check if this building has population assigned
          const hasWorkers = assignedBuildingIds.has(buildingId);
          const workerCount = planetPopulations.find(pop => pop.employed_at === buildingId)?.count || 0;
          
          // Add to warnings if no workers
          if (!hasWorkers) {
            buildingWarnings.push(displayName);
          }
          
          const warningIcon = hasWorkers ? '' : '<span class="text-yellow-400 text-sm ml-1" title="No workers assigned">⚠️</span>';
          const workerInfo = hasWorkers ? `<div class="text-xs text-green-400">${workerCount} workers</div>` : '<div class="text-xs text-yellow-400">No workers</div>';
          
          return `
          <li class="p-3 ${hasWorkers ? 'bg-space-700' : 'bg-yellow-900/20 border border-yellow-600/30'} rounded-md flex items-center justify-between hover:bg-space-600 transition-colors">
            <div class="flex items-center gap-2">
              <span class="text-xl">${buildingIcon}</span>
              <div>
                <div class="flex items-center">
                  <span class="font-semibold">${displayName}</span>
                  ${warningIcon}
                </div>
                ${workerInfo}
              </div>
            </div>
            <span class="text-sm text-space-300">Level ${level}</span>
          </li>
        `;
        })
        .join("");
      
      let warningHtml = '';
      if (buildingWarnings.length > 0) {
        warningHtml = `
        <div class="p-3 bg-yellow-900/20 border border-yellow-600/50 rounded-md mb-3">
          <div class="flex items-center gap-2 mb-2">
            <span class="text-yellow-400">⚠️</span>
            <span class="font-semibold text-yellow-200">Production Warning</span>
          </div>
          <div class="text-sm text-yellow-100">
            ${buildingWarnings.length} building${buildingWarnings.length > 1 ? 's' : ''} without workers: 
            <strong>${buildingWarnings.join(', ')}</strong>
          </div>
          <div class="text-xs text-yellow-300 mt-1">
            Buildings without assigned population will not produce resources.
          </div>
        </div>`;
      }
      
      buildingsHtml = `${warningHtml}<ul class="space-y-2">${buildingEntries}</ul>`;
    }

    const isColonized = planet.colonized_by && planet.colonized_by !== "";
    const isOwnedByPlayer =
      isColonized && planet.colonized_by === this.currentUser?.id;
    const canColonize = !isColonized && this.currentUser && this.gameState;

    // Redraw inner content structure
    container.innerHTML = `
      <div class="floating-panel-content">
        <div id="planet-header" class="panel-header flex justify-between items-center p-3 cursor-move border-b border-space-700/50 bg-gradient-to-r from-nebula-900/30 to-plasma-900/30" draggable="false">
          <div class="flex items-center gap-2">
            <span class="material-icons text-space-400 drag-handle">drag_indicator</span>
            <span id="planet-icon" class="text-2xl"></span>
            <span id="planet-name" class="text-xl font-bold text-nebula-200"></span>
          </div>
          <div class="flex items-center gap-4">
            <div class="text-right">
              <div id="planet-seed" class="font-semibold text-nebula-200 text-sm"></div>
              <div id="planet-system" class="font-mono text-xs text-gray-500"></div>
            </div>
            <button onclick="window.uiController.clearExpandedView()"
                    class="btn-icon hover:bg-space-700 rounded">
              <span class="material-icons text-sm">close</span>
            </button>
          </div>
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
    const planetTypeGradient = this.getPlanetTypeGradient(planetTypeValue);
    const planetHeader = container.querySelector("#planet-header");
    planetHeader.className = `panel-header flex justify-between items-center p-3 cursor-move border-b border-space-700/50 bg-gradient-to-r ${planetTypeGradient}`;

    container.querySelector("#planet-icon").innerHTML =
      this.getPlanetAnimatedGif(planetTypeName) || planetIcon;
    container.querySelector("#planet-name").textContent = planetName;
    container.querySelector("#planet-seed").textContent =
      `Seed: ${planet.id.slice(-8)}`;
    container.querySelector("#planet-system").textContent = systemName;
    container.querySelector("#planet-type-size").innerHTML = `
      <div class="text-center">
        <div class="font-medium">${planetTypeName}</div>
        <div class="text-xs text-space-400">Size ${planet.size || "N/A"} • ${systemName}</div>
      </div>
    `;

    // Update resource icons
    const resourceIcons = await this.getResourceIcons(planet);
    container.querySelector("#planet-resources-icons").innerHTML =
      resourceIcons.join("");

    // Generate resource nodes HTML
    let resourceNodesHtml =
      '<div class="text-sm text-space-400">No resource deposits detected.</div>';
    if (resourceNodes && resourceNodes.length > 0) {
      // Get all resource types for lookup (since expand might not be working)
      const resourceTypesPromise = this.loadResourceTypes();
      const resourceTypes = await resourceTypesPromise;
      const resourceTypeMap = {};
      resourceTypes.forEach((rt) => {
        resourceTypeMap[rt.id] = rt;
      });

      const resourcesByType = {};
      resourceNodes.forEach((node) => {
        let resourceType, resourceTypeData;

        // Check if expand worked
        if (node.expand && node.expand.resource_type) {
          resourceType = node.expand.resource_type.name;
          resourceTypeData = node.expand.resource_type;
        } else {
          // Fallback: use the resource type ID to lookup
          resourceTypeData = resourceTypeMap[node.resource_type];
          resourceType = resourceTypeData
            ? resourceTypeData.name
            : node.resource_type;
        }

        if (resourceType) {
          if (!resourcesByType[resourceType]) {
            resourcesByType[resourceType] = { nodes: [], resourceTypeData };
          }
          resourcesByType[resourceType].nodes.push(node);
        }
      });

      const resourceEntries = Object.entries(resourcesByType)
        .map(([resourceType, data]) => {
          const { nodes, resourceTypeData } = data;
          const totalRichness = nodes.reduce(
            (sum, node) => sum + node.richness,
            0,
          );
          const avgRichness = (totalRichness / nodes.length).toFixed(1);
          const nodeCount = nodes.length;

          const iconUrl = resourceTypeData?.icon || "/placeholder-planet.svg";

          return `
          <li class="p-3 bg-space-700 rounded-md flex items-center justify-between hover:bg-space-600 transition-colors">
            <div class="flex items-center gap-3">
              <img src="${iconUrl}" class="w-6 h-6" title="${resourceType}" alt="${resourceType}" />
              <div>
                <span class="font-semibold">${resourceType}</span>
                <div class="text-xs text-space-400">${nodeCount} deposit${nodeCount > 1 ? "s" : ""}</div>
              </div>
            </div>
            <div class="text-right">
              <div class="text-sm font-medium">Richness: ${avgRichness}</div>
              <div class="text-xs text-space-400">Total: ${totalRichness}</div>
            </div>
          </li>
        `;
        })
        .join("");
      resourceNodesHtml = `<ul class="space-y-2">${resourceEntries}</ul>`;
    }

    // Update resources and buildings
    const resourcesContainer = container.querySelector(
      "#planet-resources-container",
    );
    const buildingsContainer = container.querySelector(
      "#planet-buildings-container",
    );

    // Always show resource nodes for any planet
    container.querySelector("#planet-resources-html").innerHTML =
      resourceNodesHtml;
    resourcesContainer.style.display = "block";

    // Update the section title to reflect resource nodes
    const resourcesTitle = resourcesContainer.querySelector("h3");
    if (resourcesTitle) {
      resourcesTitle.textContent = "Resource Deposits";
    }

    if (isOwnedByPlayer) {
      container.querySelector("#planet-buildings-html").innerHTML =
        buildingsHtml;
      buildingsContainer.style.display = "block";

      // --- CONSTRUCTION QUEUE SECTION REMOVED ---
      // Building construction queue display is deferred as per recent changes.
      // The logic previously here for `buildQueueContainer` that used `window.gameState.orders`
      // and filtered for `building_construct` has been removed.
    } else {
      buildingsContainer.style.display = "none";
    }

    // Update action buttons
    const actionsContainer = container.querySelector(
      "#planet-actions-container",
    );
    actionsContainer.innerHTML = ""; // Clear previous buttons

    if (canColonize) {
      const availableSettlerFleets = this.getAvailableSettlerFleets(
        planet.system_id,
      );

      if (availableSettlerFleets.length > 0) {
        const colonizeButton = document.createElement("button");
        colonizeButton.className =
          "w-full btn btn-success py-3 flex items-center justify-center gap-2";
        colonizeButton.innerHTML = `<span class="material-icons">rocket_launch</span> Colonize Planet (Settler Ship)`;
        colonizeButton.onclick = () =>
          window.uiController.colonizePlanetWrapper(planet.id);
        actionsContainer.appendChild(colonizeButton);
      }
    } else if (!isColonized && this.currentUser && this.gameState) {
      const noSettlerButton = document.createElement("button");
      noSettlerButton.className =
        "w-full btn btn-disabled py-3 flex items-center justify-center gap-2";
      noSettlerButton.innerHTML = `<span class="material-icons">rocket_launch</span> Colonize Planet (Need Settler Ship)`;
      noSettlerButton.disabled = true;
      actionsContainer.appendChild(noSettlerButton);
    }

    if (isOwnedByPlayer) {
      const constructButton = document.createElement("button");
      constructButton.className =
        "w-full btn btn-primary py-3 flex items-center justify-center gap-2";
      constructButton.innerHTML = `<span>🏗️</span> Construct Building`;
      constructButton.onclick = () =>
        window.uiController.showPlanetBuildModal(planet);
      actionsContainer.appendChild(constructButton);
    }

    const backButton = document.createElement("button");
    backButton.className =
      "w-full btn btn-secondary py-3 flex items-center justify-center gap-2";
    backButton.textContent = "← Back to System";
    backButton.onclick = () =>
      window.uiController.goBackToSystemView(planet.system_id);
    actionsContainer.appendChild(backButton);

    container.classList.remove("hidden"); // Make sure it's not hidden before positioning
    if (screenX !== undefined && screenY !== undefined) {
      this.positionPanel(container, screenX, screenY);
    } else if (
      container.style.left === "-2000px" ||
      container.style.left === "-9999px" ||
      !container.style.left
    ) {
      container.style.top = "20px";
      container.style.left = "20px";
      container.style.right = "auto";
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
      console.warn(
        "Building types not available or empty in gameState for showPlanetBuildModal.",
      );
      this.showModal(
        `Construct on ${planet.name || `Planet ${planet.id.slice(-4)}`}`,
        `<div class="text-space-400">No building types available or data is still loading.</div>
         <button class="w-full mt-2 btn btn-secondary" onclick="window.uiController.hideModal()">Close</button>`,
      );
      return;
    }

    // Get available resource types on this planet
    const availableResources = new Set();
    if (planet.resourceNodes && planet.resourceNodes.length > 0) {
      planet.resourceNodes.forEach((node) => {
        if (node.expand && node.expand.resource_type) {
          availableResources.add(node.expand.resource_type.name.toLowerCase());
        }
      });
    }

    // Get fleets at this system for resource checking
    const systemId = planet.system_id;
    const currentUser = this.currentUser;
    const availableFleets =
      this.gameState?.fleets?.filter(
        (fleet) =>
          fleet.current_system === systemId &&
          fleet.owner_id === currentUser?.id,
      ) || [];

    const buildingOptions = buildingTypes
      .map((buildingType) => {
        let costString = "Cost: ";
        let canBuild = true;
        let missingResources = [];

        // Handle new resource cost system
        if (buildingType.cost_resource_type && buildingType.cost_quantity > 0) {
          const resourceName =
            buildingType.cost_resource_name || "Unknown Resource";
          costString += `${buildingType.cost_quantity} ${resourceName}`;

          // Check if any fleet at this system has enough resources
          let hasEnoughResources = false;
          if (availableFleets.length > 0) {
            // For now, assume fleets have resources (we'll implement cargo checking later)
            hasEnoughResources = true;
          }

          if (!hasEnoughResources) {
            canBuild = false;
            missingResources.push(
              `${buildingType.cost_quantity} ${resourceName}`,
            );
          }
        } else if (buildingType.cost > 0) {
          // Legacy credit system
          costString += `${buildingType.cost} Credits`;
        } else {
          costString += "Free";
        }

        // Check if building requires specific resources
        const requiresResources =
          buildingType.resource_nodes && buildingType.resource_nodes.length > 0;
        if (requiresResources) {
          const requiredResourceIds = buildingType.resource_nodes;
          const resourceTypesMap = (this.gameState?.resourceTypes || []).reduce(
            (map, rt) => {
              map[rt.id] = rt.name;
              return map;
            },
            {},
          );

          requiredResourceIds.forEach((resourceId) => {
            const resourceName = resourceTypesMap[resourceId] || resourceId;
            if (!availableResources.has(resourceName.toLowerCase())) {
              canBuild = false;
              missingResources.push(resourceName);
            }
          });
        }

        const safePlanetId = planet.id.replace(/'/g, "\\'");
        const safeBuildingTypeId = buildingType.id.replace(/'/g, "\\'");

        const buttonClass = canBuild
          ? "w-full p-3 bg-space-700 hover:bg-space-600 rounded mb-2 text-left cursor-pointer"
          : "w-full p-3 bg-space-800 rounded mb-2 text-left cursor-not-allowed opacity-60";

        const onclickHandler =
          canBuild && availableFleets.length > 0
            ? `onclick="window.gameState.queueBuilding('${safePlanetId}', '${safeBuildingTypeId}', '${availableFleets[0].id}'); window.uiController.hideModal();"`
            : "";

        // Update canBuild to require fleets
        if (availableFleets.length === 0) {
          canBuild = false;
          missingResources.push("Fleet at this system");
        }

        let requirementsText = "";
        if (!canBuild && missingResources.length > 0) {
          requirementsText = `<div class="text-xs text-red-400 mt-1">Missing: ${missingResources.join(", ")}</div>`;
        }

        return `
      <button class="${buttonClass}" ${onclickHandler} ${!canBuild ? "disabled" : ""}>
        <div class="font-semibold ${canBuild ? "" : "text-space-400"}">${buildingType.name || "Unknown Building"}</div>
        <div class="text-sm text-space-300">${buildingType.description || "No description available."}</div>
        <div class="text-sm">${costString}</div>
        ${requirementsText}
      </button>
    `;
      })
      .join("");

    // Show resource summary
    const resourceSummary =
      availableResources.size > 0
        ? `<div class="mb-4 p-3 bg-space-800 rounded">
           <div class="text-sm font-semibold mb-2">Available Resources:</div>
           <div class="text-xs text-space-300">${Array.from(availableResources)
             .map((r) => r.charAt(0).toUpperCase() + r.slice(1))
             .join(", ")}</div>
         </div>`
        : `<div class="mb-4 p-3 bg-space-800 rounded">
           <div class="text-sm text-red-400">No resource deposits found on this planet.</div>
         </div>`;

    this.showModal(
      `Construct on ${planet.name || `Planet ${planet.id.slice(-4)}`}`,
      `
      ${resourceSummary}
      <div class="space-y-2 max-h-96 overflow-y-auto">
        ${buildingOptions.length > 0 ? buildingOptions : '<div class="text-space-400">No buildings available to construct.</div>'}
      </div>
      <button class="w-full mt-4 btn btn-secondary" onclick="window.uiController.hideModal()">Cancel</button>
    `,
    );
  }

  // Helper method to get fleets with settler ships at a specific system
  getAvailableSettlerFleets(systemId) {
    if (!this.gameState || !this.gameState.fleets) {
      return [];
    }

    return this.gameState.fleets.filter((fleet) => {
      // Fleet must be at the target system
      if (fleet.current_system !== systemId) {
        return false;
      }

      // Fleet must be owned by the current user
      if (fleet.owner_id !== this.currentUser?.id) {
        return false;
      }

      // Check if fleet has settler ships
      return (
        fleet.ships &&
        fleet.ships.some(
          (ship) => ship.ship_type_name === "settler" && ship.count > 0,
        )
      );
    });
  }

  // Wrapper for colonizePlanet to fit new UI structure if needed
  colonizePlanetWrapper(planetId) {
    // Find planet data again, or ensure it's correctly passed
    // For simplicity, assuming colonizePlanet can fetch necessary data or is adapted
    if (
      !this.gameState ||
      !this.gameState.mapData ||
      !this.gameState.mapData.planets
    ) {
      this.showError("Game data not loaded. Cannot colonize.");
      return;
    }
    const planet = this.gameState.mapData.planets.find(
      (p) => p.id === planetId,
    );
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
        const planet = this.gameState?.mapData?.planets?.find(
          (p) => p.id === planetId,
        );
        if (!planet) {
          this.showError("Planet not found in game data");
          return;
        }
        targetSystemId = planet.system_id;
      }

      const availableSettlerFleets =
        this.getAvailableSettlerFleets(targetSystemId);

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
        if (
          this.gameState &&
          this.gameState.selectedSystem &&
          this.gameState.selectedSystem.id === targetSystemId
        ) {
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

  goBackToSystemView() {
    const system = this.gameState.getSelectedSystem();
    if (system) {
      let planetsInSystem = [];
      if (this.gameState.mapData.planets) {
        planetsInSystem = this.gameState.mapData.planets.filter((p) => {
          if (Array.isArray(p.system_id))
            return p.system_id.includes(system.id);
          return p.system_id === system.id;
        });
      }
      this.displaySystemView(system, planetsInSystem);
    } else {
      this.clearExpandedView();
    }
  }

  async displayFleetView(fleet, screenX, screenY) {
    const container = this.expandedView;
    if (!container) {
      console.error("#expanded-view-container not found in displayFleetView");
      return;
    }

    // Preserve pinned state while clearing content
    const wasPinned = container._isPinned;
    const pinnedClasses = container.classList.contains("pinned-panel");
    
    // Clear content but preserve state
    if (!wasPinned) {
      this.clearExpandedView();
    }

    // Set up the container
    container.classList.remove("hidden");
    container.classList.add("floating-panel", "glass-panel");
    
    // Restore pinned state if it was pinned
    if (wasPinned) {
      container.classList.add("pinned-panel");
      container._isPinned = true;
    }

    // Render fleet content using FleetComponent
    const fleetContent = this.renderFleetView(fleet);
    const fleetName = fleet.name || `Fleet ${fleet.id.slice(-4)}`;

    container.innerHTML = `
      <div class="floating-panel-content">
        ${fleetContent}
      </div>
    `;

    // Position the panel near the fleet if coordinates provided (only if not pinned)
    if (screenX !== null && screenY !== null && !container._isPinned) {
      this.positionPanel(container, screenX, screenY);
    }

    // Set up dragging functionality
    this.setupPanelDragging(container);
    
    // Add focus effect
    this.addPanelFocusEffects(container);
  }

  setupPanelDragging(container) {
    this.makePanelDraggable(container);
  }

  addPanelFocusEffects(container) {
    // Add focus when clicking on panel
    container.addEventListener('mousedown', () => {
      // Remove focused class from other panels
      document.querySelectorAll('.floating-panel.focused').forEach(panel => {
        if (panel !== container) {
          panel.classList.remove('focused');
        }
      });
      
      // Add focused class to this panel
      container.classList.add('focused');
    });

    // Remove focus when clicking outside (unless pinned)
    document.addEventListener('click', (e) => {
      if (!container.contains(e.target) && !container._isPinned) {
        container.classList.remove('focused');
      }
    });
  }

  togglePinPanel(container) {
    if (!container) return;
    
    const isPinned = container.classList.contains('pinned-panel');
    const pinButton = container.querySelector('.pin-button');
    const pinIcon = pinButton?.querySelector('.material-icons');
    
    if (isPinned) {
      // Unpin the panel
      container.classList.remove('pinned-panel');
      container._isPinned = false;
      if (pinButton) {
        pinButton.classList.remove('pinned');
        pinButton.title = 'Pin';
      }
      if (pinIcon) {
        pinIcon.textContent = 'push_pin';
        pinIcon.style.transform = '';
      }
    } else {
      // Pin the panel
      container.classList.add('pinned-panel');
      container._isPinned = true;
      if (pinButton) {
        pinButton.classList.add('pinned');
        pinButton.title = 'Unpin';
      }
      if (pinIcon) {
        pinIcon.textContent = 'push_pin';
      }
    }
  }

  renderFleetView(fleet) {
    if (!this.fleetComponentManager) {
      return `
        <div class="p-4 bg-space-700 rounded-lg border border-space-600">
          <h3 class="text-red-400 font-semibold mb-2">Fleet System Not Available</h3>
          <p class="text-space-300 text-sm">Fleet management system is not initialized.</p>
        </div>
      `;
    }

    // Use the FleetComponent to render the fleet details
    const fleetComponent = this.fleetComponentManager.fleetComponent;
    return fleetComponent.render(fleet.id);
  }

  manageColony(planetId) {
    // Placeholder for managing colony - could open build modal for this planet
    // For now, let's try to open the general build modal, but ideally it would be context-aware
    // This will require finding the system for this planet first
    if (
      !this.gameState ||
      !this.gameState.mapData ||
      !this.gameState.mapData.planets
    ) {
      this.showError("Game data not loaded. Cannot manage colony.");
      return;
    }
    const planet = this.gameState.mapData.planets.find(
      (p) => p.id === planetId,
    );
    if (!planet) {
      this.showError("Planet data not found.");
      return;
    }
    const system = this.gameState.mapData.systems.find(
      (s) => s.id === planet.system_id,
    );
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
    
    // Initialize FleetComponentManager if not already done
    if (!this.fleetComponentManager && this.currentUser) {
      this.fleetComponentManager = new FleetComponentManager(this, state);
    } else if (this.fleetComponentManager) {
      this.fleetComponentManager.updateGameState(state);
    }
    
    this.updateResourcesUI(state.playerResources);
    // this.updateSystemInfoUI(state.selectedSystem); // Removed: Replaced by expanded-view-container logic
    this.updateGameStatusUI(state);
  }

  loadResourcePreferences() {
    try {
      const stored = localStorage.getItem("xan_displayed_resources");
      return stored ? JSON.parse(stored) : ["credits", "ore"]; // Default to credits and ore
    } catch (error) {
      return ["credits", "ore"];
    }
  }

  saveResourcePreferences() {
    try {
      localStorage.setItem(
        "xan_displayed_resources",
        JSON.stringify(this.displayedResources),
      );
    } catch (error) {
      console.warn("Failed to save resource preferences:", error);
    }
  }

  initializeResourcesDropdown() {
    // Set up resources dropdown toggle
    document.addEventListener('click', (e) => {
      const dropdown = document.getElementById('resources-dropdown');
      const toggle = document.getElementById('resources-toggle');
      
      if (e.target.closest('#resources-toggle')) {
        if (dropdown.classList.contains('hidden')) {
          // Position dropdown near the toggle button like context menu
          const toggleRect = toggle.getBoundingClientRect();
          const canvasRect = document.getElementById('game-canvas').getBoundingClientRect();
          
          // Position relative to canvas
          const x = toggleRect.right - canvasRect.left - 256; // 256px = dropdown width
          const y = toggleRect.bottom - canvasRect.top + 8; // 8px gap
          
          dropdown.style.left = `${x}px`;
          dropdown.style.top = `${y}px`;
          dropdown.classList.remove('hidden');
        } else {
          dropdown.classList.add('hidden');
        }
      } else if (!e.target.closest('#resources-dropdown')) {
        dropdown.classList.add('hidden');
      }
    });

    // Set up settings button
    document.addEventListener('click', (e) => {
      if (e.target.closest('#resources-settings')) {
        this.showResourcesSettingsModal();
      }
    });
  }

  updateResourcesDropdown() {
    const resourcesList = document.getElementById("resources-list");
    if (!resourcesList) return;

    const resources = this.gameState?.playerResources || {};
    resourcesList.innerHTML = "";

    // Define all possible resources with their default icons
    const allResources = [
      {
        name: "credits",
        icon: "account_balance_wallet",
        color: "text-nebula-300",
      },
      { name: "ore", icon: "construction", color: "text-orange-400" },
      { name: "food", icon: "restaurant", color: "text-green-400" },
      { name: "fuel", icon: "local_gas_station", color: "text-blue-400" },
      { name: "metal", icon: "build", color: "text-gray-400" },
      {
        name: "titanium",
        icon: "precision_manufacturing",
        color: "text-purple-400",
      },
      { name: "xanium", icon: "auto_awesome", color: "text-yellow-400" },
    ];

    // Add each resource type
    for (const resourceDef of allResources) {
      const amount = resources[resourceDef.name] || 0;

      if (resourceDef.name === "credits") {
        // Special handling for credits with income display
        const incomeText =
          this.gameState?.creditIncome > 0
            ? ` (+${this.gameState.creditIncome}/tick)`
            : "";
        resourcesList.innerHTML += `
          <div class="flex items-center justify-between py-1">
            <div class="flex items-center gap-2">
              <span class="material-icons text-sm ${resourceDef.color}">${resourceDef.icon}</span>
              <span class="text-sm">Credits</span>
            </div>
            <div class="text-sm font-mono">
              <span class="${resourceDef.color}">${amount.toLocaleString()}</span>
              ${incomeText ? `<span class="text-xs text-plasma-300">${incomeText}</span>` : ""}
            </div>
          </div>
        `;
      } else {
        // Get resource icon from resource types, fall back to default
        const resourceType = this.resourceTypes.get(
          resourceDef.name.toLowerCase(),
        );
        const icon = resourceType?.icon
          ? `<img src="${resourceType.icon}" class="w-4 h-4" alt="${resourceDef.name}" />`
          : `<span class="material-icons text-sm ${resourceDef.color}">${resourceDef.icon}</span>`;

        resourcesList.innerHTML += `
          <div class="flex items-center justify-between py-1">
            <div class="flex items-center gap-2">
              ${icon}
              <span class="text-sm capitalize">${resourceDef.name}</span>
            </div>
            <span class="text-sm font-mono text-space-200">${amount.toLocaleString()}</span>
          </div>
        `;
      }
    }
  }

  updateResourcesUI(resources) {
    // Update main resources display
    const resourcesDisplay = document.getElementById("resources-display");
    if (!resourcesDisplay) return;

    // Clear current display
    resourcesDisplay.innerHTML = "";

    // Add each displayed resource
    for (const resourceName of this.displayedResources) {
      const amount = resources[resourceName];
      if (amount === undefined) continue;

      if (resourceName === "credits") {
        // Special handling for credits
        resourcesDisplay.innerHTML += `
          <button id="credits-btn" class="text-nebula-300 hover:text-nebula-200 hover:bg-nebula-900/20 px-2 py-1 rounded transition-all cursor-pointer border border-nebula-600/30 hover:border-nebula-500/50 flex items-center gap-1">
            <span class="material-icons text-base">account_balance_wallet</span>
            <span id="credits">${amount?.toLocaleString() || 0}</span><span id="credit-income" class="text-xs text-plasma-300 ml-1"></span>
          </button>
        `;

        // Set up credit income display
        const incomeElement = document.getElementById("credit-income");
        if (incomeElement) {
          if (this.gameState?.creditIncome > 0) {
            incomeElement.textContent = `(+${this.gameState.creditIncome}/tick)`;
            incomeElement.style.display = "inline";
          } else {
            incomeElement.style.display = "none";
          }
        }

        // Set up credits button click handler for breakdown modal
        const creditsBtn = document.getElementById("credits-btn");
        if (creditsBtn) {
          creditsBtn.onclick = () => this.showCreditsBreakdown();
        }
      } else {
        // Other resources
        const resourceDef = this.getResourceDefinition(resourceName);
        const resourceType = this.resourceTypes.get(resourceName.toLowerCase());
        const icon = resourceType?.icon
          ? `<img src="${resourceType.icon}" class="w-4 h-4" alt="${resourceName}" />`
          : `<span class="material-icons text-base">${resourceDef.icon}</span>`;

        const colorClass = resourceDef.color || "text-space-300";

        resourcesDisplay.innerHTML += `
          <button class="${colorClass} hover:text-space-200 hover:bg-space-900/20 px-2 py-1 rounded transition-all cursor-pointer border border-space-600/30 hover:border-space-500/50 flex items-center gap-1">
            ${icon}
            <span class="font-mono">${amount?.toLocaleString() || 0}</span>
          </button>
        `;
      }
    }

    // Update dropdown display
    this.updateResourcesDropdown();
  }

  getResourceDefinition(resourceName) {
    const resourceDefs = {
      credits: { icon: "account_balance_wallet", color: "text-nebula-300" },
      ore: { icon: "construction", color: "text-orange-400" },
      food: { icon: "restaurant", color: "text-green-400" },
      fuel: { icon: "local_gas_station", color: "text-blue-400" },
      metal: { icon: "build", color: "text-gray-400" },
      titanium: { icon: "precision_manufacturing", color: "text-purple-400" },
      xanium: { icon: "auto_awesome", color: "text-yellow-400" },
    };
    return (
      resourceDefs[resourceName] || {
        icon: "inventory",
        color: "text-space-300",
      }
    );
  }

  showResourcesSettingsModal() {
    // Define all possible resources for settings
    const allResources = [
      { name: "credits", icon: "account_balance_wallet" },
      { name: "ore", icon: "construction" },
      { name: "food", icon: "restaurant" },
      { name: "fuel", icon: "local_gas_station" },
      { name: "metal", icon: "build" },
      { name: "titanium", icon: "precision_manufacturing" },
      { name: "xanium", icon: "auto_awesome" },
    ];

    const resourceOptions = allResources
      .map((resourceDef) => {
        const isDisplayed = this.displayedResources.includes(resourceDef.name);
        const resourceType = this.resourceTypes.get(resourceDef.name);
        const icon = resourceType?.icon
          ? `<img src="${resourceType.icon}" class="w-4 h-4" alt="${resourceDef.name}" />`
          : `<span class="material-icons text-sm">${resourceDef.icon}</span>`;

        return `
        <label class="flex items-center gap-3 p-2 hover:bg-space-700/50 rounded cursor-pointer">
          <input type="checkbox" ${isDisplayed ? "checked" : ""}
                 class="resource-checkbox" data-resource="${resourceDef.name}">
          <div class="flex items-center gap-2">
            ${icon}
            <span class="capitalize">${resourceDef.name}</span>
          </div>
        </label>
      `;
      })
      .join("");

    this.showModal(
      "Resource Display Settings",
      `
      <form id="resources-settings-form" class="space-y-4">
        <div>
          <p class="text-sm text-space-400 mb-3">Choose which resources to display in the top bar:</p>
          <div class="space-y-1 max-h-60 overflow-y-auto">
            ${resourceOptions}
          </div>
        </div>
        <div class="flex space-x-2">
          <button type="submit" class="flex-1 btn btn-success">
            Save Settings
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
      .getElementById("resources-settings-form")
      .addEventListener("submit", (e) => {
        e.preventDefault();

        const checkboxes = document.querySelectorAll(".resource-checkbox");
        this.displayedResources = Array.from(checkboxes)
          .filter((cb) => cb.checked)
          .map((cb) => cb.dataset.resource);

        // Ensure at least one resource is displayed
        if (this.displayedResources.length === 0) {
          this.displayedResources = ["credits"];
        }

        this.saveResourcePreferences();
        this.updateResourcesUI(this.gameState?.playerResources || {});
        this.hideModal();
      });
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
      if (prevTick !== newTick && prevTick !== "Tick: 0") {
        // Avoid flash on initial load
        tickElement.style.animation = "none";
        tickElement.offsetHeight; // Trigger reflow
        tickElement.style.animation = "flash 0.5s ease-out";
      }
    }

    // Update tick rate display (this part of the logic might be combined with startTickTimer or be static if only countdown changes)
    const nextTickRateElement = document.getElementById("next-tick-display");
    if (nextTickRateElement && !this.tickTimer) {
      // Only set this if timer isn't running
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
        if (nextTickDisplayElement)
          nextTickDisplayElement.textContent = "Next Tick: Processing...";
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

    // Add glass effect classes to modal
    modalContent.classList.add("modal-content");

    modalContent.innerHTML = `
      <div class="modal-header panel-header cursor-move flex justify-between items-center p-4">
        <div class="flex items-center gap-2">
          <span class="material-icons drag-handle text-space-400">drag_indicator</span>
          <h2 class="text-xl font-bold">${title}</h2>
        </div>
        <div class="flex items-center gap-2">
          <button onclick="window.uiController.togglePinModal()" 
                  class="pin-button text-space-400 hover:text-white transition-colors" 
                  title="Pin">
            <span class="material-icons text-sm">push_pin</span>
          </button>
          <button id="modal-close" class="text-space-400 hover:text-white transition-colors" title="Close">
            <span class="material-icons">close</span>
          </button>
        </div>
      </div>
      <div class="modal-body p-4">
        ${content}
      </div>
    `;

    modalOverlay.classList.remove("hidden");

    // Set up close button
    document.getElementById("modal-close").addEventListener("click", () => {
      this.hideModal();
    });

    // Make modal draggable
    this.makePanelDraggable(modalContent);
    
    // Add focus effects
    this.addPanelFocusEffects(modalContent);
    
    // Reset position to center
    modalContent.style.left = '';
    modalContent.style.top = '';
    modalContent.style.transform = 'translate(-50%, -50%)';
  }

  hideModal() {
    const modalOverlay = document.getElementById("modal-overlay");
    const modalContent = document.getElementById("modal-content");
    
    // Don't close if modal is pinned
    if (modalContent._isPinned) {
      return;
    }
    
    // Clean up drag handlers
    if (modalContent._dragCleanup) {
      modalContent._dragCleanup();
      delete modalContent._dragCleanup;
    }
    
    modalOverlay.classList.add("hidden");
    modalContent.classList.remove("focused", "pinned-panel");
    modalContent._isPinned = false;
  }

  togglePinModal() {
    const modalContent = document.getElementById("modal-content");
    this.togglePinPanel(modalContent);
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
          const resourceTypesMap = (this.gameState?.resourceTypes || []).reduce(
            (map, rt) => {
              map[rt.id] = rt.name;
              return map;
            },
            {},
          );
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
    if (!this.fleetComponentManager) {
      this.showModal(
        "Your Fleets",
        '<div class="text-space-400">Fleet system not initialized.</div>',
      );
      return;
    }
    
    // Use floating panel instead of modal
    this.showFleetPanelAsFloating();
  }

  showFleetPanelAsFloating() {
    const container = this.expandedView;
    if (!container) {
      console.error("#expanded-view-container not found in showFleetPanelAsFloating");
      return;
    }

    // Preserve pinned state while clearing content
    const wasPinned = container._isPinned;
    
    // Clear content but preserve state
    if (!wasPinned) {
      this.clearExpandedView();
    }

    // Set up the container
    container.classList.remove("hidden");
    container.classList.add("floating-panel", "glass-panel");
    
    // Restore pinned state if it was pinned
    if (wasPinned) {
      container.classList.add("pinned-panel");
      container._isPinned = true;
    }

    // Get fleet content
    const content = this.fleetComponentManager.fleetListComponent.render();
    const title = this.fleetComponentManager.getFleetCount();

    container.innerHTML = `
      <div class="panel-header flex items-center justify-between p-3 cursor-move border-b border-space-600/30">
        <div class="flex items-center gap-2">
          <span class="material-icons drag-handle text-space-400">drag_indicator</span>
          <h2 class="text-lg font-semibold text-white">${title}</h2>
        </div>
        <div class="flex items-center gap-2">
          <button onclick="window.uiController.togglePinPanel(this.closest('.floating-panel'))" 
                  class="pin-button text-space-400 hover:text-white transition-colors" 
                  title="Pin">
            <span class="material-icons text-sm">push_pin</span>
          </button>
          <button onclick="window.uiController.clearExpandedView()" 
                  class="text-space-400 hover:text-white transition-colors"
                  title="Close">
            <span class="material-icons">close</span>
          </button>
        </div>
      </div>
      <div class="floating-panel-content p-4">
        ${content}
      </div>
    `;

    // Position the panel (only if not pinned)
    if (!container._isPinned) {
      container.style.top = "20px";
      container.style.left = "20px";
      container.style.right = "auto";
    }

    // Set up dragging functionality
    this.setupPanelDragging(container);
    
    // Add focus effects
    this.addPanelFocusEffects(container);
  }

  showFleetDetailsAsFloating(fleetId) {
    const container = this.expandedView;
    if (!container) {
      console.error("#expanded-view-container not found in showFleetDetailsAsFloating");
      return;
    }

    // Preserve pinned state while clearing content
    const wasPinned = container._isPinned;
    
    // Clear content but preserve state
    if (!wasPinned) {
      this.clearExpandedView();
    }

    // Set up the container
    container.classList.remove("hidden");
    container.classList.add("floating-panel", "glass-panel");
    
    // Restore pinned state if it was pinned
    if (wasPinned) {
      container.classList.add("pinned-panel");
      container._isPinned = true;
    }

    // Get fleet content
    const content = this.fleetComponentManager.fleetComponent.render(fleetId);
    const fleet = this.fleetComponentManager.gameState.fleets?.find(f => f.id === fleetId);
    const fleetName = fleet ? (fleet.name || `Fleet ${fleet.id.slice(-4)}`) : 'Unknown Fleet';

    container.innerHTML = `
      <div class="panel-header flex items-center justify-between p-3 cursor-move border-b border-space-600/30">
        <div class="flex items-center gap-2">
          <span class="material-icons drag-handle text-space-400">drag_indicator</span>
          <h2 class="text-lg font-semibold text-white">${fleetName}</h2>
        </div>
        <div class="flex items-center gap-2">
          <button onclick="window.uiController.togglePinPanel(this.closest('.floating-panel'))" 
                  class="pin-button text-space-400 hover:text-white transition-colors" 
                  title="Pin">
            <span class="material-icons text-sm">push_pin</span>
          </button>
          <button onclick="window.uiController.clearExpandedView()" 
                  class="text-space-400 hover:text-white transition-colors"
                  title="Close">
            <span class="material-icons">close</span>
          </button>
        </div>
      </div>
      <div class="floating-panel-content p-4">
        ${content}
      </div>
    `;

    // Position the panel (only if not pinned)
    if (!container._isPinned) {
      container.style.top = "20px";
      container.style.left = "20px";
      container.style.right = "auto";
    }

    // Set up dragging functionality
    this.setupPanelDragging(container);
    
    // Add focus effects
    this.addPanelFocusEffects(container);
  }

  showShipDetailsAsFloating(shipId) {
    const container = this.expandedView;
    if (!container) {
      console.error("#expanded-view-container not found in showShipDetailsAsFloating");
      return;
    }

    // Preserve pinned state while clearing content
    const wasPinned = container._isPinned;
    
    // Clear content but preserve state
    if (!wasPinned) {
      this.clearExpandedView();
    }

    // Set up the container
    container.classList.remove("hidden");
    container.classList.add("floating-panel", "glass-panel");
    
    // Restore pinned state if it was pinned
    if (wasPinned) {
      container.classList.add("pinned-panel");
      container._isPinned = true;
    }

    // Get ship content
    const content = this.fleetComponentManager.shipComponent.render(shipId);
    const ship = this.fleetComponentManager.findShip(shipId);
    const shipName = ship ? `${ship.count || 1}x ${ship.ship_type_name || 'Unknown'}` : 'Unknown Ship';

    container.innerHTML = `
      <div class="panel-header flex items-center justify-between p-3 cursor-move border-b border-space-600/30">
        <div class="flex items-center gap-2">
          <span class="material-icons drag-handle text-space-400">drag_indicator</span>
          <h2 class="text-lg font-semibold text-white">${shipName}</h2>
        </div>
        <div class="flex items-center gap-2">
          <button onclick="window.uiController.togglePinPanel(this.closest('.floating-panel'))" 
                  class="pin-button text-space-400 hover:text-white transition-colors" 
                  title="Pin">
            <span class="material-icons text-sm">push_pin</span>
          </button>
          <button onclick="window.uiController.clearExpandedView()" 
                  class="text-space-400 hover:text-white transition-colors"
                  title="Close">
            <span class="material-icons">close</span>
          </button>
        </div>
      </div>
      <div class="floating-panel-content p-4">
        ${content}
      </div>
    `;

    // Position the panel (only if not pinned)
    if (!container._isPinned) {
      container.style.top = "20px";
      container.style.left = "20px";
      container.style.right = "auto";
    }

    // Set up dragging functionality
    this.setupPanelDragging(container);
    
    // Add focus effects
    this.addPanelFocusEffects(container);
  }

  showShipCargoAsFloating(shipId) {
    const container = this.expandedView;
    if (!container) {
      console.error("#expanded-view-container not found in showShipCargoAsFloating");
      return;
    }

    // Preserve pinned state while clearing content
    const wasPinned = container._isPinned;
    
    // Clear content but preserve state
    if (!wasPinned) {
      this.clearExpandedView();
    }

    // Set up the container
    container.classList.remove("hidden");
    container.classList.add("floating-panel", "glass-panel");
    
    // Restore pinned state if it was pinned
    if (wasPinned) {
      container.classList.add("pinned-panel");
      container._isPinned = true;
    }

    // Get cargo content
    const content = this.fleetComponentManager.shipCargoComponent.render(shipId);
    const ship = this.fleetComponentManager.findShip(shipId);
    const shipName = ship ? `${ship.count || 1}x ${ship.ship_type_name || 'Unknown'}` : 'Unknown Ship';

    container.innerHTML = `
      <div class="panel-header flex items-center justify-between p-3 cursor-move border-b border-space-600/30">
        <div class="flex items-center gap-2">
          <span class="material-icons drag-handle text-space-400">drag_indicator</span>
          <h2 class="text-lg font-semibold text-white">Cargo: ${shipName}</h2>
        </div>
        <div class="flex items-center gap-2">
          <button onclick="window.uiController.togglePinPanel(this.closest('.floating-panel'))" 
                  class="pin-button text-space-400 hover:text-white transition-colors" 
                  title="Pin">
            <span class="material-icons text-sm">push_pin</span>
          </button>
          <button onclick="window.uiController.clearExpandedView()" 
                  class="text-space-400 hover:text-white transition-colors"
                  title="Close">
            <span class="material-icons">close</span>
          </button>
        </div>
      </div>
      <div class="floating-panel-content p-4">
        ${content}
      </div>
    `;

    // Position the panel (only if not pinned)
    if (!container._isPinned) {
      container.style.top = "20px";
      container.style.left = "20px";
      container.style.right = "auto";
    }

    // Set up dragging functionality
    this.setupPanelDragging(container);
    
    // Add focus effects
    this.addPanelFocusEffects(container);
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
    const incomeGeneratingBuildings = buildings.filter(
      (b) => b.credits_per_tick > 0,
    );
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
      console.warn(
        "Building types not available in gameState for building panel.",
      );
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
            const planetTypeName =
              this.getPlanetTypeName(planet.type) || "Unknown";

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

  showToast(message, type = "success", duration = 4000, clickHandler = null) {
    // Remove any existing toast
    const existingToast = document.getElementById("fleet-toast");
    if (existingToast) {
      existingToast.remove();
    }

    // Create toast element
    const toast = document.createElement("div");
    toast.id = "fleet-toast";
    toast.className = `fixed bottom-20 left-1/2 transform -translate-x-1/2 z-50 p-2 rounded shadow-md transition-all duration-200 max-w-xs`;

    // Add cursor pointer if clickable
    if (clickHandler) {
      toast.className += " cursor-pointer hover:opacity-80";
    }

    if (type === "success") {
      toast.className +=
        " bg-emerald-900/90 border-emerald-600 text-emerald-200";
    } else if (type === "error") {
      toast.className += " bg-red-900/90 border-red-600 text-red-200";
    } else if (type === "info") {
      toast.className += " bg-blue-900/90 border-blue-600 text-blue-200";
    } else if (type === "ticket") {
      toast.className += " bg-slate-900/95 text-slate-200";
      toast.style.maxWidth = "280px";
    }

    // Function to remove toast and clean up
    const removeToast = () => {
      if (toast.parentElement) {
        toast.remove();
      }
      document.removeEventListener("keydown", handleKeyPress);
    };

    if (type === "ticket") {
      toast.innerHTML = `
        <div class="flex items-start justify-between">
          <div class="flex-1">${message}</div>
          <button onclick="event.stopPropagation(); this.parentElement.parentElement.remove()" class="ml-2 text-current opacity-50 hover:opacity-100">
            <span class="material-icons text-xs">close</span>
          </button>
        </div>
      `;
    } else {
      const clickHint = clickHandler ? '<div class="text-xs mt-1 opacity-70">Click for details • Press Space or Esc to dismiss</div>' : '<div class="text-xs mt-1 opacity-70">Press Space or Esc to dismiss</div>';
      
      toast.innerHTML = `
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <span class="material-icons text-sm">${type === "success" ? "check_circle" : type === "error" ? "error" : "info"}</span>
            <span class="text-sm">${message}</span>
          </div>
          <button onclick="event.stopPropagation(); this.parentElement.parentElement.remove()" class="ml-2 text-current opacity-70 hover:opacity-100">
            <span class="material-icons text-sm">close</span>
          </button>
        </div>
        ${clickHint}
      `;
    }

    // Add click handler to the toast if provided
    if (clickHandler) {
      toast.addEventListener('click', (e) => {
        // Don't trigger on close button clicks
        if (e.target.closest('button')) {
          return;
        }
        clickHandler();
        removeToast();
      });
    }

    // Add to page
    document.body.appendChild(toast);

    // Auto-dismiss after specified duration (unless duration is 0 for manual-only)
    if (duration > 0) {
      setTimeout(() => {
        removeToast();
      }, duration);
    }

    // Set up keyboard dismissal
    const handleKeyPress = (e) => {
      if (e.code === "Space" || e.code === "Escape") {
        e.preventDefault();
        removeToast();
      }
    };

    document.addEventListener("keydown", handleKeyPress);
  }

  showSuccessMessage(message) {
    this.showToast(message, "success");
  }

  showCreditsBreakdown() {
    if (!this.currentUser) {
      this.showError("Please log in to view credit breakdown");
      return;
    }

    // Get all crypto_server buildings for the user
    const buildings = this.gameState?.getPlayerBuildings() || [];
    const cryptoServers = buildings.filter((building) => {
      const buildingTypeName = this.gameState?.buildingTypes?.find(
        (bt) => bt.id === building.type,
      )?.name;
      return buildingTypeName === "crypto_server";
    });

    let totalCredits = this.gameState?.playerResources?.credits || 0;
    let totalProduction = 0;

    // Calculate total production per tick
    cryptoServers.forEach((building) => {
      if (building.credits_per_tick) {
        totalProduction += building.credits_per_tick;
      }
    });

    const buildingsList =
      cryptoServers.length > 0
        ? cryptoServers
            .map((building) => {
              const systemName =
                building.system_name ||
                `System ${building.system_id?.slice(-3)}`;
              const storedCredits = building.stored_credits || "Unknown";
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
            })
            .join("")
        : '<div class="text-space-400 text-center py-4">No crypto servers found</div>';

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

            ${
              cryptoServers.length === 0
                ? `
              <div class="bg-amber-900/20 border border-amber-600/30 p-3 rounded">
                <div class="text-amber-300 text-sm">
                  💡 <strong>Tip:</strong> Build Crypto Servers on your planets to generate credits over time!
                </div>
              </div>
            `
                : ""
            }
          </div>

          <button class="w-full btn btn-secondary mt-4" onclick="document.getElementById('modal-overlay').classList.add('hidden')">
            Close
          </button>
        `,
    );
  }
}
