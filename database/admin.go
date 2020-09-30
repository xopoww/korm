package database

import (
	"bytes"
	"crypto/sha1"
	"database/sql"
	"errors"
	"fmt"
)


// 	Add an admin to the database
func AddAdmin(username, password, name string)error {
	_, err := db.Exec(`INSERT INTO "Admins" (username, passhash, name) VALUES ($1, $2, $3)`,
		username, makeHash(password), name)
	return err
}

func makeHash(pass string)[]byte {
	hasher := sha1.New()
	hasher.Write([]byte(pass))
	return hasher.Sum(nil)
}

func checkHash(pass string, hash []byte)bool {
	return bytes.Equal(makeHash(pass), hash)
}

//	Check whether the credentials are valid
// If the check is successful, but credentials are not valid, returns wrapped ErrBadAdmin.
func CheckAdmin(username, password string)error {
	var trueHash []byte
	err := db.QueryRow(`SELECT passhash FROM Admins WHERE username = $1`,
		username).Scan(&trueHash)
	switch {
	case err == nil:
		break
	case errors.Is(err, sql.ErrNoRows):
		return errBadUsername
	default:
		return err
	}

	if !checkHash(password, trueHash) {
		return errBadPassword
	}
	return nil
}
var (
	ErrBadAdmin = errors.New("invalid credentials")
	errBadUsername = fmt.Errorf("%w: bad username", ErrBadAdmin)
	errBadPassword = fmt.Errorf("%w: wrong password", ErrBadAdmin)
)

//  Get admin name by his username
func GetAdminName(username string)(string, error) {
	var name string
	err := db.QueryRow(`SELECT name FROM Admins WHERE username = $1`,
		username).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		err = errBadUsername
	}
	return name, err
}