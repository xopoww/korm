package admin

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"net/http"
)

// 	Check whether the client is authorized
// Returns:
//
// - nil, if the client is authorized;
//
// - http.ErrNoCookie, if the client is not authorized;
//
// - other error, if something went wrong
func checkAuthCookie(r *http.Request)error {
	username, err := r.Cookie("username")
	if err != nil {
		return err
	}
	token, err := r.Cookie("auth")
	if err != nil {
		return err
	}

	tokenBytes := make([]byte, hex.DecodedLen(len(token.Value)))
	_, err = hex.Decode(tokenBytes, []byte(token.Value))
	if err != nil {
		return err
	}
	if bytes.Equal(tokenBytes, createAuthToken(username.Value)) {
		return nil
	} else {
		return http.ErrNoCookie
	}
}


// 	Create auth cookie value for user session
func createAuthToken(username string)[]byte {
	hash := sha1.New()
	hash.Write([]byte(username))
	// TODO: figure out a better way to add key
	return hash.Sum(authTokenKey)
}
var authTokenKey = []byte("can be literally anything")


// authHandler wraps another http.Handler inside of it. It checks client's "username" and "auth"
// cookies and, if they aren't present / have invalid values, either redirects to login page
// or returns http.StatusForbidden (depends on redirect field).
type authHandler struct {
	next http.Handler
	redirect bool
}
func (h authHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := checkAuthCookie(r)
	switch err {
	case nil:
		// authenticated
		h.next.ServeHTTP(w, r)
		return
	case http.ErrNoCookie:
		// not authenticated
		if h.redirect {
			w.Header().Set("location", "/admin/login")
			w.WriteHeader(http.StatusTemporaryRedirect)
		} else {
			w.WriteHeader(http.StatusForbidden)
		}
		return
	default:
		// error
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// 	Wrap a handler into authHandler
func mustAuth(next http.Handler)http.Handler {
	return authHandler{next: next, redirect: true}
}

func mustAuthAPI(next http.Handler) http.Handler {
	return authHandler{next: next, redirect: false}
}