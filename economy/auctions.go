package economy

import (
	"fmt"
	"sync"
)

// Auction is a timed competitive bid for a rare item.
// Items include: mega-structure blueprints, rare ship designs,
// system claims, resource caches, or tech boosts.
type Auction struct {
	ID          int
	Item        string // what's being sold
	Description string
	MinBid      int    // starting price
	CurrentBid  int    // highest bid
	Bidder      string // current highest bidder
	Seller      string // who listed it (empty = galaxy-generated)
	TicksLeft   int    // ticks until auction ends
	Completed   bool
}

// AuctionHouse manages galaxy-wide auctions.
type AuctionHouse struct {
	mu       sync.RWMutex
	auctions []*Auction
	nextID   int
}

// NewAuctionHouse creates an empty auction house.
func NewAuctionHouse() *AuctionHouse {
	return &AuctionHouse{
		auctions: make([]*Auction, 0),
		nextID:   1,
	}
}

// CreateAuction lists a new item for auction.
func (ah *AuctionHouse) CreateAuction(item, description, seller string, minBid, duration int) *Auction {
	ah.mu.Lock()
	defer ah.mu.Unlock()

	a := &Auction{
		ID:          ah.nextID,
		Item:        item,
		Description: description,
		MinBid:      minBid,
		CurrentBid:  0,
		Seller:      seller,
		TicksLeft:   duration,
	}
	ah.nextID++
	ah.auctions = append(ah.auctions, a)

	fmt.Printf("[Auction] #%d: %s (%s) starting at %dcr\n", a.ID, item, description, minBid)
	return a
}

// PlaceBid places a bid on an auction. Returns true if bid accepted.
func (ah *AuctionHouse) PlaceBid(auctionID int, bidder string, amount int) bool {
	ah.mu.Lock()
	defer ah.mu.Unlock()

	for _, a := range ah.auctions {
		if a.ID == auctionID && !a.Completed && a.TicksLeft > 0 {
			if amount > a.CurrentBid && amount >= a.MinBid && bidder != a.Seller {
				a.CurrentBid = amount
				a.Bidder = bidder
				return true
			}
		}
	}
	return false
}

// TickAuctions decrements timers and returns completed auctions.
func (ah *AuctionHouse) TickAuctions() []*Auction {
	ah.mu.Lock()
	defer ah.mu.Unlock()

	var completed []*Auction
	for _, a := range ah.auctions {
		if a.Completed || a.TicksLeft <= 0 {
			continue
		}
		a.TicksLeft--
		if a.TicksLeft <= 0 {
			a.Completed = true
			completed = append(completed, a)
		}
	}
	return completed
}

// GetActiveAuctions returns all active auctions.
func (ah *AuctionHouse) GetActiveAuctions() []*Auction {
	ah.mu.RLock()
	defer ah.mu.RUnlock()

	var result []*Auction
	for _, a := range ah.auctions {
		if !a.Completed && a.TicksLeft > 0 {
			result = append(result, a)
		}
	}
	return result
}
