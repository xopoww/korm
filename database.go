package main

import (
	"database/sql"
	tb "gopkg.in/tucnak/telebot.v2"
)

var (
	db *sql.DB
)

// checks if the user with this id is in the DB
// if vk is true, the id is VK user id
// else it is a Telegram user id
// returns UID is the user exists and 0 if not
func checkUser(id int, vk bool)(int, error) {
	var xID, xNet string
	if vk {
		xID = "vkID"
		xNet = "VK"
	} else {
		xID = "tgID"
		xNet = "TG"
	}

	r, err := db.Query(`SELECT id FROM "Users" WHERE $1 = $2`, xID, id)
	if err != nil {
		return 0, err
	}
	defer r.Close()
	dbLogger.Logf(VERBOSE, "Checked a %s user with id %d", xNet, id)
	if r.Next() {
		var uid int
		err = r.Scan(&uid)
		if err != nil {
			return 0, err
		}
		return uid, nil
	}
	return 0, nil
}

// adds vk user
func addVkUser(user *vkUser)(int, error) {
	_, err := db.Exec(`INSERT INTO "vkUsers" (FirstName, LastName, id) VALUES ($1, $2, $3)`,
		user.FirstName, user.LastName, user.ID)
	if err != nil {
		return 0, err
	}

	ra, err := db.Exec(`INSERT INTO "Users" (vkID) VALUES ($1)`,
		user.ID)
	if err != nil {
		return 0, err
	}
	uid, err := ra.LastInsertId()
	if err != nil {
		return 0, err
	}

	dbLogger.Debugf("Added a new vkUser with id %d", user.ID)
	return int(uid), nil
}

// adds tg user
func addTgUser(user *tb.User)(int, error) {
	_, err := db.Exec(`INSERT INTO "tgUsers" (FirstName, LastName, Username,  id) VALUES ($1, $2, $3, $4)`,
		user.FirstName, user.LastName, user.Username, user.ID)
	if err != nil {
		return 0, err
	}

	ra, err := db.Exec(`INSERT INTO "Users" (tgID) VALUES ($1)`,
		user.ID)
	if err != nil {
		return 0, err
	}
	uid, err := ra.LastInsertId()
	if err != nil {
		return 0, err
	}

	dbLogger.Debugf("Added a new tgUser with id %d", user.ID)
	return int(uid), nil
}

// get a vk user by uid
func getVkUser(uid int)(vkUser, error) {
	var user vkUser
	r, err := db.Query(`SELECT VkUsers.id, FirstName, LastName FROM VkUsers JOIN Users WHERE Users.id = $1`,
		uid)
	if err != nil {
		return user, err
	}
	r.Next()
	err = r.Scan(&user.ID, &user.FirstName, &user.LastName)
	return user, err
}

/*
// checks if the user is already synced
func isSynced(id int, vk bool)(bool, error) {
	toID, fromID := "vkID", "tgID"
	if vk {
		toID, fromID = fromID, toID
	}

	r, err := db.Query(`SELECT $1 FROM Users WHERE $2 = $3`, toID, fromID, id)
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
	r, err := db.Query(`SELECT SyncKey FROM Syncro WHERE id = $1 AND fromVK = $2`, id, vkInt)
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

	_, err := db.Exec(`INSERT INTO "Syncro" (id, fromVK, SyncKey) VALUES ($1, $2, $3)`,
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
		return errors.New(fmt.Sprintf("VK User with id %d does not exist", tgID))
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

// searches for key in Syncro table and
// returns a pair (id, fromVK) if the key exists
// returns (0, false) otherwise
func getIdByKey(key string)(int, bool, error){
	r, err := db.Query(`SELECT id, fromVK FROM Syncro WHERE SyncKey = $1`, key)
	if err != nil {
		return 0, false, err
	}
	if !r.Next() {
		dbLogger.Debug("Didn't find a key in Syncro table")
		return 0, false, nil
	}
	var id, vkInt int
	err = r.Scan(&id, &vkInt)
	if err != nil {
		return 0, false, err
	}
	dbLogger.Debug("Found a key in Syncro table")
	return id, vkInt != 0, nil
}
*/

const (
	dbCreation = `
CREATE TABLE IF NOT EXISTS "VkUsers" (
	id			INTEGER NOT NULL PRIMARY KEY UNIQUE,
	FirstName	TEXT NOT NULL,
	LastName	TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS "TgUsers" (
	id			INTEGER NOT NULL PRIMARY KEY UNIQUE,
	FirstName	TEXT NOT NULL,
	LastName	TEXT,
	Username	TEXT

);

CREATE TABLE IF NOT EXISTS "Users" (
	id			INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT UNIQUE,
	vkID		INTEGER UNIQUE,
	tgID		INTEGER UNIQUE,
	
	FOREIGN KEY("vkID") REFERENCES "VkUsers"("id"),
	FOREIGN KEY("tgID") REFERENCES "TgUsers"("id")
);
`
/*
CREATE TABLE IF NOT EXISTS "Orders" (
	id			INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT UNIQUE,
	UID			INTEGER NOT NULL,
	Dish		INTEGER NOT NULL,
	Date		INTEGER NOT NULL,

	FOREIGN KEY("UID") REFERENCES "Users"("id")
);

CREATE TABLE IF NOT EXISTS "Syncro" (
	id		INTEGER NOT NULL,
	fromVK	INTEGER,
	SyncKey	TEXT NOT NULL UNIQUE
);
`
 */


	dbname = "./korm.db"
)
