package main

import (
	"database/sql"
	"errors"
	"fmt"
	vk "github.com/xopoww/vk_min_api"
	"time"

	"bytes"
	"crypto/sha1"
)

var (
	db *sql.DB
)

// ======== User management ========

/* Check if the user is in the DB by their in-app ID.

If vk is true, id is supposed to be VK user ID. Else, it is Telegram user id.

Returns uid of the user if the corresponding record exist and 0 if not.
 */
func checkUser(id int, vk bool)(int, error) {
	var xID, xNet string
	if vk {
		xID = "vkID"
		xNet = "VK"
	} else {
		xID = "tgID"
		xNet = "TG"
	}

	r, err := db.Query(fmt.Sprintf(`SELECT id FROM Users WHERE %s = $1`, xID), id)
	if err != nil {
		return 0, err
	}
	defer func(){
		if err := r.Close(); err != nil {
			dbLogger.Errorf("Error closing query result: %s", err)
		}
	}()
	if r.Next() {
		var uid int
		err = r.Scan(&uid)
		if err != nil {
			return 0, err
		}
		dbLogger.Debugf("Checked a %s user with id %d: exists", xNet, id)
		return uid, nil
	} else if err = r.Err(); err != nil {
		return 0, err
	}
	dbLogger.Debugf("Checked a %s user with id %d: doesn't exist", xNet, id)
	return 0, nil
}

/* Add the user to the database
On success returns the uid of the user added, else returns 0 and error.
 */
func addUser(user * User, vk bool)(int, error) {
	var table, idName string
	if vk {
		table = VkUsersTable
		idName = "vkID"
	} else {
		table = TgUsersTable
		idName = "tgID"
	}

	query := fmt.Sprintf(`INSERT INTO "%s" (FirstName, LastName, id) VALUES ($1, $2, $3)`, table)
	_, err := db.Exec(query, user.FirstName, user.LastName, user.ID)
	if err != nil {
		return 0, err
	}

	ra, err := db.Exec(fmt.Sprintf(`INSERT INTO "Users" (%s) VALUES ($1)`, idName), user.ID)
	if err != nil {
		return 0, err
	}
	uid, err := ra.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(uid), nil
}

/* Get a vk user by uid
 */
func getVkUser(uid int)(vk.User, error) {
	var user vk.User
	query := fmt.Sprintf(`SELECT %s.id, FirstName, LastName FROM %s JOIN Users WHERE Users.id = $1`,
		VkUsersTable, VkUsersTable)
	r, err := db.Query(query, uid)
	defer func(){
		if err := r.Close(); err != nil {
			dbLogger.Errorf("Error closing query result: %s", err)
		}
	}()
	if err != nil {
		return user, err
	}
	r.Next()
	err = r.Scan(&user.ID, &user.FirstName, &user.LastName)
	return user, err
}




// ======== Synchronization ========

/* Check if the user is already synced
*/
func isSynced(id int, vk bool)(bool, error) {
	toID, fromID := "vkID", "tgID"
	if vk {
		toID, fromID = fromID, toID
	}

	r, err := db.Query(fmt.Sprintf(`SELECT %s FROM Users WHERE %s = $1`, toID, fromID), id)
	if err != nil {
		return false, err
	}
	if !r.Next() {
		return false, errors.New(fmt.Sprintf("No user with %s %d", fromID, id))
	}
	var otherID int
	err = r.Scan(&otherID)
	if err != nil {
		return false, err
	}
	dbLogger.Debugf("Checked user sync for %s %d", fromID, id)
	return otherID == 0, nil
}

// returns the sync key if it was emitted
// and empty string otherwise
func getSyncKey(id int, vk bool)(string, error) {
	toID, fromID := "vkID", "tgID"
	if vk {
		toID, fromID = fromID, toID
	}
	vkInt := 0
	if vk {
		vkInt = 1
	}
	r, err := db.Query(`SELECT SyncKey FROM Synchro WHERE id = $1 AND fromVK = $2`, id, vkInt)
	if err != nil {
		return "", err
	}
	if !r.Next() {
		dbLogger.Debugf("No key for %s %d yet", fromID, id)
		return "", nil
	}
	var key string
	err = r.Scan(&key)
	if err != nil {
		return "", err
	}
	dbLogger.Debugf("Got a sync key for %s %d", fromID, id)
	return key, nil
}

func setSyncKey(id int, vk bool, key string)error {
	toID, fromID := "vkID", "tgID"
	if vk {
		toID, fromID = fromID, toID
	}
	vkInt := 0
	if vk {
		vkInt = 1
	}

	_, err := db.Exec(`INSERT INTO "Synchro" (id, fromVK, SyncKey) VALUES ($1, $2, $3)`,
		id, vkInt, key)
	if err == nil {
		dbLogger.Debugf("Set a sync key for %s %d", fromID, id)
	}
	return err
}

// merges two records of vkUser and tgUser into one
// leaves the UID associated with the tgUser
func mergeUsers(tgID, vkID int) error {
	// get UIDs
	tgUID, err := checkUser(tgID, false)
	if err != nil {
		return err
	}
	if tgUID == 0 {
		return errors.New(fmt.Sprintf("TG User with id %d does not exist", tgID))
	}
	vkUID, err := checkUser(vkID, true)
	if err != nil {
		return err
	}
	if vkUID == 0 {
		return errors.New(fmt.Sprintf("VK User with id %d does not exist", vkID))
	}

	// replace vk UID with tg UID
	_, err = db.Exec(`UPDATE Orders SET UID = $1 WHERE UID = $2`, tgUID, vkUID)
	if err != nil {
		return err
	}

	// move vkID to tgUID row and drop vkUID row
	_, err = db.Exec(`
UPDATE Users SET vkID = $1 WHERE id = $2;
DELETE FROM Users WHERE id = $3;
`, vkID, tgUID, vkUID)
	if err != nil {
		return err
	}
	dbLogger.Debugf("Merged records: old UID %d, new UID %d", vkUID, tgUID)
	return nil
}

// searches for key in Synchro table and
// returns a pair (id, fromVK) if the key exists
// returns (0, false) otherwise
func getIdByKey(key string)(int, bool, error){
	r, err := db.Query(`SELECT id, fromVK FROM Synchro WHERE SyncKey = $1`, key)
	if err != nil {
		return 0, false, err
	}
	if !r.Next() {
		dbLogger.Debug("Didn't find a key in Synchro table")
		return 0, false, nil
	}
	var id, vkInt int
	err = r.Scan(&id, &vkInt)
	if err != nil {
		return 0, false, err
	}
	dbLogger.Debug("Found a key in Synchro table")
	return id, vkInt != 0, nil
}

const SyncKeyDuration = time.Minute * 5
var keysToErase = make(chan string, 100)
func oldKeysEraser() {
	// goroutine that picks keys from chan
	// and starts child goroutines
	go func() {
		for key := range keysToErase {
			// child goroutine that sleeps
			// and removes the key
			go func(key string){
				time.Sleep(SyncKeyDuration)
				r, err := db.Exec(`DELETE FROM Synchro WHERE SyncKey = $1`, key)
				if err != nil {
					dbLogger.Errorf("Error deleting a sync key: %s", err)
					return
				}
				if nRows, _ := r.RowsAffected(); nRows != 0 {
					dbLogger.Debugf("Deleted a key %s", key)
				} else {
					dbLogger.Debugf("Key already deleted: %s", key)
				}
			}(key)
		}
	}()
	dbLogger.Info("Started oldKeysEraser")
}



// ======== Misc ========

// get the preferred locale for user
func getUserLocale(id int, vk bool)*messageTemplates{
	// TODO: locale selection and DB query here
	id = 0
	vk = false
	locale := locales["RU"].Messages
	return &locale
}

const (
	dbTemplate = "database_creation.sql"
	dbname = "korm.db"

	VkUsersTable = "VkUsers"
	TgUsersTable = "TgUsers"
)


func makeHash(pass string)[]byte {
	hasher := sha1.New()
	hasher.Write([]byte(pass))
	return hasher.Sum(nil)
}

func checkHash(pass string, hash []byte)bool {
	return bytes.Equal(makeHash(pass), hash)
}

func addAdmin(username, password, name string)error {
	_, err := db.Exec(`INSERT INTO "Admins" (username, passhash, name) VALUES ($1, $2, $3)`,
		username, makeHash(password), name)
	return err
}

type wrongUsername struct {
	username string
}
func (err * wrongUsername) Error()string {
	return fmt.Sprintf("wrong username: %s", err.username)
}

type wrongPassword struct {
	username string
}
func (err * wrongPassword) Error()string {
	return fmt.Sprintf("wrong password for %s", err.username)
}

func checkAdmin(username, password string)error {
	r, err := db.Query(`SELECT passhash FROM Admins WHERE username = $1`, username)
	if err != nil {
		return errors.New(fmt.Sprintf("error making a query: %s", err))
	}
	defer r.Close()
	if !r.Next() {
		return &wrongUsername{username}
	}
	var trueHash []byte
	err = r.Scan(&trueHash)
	if err != nil {
		return errors.New(fmt.Sprintf("error scanning query result: %s", err))
	}
	if !checkHash(password, trueHash) {
		return &wrongPassword{username}
	}
	return nil
}

func getAdminName(username string)(string, error) {
	r, err := db.Query(`SELECT name FROM Admins WHERE username = $1`, username)
	if err != nil {
		return "", err
	}
	defer r.Close()
	if !r.Next() {
		return "", errors.New("No such username: " + username)
	}
	var name string
	err = r.Scan(&name)
	return name, err
}


type Dish struct {
	ID				int
	Name			string
	Description		string
	Quantity		int
	KindDesc		string
}

func newDish(name, description string, quantity, kind int)(int, error) {
	ra, err := db.Exec(`INSERT INTO "Dishes" (name, description, quantity, kind) VALUES ($1, $2, $3, $4)`,
		name, description, quantity, kind)
	if err != nil {
		return 0, err
	}
	dbLogger.Infof("Added %d portions of \"%s\" (kind id %d) to database.", quantity, name, kind)
	id, err := ra.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

func getDishes()([]Dish, error) {
	r, err := db.Query(
		`SELECT Dishes.id, name, Dishes.description, quantity, DishKinds.description FROM Dishes JOIN DishKinds`)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	result := make([]Dish, 0)
	for r.Next() {
		var d Dish
		err = r.Scan(&d.ID, &d.Name, &d.Description, &d.Quantity, &d.KindDesc)
		if err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, nil
}

func getDishByID(id int)(Dish, error){
	r, err := db.Query(
		`SELECT name, Dishes.description, quantity, DishKinds.description` +
			`FROM Dishes JOIN DishKinds WHERE Dishes.id = $1`,
		id)
	if err != nil {
		return Dish{}, err
	}
	defer r.Close()
	if !r.Next() {
		return Dish{}, errors.New("no such dish in the database")
	}
	dish := Dish{ID: id}
	err = r.Scan(&dish.Name, &dish.Description, &dish.Quantity, &dish.KindDesc)
	if err != nil {
		return Dish{}, err
	}
	return dish, nil
}

type ErrBadID struct {
	table	string
}

func (e * ErrBadID) Error()string {
	return fmt.Sprintf("no such id in \"%s\"", e.table)
}

func checkID(id int, table string)error {
	query := fmt.Sprintf("SELECT 1 FROM %s WHERE id = $1", table)
	r, err := db.Query(query, id)
	if err != nil {
		return err
	}
	defer r.Close()
	if !r.Next() {
		return &ErrBadID{table}
	}
	return nil
}

type ErrBadArgument struct {
	msg		string
}

func (e * ErrBadArgument) Error() string {
	return e.msg
}

func subDish(id, delta int, tx *sql.Tx)error {
	err := checkID(id, "Dishes")
	if err != nil {
		return err
	}

	var ra sql.Result
	if tx == nil {
		ra, err = db.Exec(`UPDATE Dishes SET quantity = quantity - $1 WHERE id = $2 AND quantity >= $1`, delta, id)
	} else {
		ra, err = tx.Exec(`UPDATE Dishes SET quantity = quantity - $1 WHERE id = $2 AND quantity >= $1`, delta, id)
	}
	if err != nil {
		return err
	}

	nrows, err := ra.RowsAffected()
	if err != nil {
		return err
	}
	if nrows == 0 {
		return &ErrBadArgument{"cannot subtract more than quantity"}
	}
	return nil
}

func delDish(id int) error {
	r, err := db.Exec(`DELETE FROM Dishes WHERE id = $1`, id)
	if err != nil {
		return err
	}
	numRows, err := r.RowsAffected()
	if err != nil {
		return err
	}
	if numRows == 0 {
		return &ErrBadID{table: "Dishes"}
	}
	return nil
}

func addDish(id, delta int) error {
	return subDish(id, -delta, nil)
}


type OrderItem struct {
	DishID		int		`json:"dish_id"`
	Quantity	int		`json:"quantity"`
}

type Order struct {
	UID			int
	Items		[]OrderItem
}

var orderIn = make(chan Order)
var orderOut = make(chan error)

func orderWorker() {
	for order := range orderIn {
		orderOut <- makeOrder(order.UID, order.Items)
	}
}

func registerOrder(order Order) error {
	orderIn <- order
	return <- orderOut
}

func makeOrder(uid int, items []OrderItem) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	ra, err := tx.Exec(`INSERT INTO Orders (UID, Date) VALUES ($1, $2)`, uid, time.Now().Unix())
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	orderID, err := ra.LastInsertId()
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	for _, item := range items {
		err = subDish(item.DishID, item.Quantity, tx)
		if err != nil {
			_ = tx.Rollback()
			return err
		}

		_, err = tx.Exec(`INSERT INTO OrderItems (order_id, dish_id, quantity) VALUES ($1, $2, $3)`,
			orderID, item.DishID, item.Quantity)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}


type DishKind struct {
	ID				int
	Description		string
	Price			int
}

func getDishKinds()([]DishKind, error) {
	r, err := db.Query(`SELECT * FROM DishKinds`)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	kinds := make([]DishKind, 0)
	for r.Next() {
		var kind DishKind
		err = r.Scan(&kind.ID, &kind.Description, &kind.Price)
		if err != nil {
			return nil, err
		}
		kinds = append(kinds, kind)
	}

	return kinds, nil
}