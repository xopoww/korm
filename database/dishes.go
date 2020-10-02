package database

import (
	"database/sql"
	"errors"
	"fmt"
	. "github.com/xopoww/korm/types"
)

// 	Add a new dish to the database.
// On success, returns an id of the dish inserted.
func NewDish(name, description string, quantity, kind int) (int, error) {
	res, err := db.Exec(`INSERT INTO "Dishes" (name, description, quantity, kind) VALUES ($1, $2, $3, $4)`,
		name, description, quantity, kind)
	if err != nil {
		return 0, fmt.Errorf("insert into dishes: %w", err)
	}
	db.Debugf("Added %d portions of \"%s\" (kind id %d) to database.", quantity, name, kind)
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}
	return int(id), nil
}

// 	Get list of all dishes in the database.
// Calls GetDishKinds and several GetDishesByKind inside.
func GetDishes()([]Dish, error) {
	kinds, err := GetDishKinds()
	if err != nil {
		return nil, fmt.Errorf("get dish kinds: %w", err)
	}
	allDishes := make([]Dish, 0)
	for _, kind := range kinds {
		dishes, err := GetDishesByKind(kind)
		if err != nil {
			return nil, fmt.Errorf("get dishes by kind: %w", err)
		}
		allDishes = append(allDishes, dishes...)
	}
	db.Tracef("Got total of %d dishes.", len(allDishes))
	return allDishes, nil
}

// 	Get list of all dishes with the specific kind
func GetDishesByKind(kind DishKind)([]Dish, error) {
	r, err := db.Queryx(
		`
SELECT id, name, description, quantity FROM Dishes WHERE kind = $1`,
	kind.ID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if e := r.Close(); e != nil {
			db.Errorf("Could not close a query result: %s", err)
		}
	}()

	result := make([]Dish, 0)
	for r.Next() {
		dish := Dish{Kind: &kind}
		err = r.Scan(&dish.ID, &dish.Name, &dish.Description, &dish.Quantity)
		if err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		result = append(result, dish)
	}

	db.Tracef("Got %d dishes for kind %s.", len(result), kind.Repr)

	return result, nil
}

// 	Get a dish by its ID.
func GetDishByID(id int)(*Dish, error){
	d := Dish{ID: id}
	err := db.QueryRowx(
		`
SELECT name, description, quantity, repr, price
FROM Dishes JOIN DishKinds ON Dishes.Kind = DishKinds.id
WHERE Dishes.id = $1`,
		id).StructScan(&d)

	switch {
	case err == nil:
		return &d, nil
	case errors.Is(err, sql.ErrNoRows):
		return nil, ErrBadID
	default:
		return nil, err
	}
}

// 	Subtract delta portions from the dish by its id.
// If tx is not nil, it is used to execute an update.
// Otherwise, default DB handle is used. Returns ErrOutOfStock if delta is bigger
// than there are portions of the dish left.
func SubDish(id, delta int, tx *sql.Tx) error {
	if err := CheckID(id, "Dishes"); err != nil {
		return err
	}

	var handle interface{Exec(string, ...interface{})(sql.Result, error)}
	if tx == nil {
		handle = db
	} else {
		handle = tx
	}
	res, err := handle.Exec(`UPDATE Dishes SET quantity = quantity - $1 WHERE id = $2 AND quantity >= $1`, delta, id)
	if err != nil {
		return fmt.Errorf("update dishes: %w", err)
	}

	nrows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if nrows == 0 {
		return ErrOutOfStock
	}
	return nil
}

// 	Add delta portions of the dish by its id.
// Uses SubDish with negated delta and nil as tx. Might change in the future.
func AddDish(id, delta int) error {
	return SubDish(id, -delta, nil)
}

//	Delete a dish record from the database.
func DelDish(id int) error {
	r, err := db.Exec(`DELETE FROM Dishes WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete from dishes: %w", err)
	}
	numRows, err := r.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if numRows == 0 {
		return ErrBadID
	}
	return nil
}


// ======== Dish kinds ========

// 	Load a list of all available dish kinds from database.
func GetDishKinds() ([]DishKind, error) {
	r, err := db.Queryx(`SELECT * FROM DishKinds`)
	if err != nil {
		return nil, fmt.Errorf("select from dish kinds: %w", err)
	}
	defer func() {
		if e := r.Close(); e != nil {
			db.Errorf("Cannot close a result: %s", err)
		}
	}()

	kinds := make([]DishKind, 0)
	for r.Next() {
		var kind DishKind
		err = r.StructScan(&kind)
		if err != nil {
			return nil, fmt.Errorf("struct scan: %w", err)
		}
		kinds = append(kinds, kind)
	}

	return kinds, nil
}


