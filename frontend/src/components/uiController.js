// UI Controller for handling all UI interactions and updates
export class UIController {
  constructor() {
    this.currentUser = null;
    this.gameState = null;
    this.tickTimer = null;
  }

  updateAuthUI(user) {
    this.currentUser = user;
    const loginBtn = document.getElementById('login-btn');
    const userInfo = document.getElementById('user-info');
    const username = document.getElementById('username');

    if (user) {
      loginBtn.classList.add('hidden');
      userInfo.classList.remove('hidden');
      username.textContent = user.username;
    } else {
      loginBtn.classList.remove('hidden');
      userInfo.classList.add('hidden');
      username.textContent = '';
    }
  }

  updateGameUI(state) {
    this.gameState = state;
    this.updateResourcesUI(state.playerResources);
    this.updateSystemInfoUI(state.selectedSystem);
    this.updateGameStatusUI(state);
  }

  updateResourcesUI(resources) {
    document.getElementById('credits').textContent = resources.credits.toLocaleString();
    document.getElementById('food').textContent = resources.food.toLocaleString();
    document.getElementById('ore').textContent = resources.ore.toLocaleString();
    document.getElementById('goods').textContent = resources.goods.toLocaleString();
    document.getElementById('fuel').textContent = resources.fuel.toLocaleString();
    
    // Show credit income if available
    const incomeElement = document.getElementById('credit-income');
    if (this.gameState?.creditIncome > 0) {
      incomeElement.textContent = `(+${this.gameState.creditIncome}/tick)`;
      incomeElement.style.display = 'inline';
    } else {
      incomeElement.style.display = 'none';
    }
  }

  updateSystemInfoUI(selectedSystem) {
    const systemInfo = document.getElementById('selected-system');
    
    if (selectedSystem) {
      const isOwned = this.currentUser && selectedSystem.owner_id === this.currentUser.id;
      
      // Initially show basic system info
      systemInfo.innerHTML = `
        <div class="space-y-2">
          <div class="font-semibold text-orange-300">${selectedSystem.name || `System ${selectedSystem.id.slice(-3)}`}</div>
          <div class="text-xs space-y-1">
            <div>Position: ${selectedSystem.x}, ${selectedSystem.y}</div>
            <div>Population: ${selectedSystem.pop || 0}</div>
            <div>Morale: ${selectedSystem.morale || 0}%</div>
            <div>Owner: ${selectedSystem.owner_name || 'Uncolonized'}</div>
          </div>
          
          <div class="text-xs space-y-1 pt-2 border-t border-space-600">
            <div class="font-medium">Planets:</div>
            <div id="planets-loading" class="text-space-400">Loading planets...</div>
          </div>
          
          ${isOwned ? `
            <div class="text-xs space-y-1 pt-2 border-t border-space-600">
              <div class="font-medium">Resources:</div>
              <div>Food: ${selectedSystem.food || 0}</div>
              <div>Ore: ${selectedSystem.ore || 0}</div>
              <div>Goods: ${selectedSystem.goods || 0}</div>
              <div>Fuel: ${selectedSystem.fuel || 0}</div>
            </div>
            <div class="text-xs space-y-1 pt-2 border-t border-space-600">
              <div class="font-medium">Buildings:</div>
              <div>Habitat: Lvl ${selectedSystem.hab_lvl || 0}</div>
              <div>Farm: Lvl ${selectedSystem.farm_lvl || 0}</div>
              <div>Mine: Lvl ${selectedSystem.mine_lvl || 0}</div>
              <div>Factory: Lvl ${selectedSystem.fac_lvl || 0}</div>
              <div>Shipyard: Lvl ${selectedSystem.yard_lvl || 0}</div>
            </div>
          ` : ''}
        </div>
      `;
      
      // Load planets asynchronously
      this.loadSystemPlanets(selectedSystem.id);
    } else {
      systemInfo.innerHTML = 'Click a system to view details';
    }
  }

  async loadSystemPlanets(systemId) {
    try {
      // Use custom API endpoint instead of PocketBase collections
      const response = await fetch(`http://localhost:8090/api/planets?system_id=${systemId}`);
      const data = await response.json();
      const planets = data.items || [];
      
      const planetsContainer = document.getElementById('planets-loading');
      if (!planetsContainer) return; // System changed while loading
      
      if (planets.length === 0) {
        planetsContainer.innerHTML = '<div class="text-space-400">No planets in this system</div>';
        return;
      }
      
      const planetsHtml = planets.map(planet => {
        const isColonized = planet.colonized_by != null && planet.colonized_by !== '';
        const planetTypeName = planet.type || 'Unknown';
        const colonizedByMe = isColonized && this.currentUser && planet.colonized_by === this.currentUser.id;
        
        return `
          <div class="p-2 bg-space-800 rounded mb-1 cursor-pointer hover:bg-space-700" 
               onclick="window.uiController.selectPlanet('${planet.id}')">
            <div class="font-medium text-space-200">${planet.name}</div>
            <div class="text-xs text-space-400">${planetTypeName} • Size ${planet.size}</div>
            <div class="text-xs ${isColonized ? (colonizedByMe ? 'text-emerald-400' : 'text-red-400') : 'text-space-300'}">
              ${isColonized ? (colonizedByMe ? 'Your Colony' : 'Colonized') : 'Uncolonized'}
            </div>
          </div>
        `;
      }).join('');
      
      planetsContainer.innerHTML = planetsHtml;
      
      // Store reference for planet selection
      window.uiController = this;
      
    } catch (error) {
      console.error('Error loading planets:', error);
      const planetsContainer = document.getElementById('planets-loading');
      if (planetsContainer) {
        planetsContainer.innerHTML = '<div class="text-red-400">Failed to load planets</div>';
      }
    }
  }

  selectPlanet(planetId) {
    // For now, just show colonize modal if planet is not colonized
    fetch(`http://localhost:8090/api/planets`)
      .then(response => response.json())
      .then(data => {
        const planet = data.items.find(p => p.id === planetId);
        if (!planet) {
          this.showError('Planet not found');
          return;
        }
        
        if (!planet.colonized_by || planet.colonized_by === '') {
          // Planet is available for colonization
          this.showPlanetColonizeModal(planet);
        } else {
          // Planet is already colonized, show info
          this.showPlanetInfo(planet);
        }
      }).catch(err => {
        console.error('Error fetching planet:', err);
        this.showError('Failed to load planet information');
      });
  }

  showPlanetColonizeModal(planet) {
    if (!this.currentUser) {
      this.showError('Please log in to colonize planets');
      return;
    }

    const planetTypeName = planet.type || 'Unknown';
    
    this.showModal(`Colonize ${planet.name}`, `
      <div class="space-y-4">
        <div class="p-3 bg-space-800 rounded">
          <div class="font-semibold text-emerald-300">${planet.name}</div>
          <div class="text-sm text-space-300">Type: ${planetTypeName}</div>
          <div class="text-sm text-space-300">Size: ${planet.size}</div>
          <div class="text-sm text-emerald-400">Available for colonization</div>
        </div>
        
        <div class="text-sm text-space-300">
          Establishing a colony will:
          <ul class="list-disc list-inside mt-2 space-y-1">
            <li>Create an initial population of 100</li>
            <li>Build a basic command center</li>
            <li>Start resource production</li>
          </ul>
        </div>
        
        <div class="flex space-x-2">
          <button class="flex-1 px-4 py-2 bg-emerald-700 hover:bg-emerald-600 rounded" 
                  onclick="window.uiController.colonizePlanet('${planet.id}')">
            Colonize Planet
          </button>
          <button class="flex-1 px-4 py-2 bg-space-700 hover:bg-space-600 rounded" 
                  onclick="window.uiController.hideModal()">
            Cancel
          </button>
        </div>
      </div>
    `);
  }

  showPlanetInfo(planet) {
    const planetTypeName = planet.type || 'Unknown';
    const isMyColony = this.currentUser && planet.colonized_by === this.currentUser.id;
    
    this.showModal(`${planet.name} Information`, `
      <div class="space-y-4">
        <div class="p-3 bg-space-800 rounded">
          <div class="font-semibold text-orange-300">${planet.name}</div>
          <div class="text-sm text-space-300">Type: ${planetTypeName}</div>
          <div class="text-sm text-space-300">Size: ${planet.size}</div>
          <div class="text-sm ${isMyColony ? 'text-emerald-400' : 'text-red-400'}">
            ${isMyColony ? 'Your Colony' : 'Colonized by another player'}
          </div>
        </div>
        
        ${isMyColony ? `
          <div class="text-sm text-space-300">
            This is one of your colonies. You can manage it through the buildings and resources panels.
          </div>
        ` : `
          <div class="text-sm text-space-300">
            This planet has already been colonized by another player.
          </div>
        `}
        
        <button class="w-full px-4 py-2 bg-space-700 hover:bg-space-600 rounded" 
                onclick="window.uiController.hideModal()">
          Close
        </button>
      </div>
    `);
  }

  updateGameStatusUI(state) {
    const tickElement = document.getElementById('current-turn');
    const prevTick = tickElement.textContent;
    const newTick = `Tick ${state.currentTick}`;
    
    tickElement.textContent = newTick;
    
    // Add flash animation if tick changed
    if (prevTick !== newTick) {
      tickElement.style.animation = 'none';
      tickElement.offsetHeight; // Trigger reflow
      tickElement.style.animation = 'flash 0.5s ease-out';
    }
    
    document.getElementById('player-count').textContent = state.systems.filter(s => s.owner_id).length;
    
    // Update tick rate display
    const tickRate = state.ticksPerMinute || 6;
    const secondsPerTick = Math.round(60 / tickRate);
    document.getElementById('next-tick').textContent = `${tickRate}/min (${secondsPerTick}s)`;
  }

  startTickTimer(nextTickTime) {
    if (this.tickTimer) {
      clearInterval(this.tickTimer);
    }

    const updateTimer = () => {
      const now = new Date();
      const remaining = nextTickTime - now;
      
      if (remaining <= 0) {
        document.getElementById('next-tick').textContent = 'Processing...';
        clearInterval(this.tickTimer);
        return;
      }

      const minutes = Math.floor(remaining / 60000);
      const seconds = Math.floor((remaining % 60000) / 1000);
      document.getElementById('next-tick').textContent = `${minutes}:${seconds.toString().padStart(2, '0')}`;
    };

    updateTimer();
    this.tickTimer = setInterval(updateTimer, 1000);
  }

  showModal(title, content) {
    const modalOverlay = document.getElementById('modal-overlay');
    const modalContent = document.getElementById('modal-content');
    
    modalContent.innerHTML = `
      <div class="flex justify-between items-center mb-4">
        <h2 class="text-xl font-bold">${title}</h2>
        <button id="modal-close" class="text-space-400 hover:text-space-200">&times;</button>
      </div>
      ${content}
    `;
    
    modalOverlay.classList.remove('hidden');
    
    // Set up close button
    document.getElementById('modal-close').addEventListener('click', () => {
      this.hideModal();
    });
  }

  hideModal() {
    document.getElementById('modal-overlay').classList.add('hidden');
  }

  showError(message) {
    this.showModal('Error', `
      <div class="text-red-400 mb-4">${message}</div>
      <button class="w-full px-4 py-2 bg-space-700 hover:bg-space-600 rounded" onclick="document.getElementById('modal-overlay').classList.add('hidden')">
        OK
      </button>
    `);
  }

  showBuildModal(system) {
    const buildings = [
      { type: 'habitat', name: 'Habitat', cost: 100, description: 'Increases population capacity' },
      { type: 'farm', name: 'Farm', cost: 150, description: 'Produces food' },
      { type: 'mine', name: 'Mine', cost: 200, description: 'Produces ore' },
      { type: 'factory', name: 'Factory', cost: 300, description: 'Produces goods' },
      { type: 'shipyard', name: 'Shipyard', cost: 500, description: 'Enables fleet construction' },
      { type: 'bank', name: 'Bank', cost: 1000, description: 'Generates 1 credit per tick' }
    ];

    const buildingOptions = buildings.map(building => `
      <button class="w-full p-3 bg-space-700 hover:bg-space-600 rounded mb-2 text-left" 
              onclick="window.gameState.queueBuilding('${system.id}', '${building.type}')">
        <div class="font-semibold">${building.name}</div>
        <div class="text-sm text-space-300">${building.description}</div>
        <div class="text-sm text-green-400">Cost: ${building.cost} credits</div>
      </button>
    `).join('');

    this.showModal(`Build in ${system.name || `System ${system.id.slice(-3)}`}`, `
      <div class="space-y-2">
        ${buildingOptions}
      </div>
    `);
  }

  showSendFleetModal(system) {
    const ownedSystems = this.gameState?.getOwnedSystems() || [];
    
    if (ownedSystems.length === 0) {
      this.showError('You need to own at least one system to send fleets');
      return;
    }

    const systemOptions = ownedSystems.map(s => 
      `<option value="${s.id}">${s.name || `System ${s.id.slice(-3)}`}</option>`
    ).join('');

    this.showModal('Send Fleet', `
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
    `);

    document.getElementById('fleet-form').addEventListener('submit', async (e) => {
      e.preventDefault();
      try {
        const fromId = document.getElementById('from-system').value;
        const toId = document.getElementById('to-system-id').value;
        const strength = parseInt(document.getElementById('fleet-strength').value);
        
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
      this.showError('You need to own at least one system to create trade routes');
      return;
    }

    const systemOptions = ownedSystems.map(s => 
      `<option value="${s.id}">${s.name || `System ${s.id.slice(-3)}`}</option>`
    ).join('');

    const cargoTypes = ['food', 'ore', 'goods', 'fuel'];
    const cargoOptions = cargoTypes.map(type =>
      `<option value="${type}">${type.charAt(0).toUpperCase() + type.slice(1)}</option>`
    ).join('');

    this.showModal('Create Trade Route', `
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
    `);

    document.getElementById('trade-form').addEventListener('submit', async (e) => {
      e.preventDefault();
      try {
        const fromId = document.getElementById('trade-from-system').value;
        const toId = document.getElementById('trade-to-system-id').value;
        const cargo = document.getElementById('cargo-type').value;
        const capacity = parseInt(document.getElementById('cargo-capacity').value);
        
        await this.gameState.createTradeRoute(fromId, toId, cargo, capacity);
        this.hideModal();
      } catch (error) {
        this.showError(`Failed to create trade route: ${error.message}`);
      }
    });
  }

  showFleetPanel() {
    const fleets = this.gameState?.getPlayerFleets() || [];
    
    const fleetList = fleets.length > 0 ? fleets.map(fleet => `
      <div class="bg-space-700 p-3 rounded mb-2">
        <div class="font-semibold">Fleet ${fleet.id.slice(-3)}</div>
        <div class="text-sm text-space-300">
          <div>From: ${fleet.from_name || fleet.from_id}</div>
          <div>To: ${fleet.to_name || fleet.to_id}</div>
          <div>Strength: ${fleet.strength}</div>
          <div>ETA: ${fleet.eta_tick ? `Tick ${fleet.eta_tick}` : 'Unknown'}</div>
        </div>
      </div>
    `).join('') : '<div class="text-space-400">No fleets in transit</div>';

    this.showModal('Your Fleets', fleetList);
  }

  showTradePanel() {
    const trades = this.gameState?.getPlayerTrades() || [];
    
    const tradeList = trades.length > 0 ? trades.map(trade => `
      <div class="bg-space-700 p-3 rounded mb-2">
        <div class="font-semibold">Trade Route ${trade.id.slice(-3)}</div>
        <div class="text-sm text-space-300">
          <div>From: ${trade.from_name || trade.from_id}</div>
          <div>To: ${trade.to_name || trade.to_id}</div>
          <div>Cargo: ${trade.cargo}</div>
          <div>Capacity: ${trade.cap}</div>
          <div>ETA: ${trade.eta_tick ? `Tick ${trade.eta_tick}` : 'Unknown'}</div>
        </div>
      </div>
    `).join('') : '<div class="text-space-400">No active trade routes</div>';

    this.showModal('Your Trade Routes', tradeList);
  }

  showDiplomacyPanel() {
    this.showModal('Diplomacy', `
      <div class="text-center text-space-400 py-8">
        Diplomacy features coming soon!
      </div>
    `);
  }

  showBuildingsPanel() {
    const buildings = this.gameState?.getPlayerBuildings() || [];
    const banks = buildings.filter(b => b.type === 'bank');
    const totalIncome = banks.reduce((sum, bank) => sum + (bank.credits_per_tick || 1), 0);
    
    // Group buildings by type
    const buildingsByType = buildings.reduce((acc, building) => {
      if (!acc[building.type]) acc[building.type] = [];
      acc[building.type].push(building);
      return acc;
    }, {});

    const buildingTypeNames = {
      habitat: 'Habitats',
      farm: 'Farms', 
      mine: 'Mines',
      factory: 'Factories',
      shipyard: 'Shipyards',
      bank: 'Banks'
    };

    const buildingSections = Object.entries(buildingsByType).map(([type, typeBuildings]) => `
      <div class="mb-4">
        <h3 class="text-lg font-semibold text-plasma-300 mb-2">${buildingTypeNames[type] || type} (${typeBuildings.length})</h3>
        <div class="space-y-2">
          ${typeBuildings.map(building => `
            <div class="bg-space-700 p-3 rounded">
              <div class="font-semibold text-nebula-300">${building.name || `${buildingTypeNames[type]?.slice(0, -1) || type}-${building.id.slice(-3)}`}</div>
              <div class="text-sm text-space-300">
                <div>System: ${building.system_name || building.system_id}</div>
                ${building.type === 'bank' ? `<div class="text-nebula-300">Income: ${building.credits_per_tick || 1} credits/tick</div>` : ''}
                <div class="text-xs ${building.active !== false ? 'text-green-400' : 'text-red-400'}">
                  ${building.active !== false ? 'Active' : 'Inactive'}
                </div>
              </div>
            </div>
          `).join('')}
        </div>
      </div>
    `).join('');

    this.showModal('Buildings Overview', `
      ${banks.length > 0 ? `
        <div class="mb-4 p-3 bg-space-800 rounded">
          <div class="text-lg font-semibold text-plasma-300">Credit Income: ${totalIncome} credits/tick</div>
          <div class="text-sm text-space-400">${totalIncome * 6} credits/minute • ${totalIncome * 360} credits/hour</div>
        </div>
      ` : ''}
      
      ${buildingSections || '<div class="text-space-400 text-center py-8">No buildings constructed</div>'}
      
      <div class="mt-4 text-xs text-space-400 border-t border-space-600 pt-2">
        💡 Build structures at your systems to improve production and defense
      </div>
    `);
  }

  showColonizeModal(system) {
    if (!this.currentUser) {
      this.showError('Please log in to colonize planets');
      return;
    }

    // We need to fetch planets in this system
    fetch(`http://localhost:8090/api/planets?system_id=${system.id}`)
      .then(response => response.json())
      .then(data => {
        const planets = data.items || [];
        if (planets.length === 0) {
          this.showError('No planets found in this system');
          return;
        }

        const planetOptions = planets.map(planet => {
          const isColonized = planet.colonized_by != null && planet.colonized_by !== '';
          const planetTypeName = planet.type || 'Unknown';
          
          return `
            <div class="p-3 bg-space-700 rounded mb-2 ${isColonized ? 'opacity-50' : 'hover:bg-space-600 cursor-pointer'}" 
                 ${!isColonized ? `onclick="window.uiController.colonizePlanet('${planet.id}')"` : ''}>
              <div class="font-semibold">${planet.name}</div>
              <div class="text-sm text-space-300">Type: ${planetTypeName}</div>
              <div class="text-sm text-space-300">Size: ${planet.size}</div>
              ${isColonized ? 
                `<div class="text-sm text-red-400">Already colonized</div>` : 
                `<div class="text-sm text-emerald-400">Available for colonization</div>`
              }
            </div>
          `;
        }).join('');

        this.showModal(`Colonize Planet in ${system.name || `System ${system.id.slice(-3)}`}`, `
          <div class="space-y-2">
            <div class="text-sm text-space-300 mb-4">
              Select a planet to establish a new colony:
            </div>
            ${planetOptions}
          </div>
        `);

        // Store reference for colonization
        window.uiController = this;
        
      }).catch(err => {
        console.error('Error fetching planets:', err);
        this.showError('Failed to load planets in this system');
      });
  }

  async colonizePlanet(planetId) {
    try {
      const { pb } = await import('../lib/pocketbase.js');
      
      // Debug auth status
      console.log('Auth token:', pb.authStore.token ? 'Present' : 'Missing');
      console.log('User logged in:', pb.authStore.isValid);
      console.log('Current user:', pb.authStore.model);
      
      if (!pb.authStore.isValid) {
        this.showError('Please log in first to colonize planets');
        return;
      }
      
      const response = await fetch(`${pb.baseUrl}/api/orders/colonize`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': pb.authStore.token
        },
        body: JSON.stringify({
          planet_id: planetId
        })
      });

      const result = await response.json();

      if (response.ok && result.success) {
        this.hideModal();
        this.showSuccessMessage('Planet colonized successfully! Your new colony has been established.');
        
        // Refresh game state to show the new colony
        const { gameState } = await import('../stores/gameState.js');
        gameState.refreshGameData();
      } else {
        throw new Error(result.error || result.message || 'Failed to colonize planet');
      }
    } catch (error) {
      console.error('Colonization error:', error);
      this.showError(`Failed to colonize planet: ${error.message}`);
    }
  }

  showSuccessMessage(message) {
    this.showModal('Success', `
      <div class="text-emerald-400 mb-4">${message}</div>
      <button class="w-full px-4 py-2 bg-space-700 hover:bg-space-600 rounded" onclick="document.getElementById('modal-overlay').classList.add('hidden')">
        OK
      </button>
    `);
  }
}