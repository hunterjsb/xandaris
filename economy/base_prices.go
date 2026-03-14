package economy

// BasePrices maps resource types to their anchor prices.
// These are the equilibrium prices when supply equals demand.
var BasePrices = map[string]float64{
	"Iron":        75,
	"Water":       100,
	"Oil":         150,
	"Helium-3":    600,
	"Rare Metals": 500,
	"Fuel":        200,
}

// GetBasePrice returns the base price for a resource, defaulting to 100.
func GetBasePrice(resourceType string) float64 {
	if price, ok := BasePrices[resourceType]; ok {
		return price
	}
	return 100
}
