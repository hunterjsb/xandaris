package station

import (
	"math/rand"
)

// selectRandomItems selects n random items from a slice without duplicates
func selectRandomItems(items []string, n int) []string {
	if n > len(items) {
		n = len(items)
	}

	// Create a copy to avoid modifying the original
	available := make([]string, len(items))
	copy(available, items)

	result := make([]string, 0, n)
	for i := 0; i < n; i++ {
		// Pick random index from remaining items
		idx := rand.Intn(len(available))
		result = append(result, available[idx])

		// Remove selected item by swapping with last element
		available[idx] = available[len(available)-1]
		available = available[:len(available)-1]
	}

	return result
}
