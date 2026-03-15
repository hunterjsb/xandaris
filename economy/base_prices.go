package economy

import "github.com/hunterjsb/xandaris/entities"

// BasePrices maps resource types to their anchor prices.
// These are the equilibrium prices when supply equals demand.
var BasePrices = map[string]float64{
	entities.ResIron:        75,
	entities.ResWater:       100,
	entities.ResOil:         150,
	entities.ResHelium3:     600,
	entities.ResRareMetals:  500,
	entities.ResFuel:        200,
	entities.ResElectronics: 800,
}

// GetBasePrice returns the base price for a resource, defaulting to 100.
func GetBasePrice(resourceType string) float64 {
	if price, ok := BasePrices[resourceType]; ok {
		return price
	}
	return 100
}
