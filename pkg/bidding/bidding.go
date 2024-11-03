package bidding

import (
	"errors"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/gunrgnhsr/Cycloud/pkg/models"
)

var resourceMaxBidMap = make(map[string]*models.BidWithLock)
var mapMutex sync.Mutex

func RegisterP2PConnection(isRenter bool, resourceID string, ws *websocket.Conn) error {
	mapMutex.Lock()
	defer mapMutex.Unlock()
	bid, exists := resourceMaxBidMap[resourceID]
	if exists {
		if isRenter == models.Renter {
			bid.RenterWS = ws
		} else {
			bid.LoanerWS = ws
		}
		return nil
	}
	return errors.New("resource not found")
}

func GetPeerWS(resourceID string, isRenter bool) (*websocket.Conn, error) {
	mapMutex.Lock()
	defer mapMutex.Unlock()
	bid, exists := resourceMaxBidMap[resourceID]
	if exists {
		if isRenter == models.Renter {
			return bid.LoanerWS, nil
		}
		return bid.RenterWS, nil
	}
	return nil, errors.New("resource not found")
}

func GetMaxBidForResource(resourceID string) (models.BidWithID, error) {
	mapMutex.Lock()
	defer mapMutex.Unlock()
	bid, exists := resourceMaxBidMap[resourceID]
	if exists {
		return bid.MaxBid, nil
	}
	return models.BidWithID{}, errors.New("resource not found")
}

func BidForResource(bid *models.BidWithLock) {
	mapMutex.Lock()
	prevBid, exists := resourceMaxBidMap[bid.MaxBid.RID]
	if exists {
		prevBid.MaxBid.Status = "rejected"
		prevBid.MaxBid.Amount = bid.MaxBid.Amount
		prevBid.MaxBid.Duration = bid.MaxBid.Duration
		prevBid.Lock.Unlock()
	}
	resourceMaxBidMap[bid.MaxBid.RID] = bid
	bid.Lock.Lock()
	mapMutex.Unlock()
}

func MakeResourceUnavailable(resourceID string) {
	mapMutex.Lock()
	bid, exists := resourceMaxBidMap[resourceID]
	if exists {
		bid.MaxBid.Status = "rejected"
		bid.Lock.Unlock()
	}
	mapMutex.Unlock()
}

func CheckBidsForResource(resourceID string) (models.BidWithID, error) {
	mapMutex.Lock()
	bid, exists := resourceMaxBidMap[resourceID]
	if !exists {
		defer mapMutex.Unlock()
		return models.BidWithID{}, errors.New("no bids for resource")
	}else {
		bid.MaxBid.Status = "accepted"
		bid.Lock.Unlock()
	}
	mapMutex.Unlock()

	return bid.MaxBid, nil
}
