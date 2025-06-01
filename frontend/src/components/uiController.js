// UI Controller for handling all UI interactions and updates
export class UIController {
  constructor() {
    this.currentUser = null;
    this.gameState = null;
    this.tickTimer = null;
    // Make instance available globally, for event handlers in dynamically created HTML
    window.uiController = this;

    // Listen for system selection events
    document.addEventListener("systemSelected", (event) => {
      if (event.detail && event.detail.system) {
        // Attempt to get planets directly from event if provided by mapRenderer
        let planetsInSystem = event.detail.planets;
        if (!planetsInSystem && this.gameState && this.gameState.mapData && this.gameState.mapData.planets) {
          // Fallback: Filter planets from global game state if not provided in event
          // This assumes system_id in PlanetData matches the full system ID.
          // Pocketbase relation fields often store just the ID.
          // If planet.system_id is an array of IDs, adjust accordingly.
          planetsInSystem = this.gameState.mapData.planets.filter(p => {
            if (Array.isArray(p.system_id)) {
              return p.system_id.includes(event.detail.system.id);
            }
            return p.system_id === event.detail.system.id;
          });
        } else if (!planetsInSystem) {
          planetsInSystem = []; // Ensure it's an array
        }
        this.displaySystemView(event.detail.system, planetsInSystem);
      } else {
        console.warn("systemSelected event triggered without system data", event.detail);
        this.clearExpandedView();
      }
    });
  }

  getPlanetTypeIcon(planetTypeName) {
    if (!planetTypeName) return '❓'; // Default for undefined/empty type name
    const typeMap = {
        'terrestrial': '🌍',
        'gas giant': '💨',   // Key updated to match "Gas Giant"
        'ice giant': '❄️',   // Key updated to match "Ice Giant"
        'volcanic': '🌋',
        'ocean world': '🌊', // Key updated to match "Ocean World"
        'arid': '🏜️',
        'barren': '🏜️',
        'tundra': '🏔️',
        'gaia': '🌸',
        'unknown': '❓'     // Explicit fallback type
        // Ensure these keys (converted to lowercase) match the names
        // from your 'planet_types' collection in the backend.
    };
    return typeMap[planetTypeName.toLowerCase()] || typeMap['unknown'];
  }

  clearExpandedView() {
    const container = document.getElementById("expanded-view-container");
    if (container) {
      container.innerHTML = "";
      container.classList.add("hidden");
    }
  }

  displaySystemView(system, planets) {
    const container = document.getElementById("expanded-view-container");
    if (!container) {
      console.error("Expanded view container not found!");
      return;
    }

    let planetsHtml = '<div class="text-sm text-space-400">No planets listed for this system.</div>';
    if (planets && planets.length > 0) {
      planetsHtml = planets.map(planet => {
        const planetName = planet.name || `Planet ${planet.id.slice(-4)}`;
        const planetIcon = this.getPlanetTypeIcon(planet.planet_type || planet.type); // Support both 'planet_type' and 'type'
        // Note: Ensure planet object is serializable or pass only ID if issues arise with complex objects in onclick
        return `
          <li class="mb-2 p-3 bg-space-700 hover:bg-space-600 rounded-md cursor-pointer transition-colors"
              onclick="window.uiController.displayPlanetView(JSON.parse(decodeURIComponent('${encodeURIComponent(JSON.stringify(planet))}')))">
            <span class="text-xl mr-2">${planetIcon}</span>
            <span class="font-semibold">${planetName}</span>
            <div class="text-xs text-space-300">${planet.planet_type || planet.type || 'Unknown Type'} - Size: ${planet.size || 'N/A'}</div>
          </li>
        `;
      }).join("");
    }

    container.innerHTML = `
      <div class="p-2">
        <div class="flex justify-between items-center mb-4">
          <h2 class="text-2xl font-bold text-orange-300">${system.name || `System ${system.id.slice(-4)}`}</h2>
          <button onclick="window.uiController.clearExpandedView()" class="text-gray-400 hover:text-white text-2xl">&times;</button>
        </div>
        <div class="mb-4 text-sm">
          <div>Coordinates: (${system.x}, ${system.y})</div>
          <div>Owner: ${system.owner_name || "Uncolonized"}</div>
          <div>Richness: ${system.richness || 'N/A'}</div>
        </div>
        <h3 class="text-lg font-semibold mb-2 text-nebula-200">Planets:</h3>
        <ul class="space-y-1 overflow-y-auto max-h-96">
          ${planetsHtml}
        </ul>
      </div>
    `;
    container.classList.remove("hidden");
  }

  displayPlanetView(planet) {
    const container = document.getElementById("expanded-view-container");
    if (!container) {
      console.error("Expanded view container not found!");
      return;
    }

    const planetName = planet.name || `Planet ${planet.id.slice(-4)}`;
    const planetIcon = this.getPlanetTypeIcon(planet.planet_type || planet.type);
    const systemName = planet.system_name || (this.gameState && this.gameState.mapData.systems.find(s => s.id === planet.system_id)?.name) || planet.system_id;


    let resourcesHtml = '<div class="text-sm text-space-400">No resource data.</div>';
    if (planet.Credits !== undefined) { // Check one of the new fields
      resourcesHtml = `
        <div class="grid grid-cols-2 gap-2 text-sm">
          <div>💰 Credits: <span class="font-semibold text-yellow-300">${planet.Credits?.toLocaleString() || 0}</span></div>
          <div>🧑‍🚀 Population: <span class="font-semibold text-blue-300">${planet.Pop?.toLocaleString() || 0} / ${planet.MaxPopulation?.toLocaleString() || 'N/A'}</span></div>
          <div>😊 Morale: <span class="font-semibold text-green-300">${planet.Morale || 0}%</span></div>
          <div>🍞 Food: <span class="font-semibold text-lime-300">${planet.Food?.toLocaleString() || 0}</span></div>
          <div>⛏️ Ore: <span class="font-semibold text-gray-300">${planet.Ore?.toLocaleString() || 0}</span></div>
          <div>📦 Goods: <span class="font-semibold text-orange-300">${planet.Goods?.toLocaleString() || 0}</span></div>
          <div>⛽ Fuel: <span class="font-semibold text-purple-300">${planet.Fuel?.toLocaleString() || 0}</span></div>
        </div>
      `;
    }

    let buildingsHtml = '<div class="text-sm text-space-400">No buildings on this planet.</div>';
    if (planet.Buildings && Object.keys(planet.Buildings).length > 0) {
      buildingsHtml = Object.entries(planet.Buildings).map(([buildingName, level]) => {
        // Attempt to get a nicer name from buildingTypes if available
        let displayName = buildingName;
        if (this.gameState && this.gameState.buildingTypes) {
            const buildingType = this.gameState.buildingTypes.find(bt => bt.id === buildingName || bt.name.toLowerCase() === buildingName.toLowerCase());
            if (buildingType) displayName = buildingType.name;
        }
        return `
          <li class="p-2 bg-space-700 rounded-md">
            <span class="font-semibold">${displayName}</span>: Level ${level}
          </li>
        `;
      }).join("");
      buildingsHtml = `<ul class="space-y-1 text-sm">${buildingsHtml}</ul>`;
    }

    const isColonized = planet.colonized_by && planet.colonized_by !== "";
    const canColonize = !isColonized && this.currentUser; // Add other conditions like proximity or resources if needed

    container.innerHTML = `
      <div class="p-2">
        <div class="flex justify-between items-center mb-4">
          <h2 class="text-2xl font-bold text-orange-300">
            <span class="text-3xl mr-2">${planetIcon}</span>${planetName}
          </h2>
          <button onclick="window.uiController.clearExpandedView()" class="text-gray-400 hover:text-white text-2xl">&times;</button>
        </div>
        <div class="mb-3 text-sm">
          <div>Type: <span class="font-semibold">${planet.planet_type || planet.type || 'Unknown Type'}</span></div>
          <div>Size: <span class="font-semibold">${planet.size || 'N/A'}</span></div>
          <div>System: <span class="font-semibold">${systemName}</span></div>
           <div>Colonized by: <span class="font-semibold">${planet.colonized_by_name || (planet.colonized_by ? 'Another Player' : 'Uncolonized')}</span></div>
        </div>

        <div class="mb-3">
          <h3 class="text-md font-semibold mb-1 text-nebula-200">Resources & Stats:</h3>
          ${resourcesHtml}
        </div>

        <div class="mb-3">
          <h3 class="text-md font-semibold mb-1 text-nebula-200">Buildings:</h3>
          ${buildingsHtml}
        </div>

        ${canColonize ? `
        <button class="w-full mt-4 px-4 py-2 bg-emerald-700 hover:bg-emerald-600 rounded text-white font-semibold"
                onclick="window.uiController.colonizePlanetWrapper('${planet.id}')">
          Colonize Planet (Cost: 500 Credits)
        </button>
        ` : ''}

        ${isColonized && planet.colonized_by === this.currentUser?.id ? `
        <button class="w-full mt-4 px-4 py-2 bg-blue-700 hover:bg-blue-600 rounded text-white font-semibold"
                onclick="window.uiController.manageColony('${planet.id}')">
          Manage Colony
        </button>
        ` : ''}

        <button class="w-full mt-2 px-4 py-2 bg-gray-700 hover:bg-gray-600 rounded text-white"
                onclick="window.uiController.goBackToSystemView('${planet.system_id}')">
          Back to System View
        </button>
      </div>
    `;
    container.classList.remove("hidden");
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
    this.colonizePlanet(planet.id); // Calls the existing colonizePlanet method
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
    document.getElementById("food").textContent =
      resources.food.toLocaleString();
    document.getElementById("ore").textContent = resources.ore.toLocaleString();
    document.getElementById("goods").textContent =
      resources.goods.toLocaleString();
    document.getElementById("fuel").textContent =
      resources.fuel.toLocaleString();

    // Show credit income if available
    const incomeElement = document.getElementById("credit-income");
    if (this.gameState?.creditIncome > 0) {
      incomeElement.textContent = `(+${this.gameState.creditIncome}/tick)`;
      incomeElement.style.display = "inline";
    } else {
      incomeElement.style.display = "none";
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

    const playerCountElement = document.getElementById("player-count-display");
    if (playerCountElement) {
      // This counts systems with owners, which is "Active Factions"
      playerCountElement.textContent = `Active Factions: ${state.systems.filter((s) => s.owner_id).length}`;
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
      <button class="w-full px-4 py-2 bg-space-700 hover:bg-space-600 rounded" onclick="document.getElementById('modal-overlay').classList.add('hidden')">
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
          <button type="submit" class="flex-1 px-4 py-2 bg-red-600 hover:bg-red-500 rounded">
            Send Fleet
          </button>
          <button type="button" onclick="document.getElementById('modal-overlay').classList.add('hidden')"
                  class="flex-1 px-4 py-2 bg-space-700 hover:bg-space-600 rounded">
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
          <button type="submit" class="flex-1 px-4 py-2 bg-green-600 hover:bg-green-500 rounded">
            Create Route
          </button>
          <button type="button" onclick="document.getElementById('modal-overlay').classList.add('hidden')"
                  class="flex-1 px-4 py-2 bg-space-700 hover:bg-space-600 rounded">
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
    const banks = buildings.filter((b) => b.type === "bank");
    const totalIncome = banks.reduce(
      (sum, bank) => sum + (bank.credits_per_tick || 1),
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
                ${building.type === "bank" ? `<div class="text-nebula-300">Income: ${building.credits_per_tick || 1} credits/tick</div>` : ""}
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
        banks.length > 0
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
            const planetTypeName = planet.type || "Unknown";

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

  async colonizePlanet(planetId) {
    try {
      const { pb } = await import("../lib/pocketbase.js");

      // Debug auth status
      console.log("Auth token:", pb.authStore.token ? "Present" : "Missing");
      console.log("User logged in:", pb.authStore.isValid);
      console.log("Current user:", pb.authStore.model);

      if (!pb.authStore.isValid) {
        this.showError("Please log in first to colonize planets");
        return;
      }

      const response = await fetch(`${pb.baseUrl}/api/orders/colonize`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: pb.authStore.token,
        },
        body: JSON.stringify({
          planet_id: planetId,
        }),
      });

      const result = await response.json();

      if (response.ok && result.success) {
        this.hideModal();
        this.showSuccessMessage(
          "Planet colonized successfully! Your new colony has been established.",
        );

        // Refresh game state to show the new colony
        const { gameState } = await import("../stores/gameState.js");
        gameState.refreshGameData();
      } else {
        throw new Error(
          result.error || result.message || "Failed to colonize planet",
        );
      }
    } catch (error) {
      console.error("Colonization error:", error);
      this.showError(`Failed to colonize planet: ${error.message}`);
    }
  }

  showSuccessMessage(message) {
    this.showModal(
      "Success",
      `
      <div class="text-emerald-400 mb-4">${message}</div>
      <button class="w-full px-4 py-2 bg-space-700 hover:bg-space-600 rounded" onclick="document.getElementById('modal-overlay').classList.add('hidden')">
        OK
      </button>
    `,
    );
  }
}
