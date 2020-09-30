package database

import (
	"errors"
	"fmt"
)

// Errors
var (
	ErrBadID = errors.New("no such id")
	ErrOutOfStock = errors.New("cannot subtract more portions than there is in stock")
)

// ======== Utils ========

// Check if there is a record with the given id in the table.
// Table must have an "id" column.
func CheckID(id int, table string) error {
	res, err := db.Query(fmt.Sprintf(`SELECT 1 FROM %s WHERE id = $1 LIMIT 1`, table), id)
	if err != nil {
		return err
	}
	if !res.Next() {
		return ErrBadID
	}
	return nil
}