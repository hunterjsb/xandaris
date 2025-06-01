import { UIController } from './uiController';
import { gameState } from '../stores/gameState'; // This will be the mock from jest.setup.js

// Mock specific gameState methods for more granular control if needed within tests
// For example:
// import { gameState as mockGameState } from '../stores/gameState';
// mockGameState.getSystemPlanets.mockReturnValue([...]);

describe('UIController', () => {
  let uiController;
  let mockExpandedViewContainer;

  beforeEach(() => {
    // Setup a basic DOM environment for each test
    document.body.innerHTML = `
      <div id="expanded-view-container"></div>
      <div id="modal-overlay" class="hidden">
        <div id="modal-content"></div>
      </div>
      <span id="credits">0</span>
      <span id="food">0</span>
      <span id="ore">0</span>
      <span id="goods">0</span>
      <span id="fuel">0</span>
      <span id="credit-income" style="display: none;"></span>
      <div id="user-info" class="hidden">
        <span id="username"></span>
      </div>
      <button id="login-btn"></button>
      <div id="game-tick-display">Tick: 0</div>
      <div id="next-tick-display">Next Tick: (10s period)</div>
    `;
    mockExpandedViewContainer = document.getElementById('expanded-view-container');
    uiController = new UIController();
    // Ensure uiController can find its DOM elements
    uiController.expandedView = mockExpandedViewContainer;

    // Reset mocks for gameState if they are manipulated directly in tests
    jest.clearAllMocks();
  });

  describe('Initialization', () => {
    test('should cache #expanded-view-container and initialize it correctly', () => {
      expect(uiController.expandedView).toBeDefined();
      expect(uiController.expandedView.classList.contains('hidden')).toBe(true);
      expect(uiController.expandedView.classList.contains('floating-panel')).toBe(true);
      expect(uiController.expandedView.style.left).toBe('-2000px');
    });
  });

  describe('clearExpandedView', () => {
    test('should hide and move the expanded view off-screen', () => {
      uiController.expandedView.classList.remove('hidden');
      uiController.expandedView.style.left = '100px';

      uiController.clearExpandedView();

      expect(uiController.expandedView.classList.contains('hidden')).toBe(true);
      expect(uiController.expandedView.style.left).toBe('-2000px');
      expect(uiController.currentSystemId).toBeNull();
    });
  });

  describe('positionPanel', () => {
    beforeEach(() => {
      // Set initial size for the panel for testing positioning
      Object.defineProperty(mockExpandedViewContainer, 'offsetWidth', { configurable: true, value: 300 });
      Object.defineProperty(mockExpandedViewContainer, 'offsetHeight', { configurable: true, value: 400 });

      // Mock window dimensions
      global.innerWidth = 1000;
      global.innerHeight = 800;
    });

    test('should position panel to the right and below cursor by default', () => {
      uiController.positionPanel(mockExpandedViewContainer, 100, 100);
      expect(mockExpandedViewContainer.style.left).toBe('115px'); // 100 + 15 margin
      expect(mockExpandedViewContainer.style.top).toBe('115px');  // 100 + 15 margin
    });

    test('should flip to left if too close to right edge', () => {
      uiController.positionPanel(mockExpandedViewContainer, 800, 100); // 800 + 15 + 300 > 1000
      expect(mockExpandedViewContainer.style.left).toBe(`${800 - 300 - 15}px`); // 800 - 300 - 15 = 485px
      expect(mockExpandedViewContainer.style.top).toBe('115px');
    });

    test('should align to bottom edge if too close to bottom', () => {
      uiController.positionPanel(mockExpandedViewContainer, 100, 500); // 500 + 15 + 400 > 800
      expect(mockExpandedViewContainer.style.left).toBe('115px');
      expect(mockExpandedViewContainer.style.top).toBe(`${800 - 400 - 15}px`); // 800 - 400 - 15 = 385px
    });

    test('should ensure panel is not off-screen left or top', () => {
        uiController.positionPanel(mockExpandedViewContainer, 5, 5); // Would go < margin
        expect(mockExpandedViewContainer.style.left).toBe('15px');
        expect(mockExpandedViewContainer.style.top).toBe('15px');
    });
  });

  describe('displaySystemView', () => {
    const mockSystem = { id: 'sys1', name: 'Sol', x: 0, y: 0, richness: 'High' };
    const mockPlanets = [
      { id: 'p1', name: 'Earth', planet_type: 'Terrestrial', colonized_by: 'user123', Pop: 1000, MaxPopulation: 5000, Credits:100, Morale:80 },
      { id: 'p2', name: 'Mars', planet_type: 'Arid', colonized_by: null, Pop: 0, MaxPopulation: 1000 },
    ];

    test('should render system information and planets', () => {
      // Mock current user for ownership checks in updatePlanetList
      uiController.currentUser = { id: 'user123' };
      // Mock player resources for colonization button check
      gameState.playerResources = { credits: 600 };


      uiController.displaySystemView(mockSystem, mockPlanets, 100, 100);

      expect(mockExpandedViewContainer.classList.contains('floating-panel')).toBe(true);
      expect(mockExpandedViewContainer.classList.contains('hidden')).toBe(false);
      expect(mockExpandedViewContainer.querySelector('#system-name').textContent).toBe('Sol');
      expect(mockExpandedViewContainer.querySelector('#system-coords').textContent).toBe('📍 (0, 0)');

      const planetListItems = mockExpandedViewContainer.querySelectorAll('#system-planets-list li');
      expect(planetListItems.length).toBe(2);
      expect(planetListItems[0].textContent).toContain('Earth');
      expect(planetListItems[1].textContent).toContain('Mars');

      // Check for colonization button on Mars (uncolonized, player has credits)
      const marsListItem = Array.from(planetListItems).find(item => item.textContent.includes('Mars'));
      expect(marsListItem.querySelector('button.btn-success').textContent).toContain('Colonize (500 Cr)');
    });

    test('should call positionPanel with coordinates', () => {
      jest.spyOn(uiController, 'positionPanel');
      uiController.displaySystemView(mockSystem, mockPlanets, 200, 250);
      expect(uiController.positionPanel).toHaveBeenCalledWith(mockExpandedViewContainer, 200, 250);
    });
  });

  describe('updatePlanetList - Direct Colonization', () => {
    let ulElement;
    const mockPlanetUncolonized = { id: 'p1', name: 'Test Planet', planet_type: 'Arid', colonized_by: null, Pop: 0, MaxPopulation: 1000 };
    const mockPlanetColonized = { id: 'p2', name: 'My Planet', planet_type: 'Terrestrial', colonized_by: 'user123', Pop: 100, MaxPopulation: 1000, Credits: 50, Morale: 70 };

    beforeEach(() => {
        ulElement = document.createElement('ul');
        uiController.currentUser = { id: 'user123' }; // Simulate logged-in user
    });

    test('should show colonize button if planet uncolonized and player has resources', () => {
        gameState.playerResources = { credits: 500 }; // Enough credits
        uiController.updatePlanetList(ulElement, [mockPlanetUncolonized], 'user123');
        const button = ulElement.querySelector('li[data-planet-id="p1"] button');
        expect(button).not.toBeNull();
        expect(button.classList.contains('btn-success')).toBe(true);
        expect(button.textContent).toContain('Colonize (500 Cr)');
    });

    test('should show disabled colonize button if planet uncolonized and player has insufficient resources', () => {
        gameState.playerResources = { credits: 100 }; // Not enough credits
        uiController.updatePlanetList(ulElement, [mockPlanetUncolonized], 'user123');
        const button = ulElement.querySelector('li[data-planet-id="p1"] button');
        expect(button).not.toBeNull();
        expect(button.classList.contains('btn-disabled')).toBe(true);
        expect(button.disabled).toBe(true);
    });

    test('should not show colonize button if planet is already colonized', () => {
        gameState.playerResources = { credits: 500 };
        uiController.updatePlanetList(ulElement, [mockPlanetColonized], 'user123');
        const button = ulElement.querySelector('li[data-planet-id="p2"] button.btn-success');
        expect(button).toBeNull();
    });

    test('colonize button should call colonizePlanetWrapper on click', () => {
        gameState.playerResources = { credits: 500 };
        jest.spyOn(uiController, 'colonizePlanetWrapper').mockImplementation(() => {}); // Mock the actual colonization call

        uiController.updatePlanetList(ulElement, [mockPlanetUncolonized], 'user123');
        const button = ulElement.querySelector('li[data-planet-id="p1"] button.btn-success');

        button.click(); // Simulate click
        expect(uiController.colonizePlanetWrapper).toHaveBeenCalledWith('p1');

        uiController.colonizePlanetWrapper.mockRestore(); // Clean up spy
    });
  });

  describe('updateResourcesUI', () => {
    test('should update resource displays correctly', () => {
      const resources = { credits: 1234, food: 567, ore: 890, goods: 12, fuel: 345 };
      gameState.creditIncome = 50; // Mock income for the test

      uiController.updateResourcesUI(resources);

      expect(document.getElementById('credits').textContent).toBe('1,234');
      expect(document.getElementById('food').textContent).toBe('567');
      expect(document.getElementById('ore').textContent).toBe('890');
      expect(document.getElementById('goods').textContent).toBe('12');
      expect(document.getElementById('fuel').textContent).toBe('345');
      expect(document.getElementById('credit-income').textContent).toBe('(+50/tick)');
      expect(document.getElementById('credit-income').style.display).toBe('inline');
    });

     test('should hide credit income if it is zero or negative', () => {
      const resources = { credits: 100, food: 100, ore: 100, goods: 100, fuel: 100 };
      gameState.creditIncome = 0;

      uiController.updateResourcesUI(resources);
      expect(document.getElementById('credit-income').style.display).toBe('none');

      gameState.creditIncome = -10;
      uiController.updateResourcesUI(resources);
      expect(document.getElementById('credit-income').style.display).toBe('none');
    });
  });

  describe('Modals', () => {
    test('showModal and hideModal should toggle visibility', () => {
      const modalOverlay = document.getElementById('modal-overlay');
      expect(modalOverlay.classList.contains('hidden')).toBe(true);

      uiController.showModal('Test Modal', '<p>Hello</p>');
      expect(modalOverlay.classList.contains('hidden')).toBe(false);
      expect(document.getElementById('modal-content').textContent).toContain('Test Modal');
      expect(document.getElementById('modal-content').textContent).toContain('Hello');

      // Simulate clicking close button
      const closeButton = document.getElementById('modal-close');
      closeButton.click();
      expect(modalOverlay.classList.contains('hidden')).toBe(true);
    });

    test('showPlanetBuildModal should display building options', () => {
        const mockPlanet = { id: 'p1', name: 'Test Planet' };
        gameState.buildingTypes = [
            { id: 'b1', name: 'Command Center', description: 'Main hub', cost: { credits: 100 } },
            { id: 'b2', name: 'Mine', description: 'Extracts ore', cost: { credits: 50, ore: 20 } },
        ];
        gameState.resourceTypes = [ // Needed for cost string
            { id: 'credits', name: 'Credits' },
            { id: 'ore', name: 'Ore' },
        ];

        uiController.showPlanetBuildModal(mockPlanet);

        const modalContent = document.getElementById('modal-content');
        expect(modalContent.textContent).toContain('Construct on Test Planet');
        const buttons = modalContent.querySelectorAll('button');
        // Includes building option buttons + cancel button
        expect(buttons.length).toBe(gameState.buildingTypes.length + 1);
        expect(buttons[0].textContent).toContain('Command Center');
        expect(buttons[0].textContent).toContain('Cost: 100 Credits');
        expect(buttons[1].textContent).toContain('Mine');
        expect(buttons[1].textContent).toContain('Cost: 50 Credits, 20 Ore');

        // Simulate click on cancel
        buttons[buttons.length -1].click();
        expect(document.getElementById('modal-overlay').classList.contains('hidden')).toBe(true);
    });
  });
});

// Helper to simulate gameState for specific tests if needed
// const setMockGameState = (newState) => {
//   Object.assign(gameState, newState);
// };
