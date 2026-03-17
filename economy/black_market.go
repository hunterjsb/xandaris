package economy

import (
	"fmt"
	"math/rand"
	"sync"
)

// BlackMarket offers trades at inflated prices but with risk of seizure.
// Sell at 3x normal price, but 15% chance your goods get confiscated.
// Buy at 1.5x normal price with instant delivery (no local restriction).
//
// The black market exists galaxy-wide — no system/planet restriction.
// This is the only way to trade without a Trading Post or cargo ship.
type BlackMarket struct {
	mu           sync.RWMutex
	transactions int
	seizures     int
}

// NewBlackMarket creates a new black market.
func NewBlackMarket() *BlackMarket {
	return &BlackMarket{}
}

// BlackMarketSell attempts to sell on the black market.
// Returns: credits earned (0 if seized), seized bool, error
func (bm *BlackMarket) BlackMarketSell(basePrice float64, quantity int) (int, bool) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	bm.transactions++

	// 15% chance of seizure
	if rand.Intn(100) < 15 {
		bm.seizures++
		fmt.Printf("[BlackMarket] Goods seized! (%d seizures / %d transactions)\n",
			bm.seizures, bm.transactions)
		return 0, true
	}

	// 3x price for successful sale
	credits := int(basePrice * 3.0 * float64(quantity))
	return credits, false
}

// BlackMarketBuy calculates the cost to buy on the black market.
// 1.5x normal price, no location restrictions.
func (bm *BlackMarket) BlackMarketBuy(basePrice float64, quantity int) int {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	bm.transactions++

	return int(basePrice * 1.5 * float64(quantity))
}

// GetStats returns black market activity stats.
func (bm *BlackMarket) GetStats() (transactions, seizures int) {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	return bm.transactions, bm.seizures
}
