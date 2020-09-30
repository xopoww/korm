package database

import (
	"fmt"
	. "github.com/xopoww/korm/types"
	"time"
)

// 	Get the list of items for the offer by its ID
func getOfferItems(id int) ([]OfferItem, error) {
	r, err := db.Queryx("SELECT * FROM OfferItems WHERE offer_id = $1", id)
	if err != nil {
		return nil, fmt.Errorf("select from offer items: %w", err)
	}
	defer func(){
		if e := r.Close(); e != nil {
			db.Errorf("Cannot close a result: %s", err)
		}
	}()

	items := make([]OfferItem, 0)
	for r.Next() {
		var item OfferItem
		err = r.StructScan(&item)
		if err != nil {
			return nil, fmt.Errorf("struct scan: %s", err)
		}
		items = append(items, item)
	}

	return items, nil
}

//  Get the list of all active offers
func GetOffers() ([]Offer, error) {
	r, err := db.Queryx(`SELECT * FROM Offers WHERE expires > $1`, time.Now().Unix())
	if err != nil {
		return nil, fmt.Errorf("select from offers: %w", err)
	}
	defer func(){
		if e := r.Close(); e != nil {
			db.Errorf("Cannot close a result: %s", err)
		}
	}()

	offers := make([]Offer, 0)
	for r.Next() {
		var (
			offer Offer
			unixTime int64
		)
		err = r.Scan(offer.ID, offer.Description, offer.Price, &unixTime)
		if err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		offer.Expires = time.Unix(unixTime, 0)
		items, err := getOfferItems(offer.ID)
		if err != nil {
			return nil, fmt.Errorf("get offer items: %w", err)
		}
		offer.Items = items
		offers = append(offers, offer)
	}

	return offers, nil
}