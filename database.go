package main

import (
	"database/sql"
)

var (
	db *sql.DB
)

func checkVkUser(id int)(int, error) {
	r, err := db.Query(`SELECT id FROM "Users" WHERE vkID = $1`, id)
	if err != nil {
		return 0, err
	}
	defer r.Close()
	dbLogger.Logf(VERBOSE, "Checked a user with id %d", id)
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

// adding vk users
func addVkUser(user vkUser)(int, error) {
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

// get a vk user by uid
func getVkUser(uid int)(vkUser, error) {
	var user vkUser
	r, err := db.Query(`SELECT id, FirstName, LastName FROM VkUsers JOIN Users WHERE Users.id = $1`,
		uid)
	if err != nil {
		return user, err
	}
	r.Next()
	err = r.Scan(&user.ID, &user.FirstName, &user.LastName)
	return user, err
}

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
	LastName	TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS "Users" (
	id			INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT UNIQUE,
	vkID		INTEGER UNIQUE,
	tgID		INTEGER UNIQUE,
	
	FOREIGN KEY("vkID") REFERENCES "VkUsers"("id"),
	FOREIGN KEY("tgID") REFERENCES "TgUsers"("id")
);
`

	dbname = "./korm.db"
)
