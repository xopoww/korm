package main

import (
	"database/sql"
	"errors"
	"fmt"
	vk "github.com/xopoww/vk_min_api"
	"time"
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
