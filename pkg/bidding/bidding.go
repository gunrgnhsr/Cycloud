package bidding

import (
	"database/sql"
	"log"
	"sync"
	"time"

	"github.com/gunrgnhsr/Cycloud/pkg/models"
	"github.com/gunrgnhsr/Cycloud/pkg/db"
)

func checkBidsForResource(db *sql.DB, resourceID string) {
	bid, err := pkg.GetMaxBidForResource(db, resourceID)
	if err != nil {
		log.Printf("Error querying bids for resource %s: %v", resourceID, err)
		return
	}

	// TODO: implement the connection from the accepted bid to the resource
	log.Printf("Accepted bid for resource %s: %s", resourceID, bid.BidWithID.BID)	
	duration := bid.BidWithID.Duration
	timer := time.NewTimer(time.Duration(duration) * time.Minute)

	go func() {
		<-timer.C
		log.Printf("Timer expired for resource %s with bid %s", resourceID, bid.BidWithID.BID)
		// TODO: implement the logic to handle the expiration of the bid duration
		err := pkg.FinishCompute(db, resourceID, bid)
		if err != nil {
			log.Printf("Error finishing compute for resource %s with bid %s: %v", resourceID, bid.BidWithID.BID, err)
		}
	}()
}

func startBiddingCycle(db *sql.DB, resources []models.ResourceWithID) {
	var wg sync.WaitGroup

	for _, resource := range resources {
		wg.Add(1)
		go func(resourceID string) {
			defer wg.Done()
			for {
				checkBidsForResource(db, resourceID)
				time.Sleep(1 * time.Minute)
			}
		}(resource.RID)
	}

	wg.Wait()
}

func StartBidding(db *sql.DB) {
	// Fetch the list of resources
	resources, err := pkg.GetAllAvailableResourcesForBidding(db)
	if err != nil {
		log.Fatalf("Error querying resources: %v", err)
	}

	// Start the bidding cycle
	startBiddingCycle(db, resources)
}