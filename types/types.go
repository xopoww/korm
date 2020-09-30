package types

import "time"

// User contains general information about a user.
// User.ID is an internal uid.
type User struct {
	FirstName	string
	LastName	string
	ID			int
}

// Dish is a full set of information about a dish.
type Dish struct {
	ID				int
	Name			string
	Description		string
	Quantity		int
	Kind			*DishKind
}

// DishKind represents a kind of dish (e.g. soup, drink)
type DishKind struct {
	ID				int
	Repr			string
	Price			int
}

type OrderItem struct {
	DishID		int		`json:"dish_id"`
	Quantity	int		`json:"quantity"`
}

type Order struct {
	UID			int
	Items		[]OrderItem
	// OfferID is an ID of an offer used (0 if it is a regular order)
	OfferID		int
}

//	A single item of an Offer
type OfferItem struct {
	Kind			*DishKind
	Quantity		int
}

//	A special offer object
type Offer struct {
	ID				int
	Description		string
	Price			int
	Expires			time.Time
	Items			[]OfferItem
}