package database

import (
	. "../types"
	"fmt"
	"time"
)

// 	Channels for registering an order.
// If a goroutine wants to register an order, it uses RegisterOrder function
// to put an Order object to orderIn chan and waits for the result (error or nil)
// to appear in orderOut.
var (
	orderIn = make(chan *Order)
	orderOut = make(chan error)
)

// orderWorker is a internal function that picks orders from orderIn, executes them synchronously
// and puts the result to orderOut
func orderWorker() {
	for order := range orderIn {
		orderOut <- makeOrder(order.UID, order.Items)
	}
}

// 	Register an order to be processed by orderWorker
// Only this function can be used to make an order from outside the package.
// If another order is being processed at the moment, RegisterOrder will block
// until worker processes the registered order.
func RegisterOrder(order *Order) error {
	orderIn <- order
	return <- orderOut
}

// 	Make an order.
// Subtracts the ordered items from the DB and records an order.
// If (at any point) an error is encountered, it's returned and no changes will be made to the DB.
func makeOrder(uid int, items []OrderItem) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	res, err := tx.Exec(`INSERT INTO Orders (UID, time) VALUES ($1, $2)`, uid, time.Now().Unix())
	if err != nil {
		if e := tx.Rollback(); e != nil {
			db.Errorf("Cannot rollback a transaction: %s", err)
		}
		return fmt.Errorf("insert into orders: %w", err)
	}
	orderID, err := res.LastInsertId()
	if err != nil {
		if e := tx.Rollback(); e != nil {
			db.Errorf("Cannot rollback a transaction: %s", err)
		}
		return fmt.Errorf("last insert id: %s", err)
	}

	for _, item := range items {
		err = SubDish(item.DishID, item.Quantity, tx)
		if err != nil {
			if e := tx.Rollback(); e != nil {
				db.Errorf("Cannot rollback a transaction: %s", err)
			}
			return fmt.Errorf("sub dish (id %d): %w", item.DishID, err)
		}

		_, err = tx.Exec(`INSERT INTO OrderItems (order_id, dish_id, quantity) VALUES ($1, $2, $3)`,
			orderID, item.DishID, item.Quantity)
		if err != nil {
			if e := tx.Rollback(); e != nil {
				db.Errorf("Cannot rollback a transaction: %s", err)
			}
			return fmt.Errorf("insert into order items: %w", err)
		}
	}

	if e := tx.Commit(); e != nil {
		db.Errorf("Cannot commit a transaction: %s", err)
		return e
	}
	db.Infof("An order (id %d) successfully made.", orderID)
	return nil
}