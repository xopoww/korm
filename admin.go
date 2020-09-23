package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/gorilla/mux"
	"html/template"
	"net/http"
	"path/filepath"
	"strconv"
	"sync"
)

const (
	host = "35.228.234.83"
	loginWsEndpoint = "/auth"
)

func setAdminSubroutes(s *mux.Router){

	// global getters:
	globGetters["header"] = func(*http.Request)(data map[string]interface{}){
		data = make(map[string]interface{})
		// TODO: retrieve actual number of orders
		data["numOrders"] = "∞"
		return
	}

	// login
	loginHandler := &templateHandler{
		filename: "login.html",
		getter: nil,
		globGetters: []string{"header"},
	}
	s.Handle("/login", loginHandler)

	// dish profile
	dishHandler := &templateHandler{
		filename: "dish_profile.html",
		getter: func(r *http.Request)map[string]interface{}{
			id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 0)
			if err != nil {
				aaLogger.Errorf("Error parsing id: %s", err)
				return map[string]interface{}{
					"error": err.Error(),
				}
			}
			dish, err := getDishByID(int(id))
			switch {
			case err == nil:
				break
			default:
				aaLogger.Errorf("Error getting dish: %s", err)
				return map[string]interface{}{
					"error": err.Error(),
				}
			}
			return map[string]interface{}{
				"dish": dish,
			}
		},
		globGetters: []string{"header"},
	}
	s.Handle("/dishes/{id:[0-9]+}", mustAuth(dishHandler))

	// order
	orderHandler := &templateHandler{
		filename: "order.html",
		getter: func(r * http.Request)(data map[string]interface{}){
			data = make(map[string]interface{})

			// list of dishes
			dishes, err := getDishes()
			if err != nil {
				aaLogger.Errorf("Error getting list of dishes: %v", err)
				data["dishes_error"] = err.Error()
			} else {
				data["dishes"] = dishes
			}
			return
		},
		globGetters: []string{"header"},
	}
	s.Handle("/order", mustAuth(orderHandler))

	// home
	homeHandler := &templateHandler{
		filename: "home.html",
		getter: func(r * http.Request)(data map[string]interface{}){
			data = make(map[string]interface{})

			// name of admin
			username, err := r.Cookie("username")
			if err == nil {
				name, err := getAdminName(username.Value)
				if err != nil {
					aaLogger.Errorf("Error getting admin name: %s", err)
				} else {
					data["name"] = name
				}
			}

			// list of dishes
			dishes, err := getDishes()
			if err != nil {
				aaLogger.Errorf("Error getting list of dishes: %v", err)
				data["dishes_error"] = err.Error()
			} else {
				data["dishes"] = dishes
			}
			return
		},
		globGetters: []string{"header"},
	}
	s.Handle("", mustAuth(homeHandler))

	// auth
	//s.HandleFunc("/auth", wrapMethod(methodAuthCheck))

	return
}

/* Check whether the client is authorized
Returns:

	- nil, if the client is authorized;

	- http.ErrNoCookie, if the client is not authorized;

	- other error, if something went wrong
 */
func checkAuthCookie(r *http.Request)error {
	username, err := r.Cookie("username")
	if err != nil {
		return err
	}
	token, err := r.Cookie("auth")
	if err != nil {
		return err
	}
	if !checkAuthToken(token.Value, username.Value) {
		// if username-token pair is invalid, just return ErrNoCookie
		// it will be treated the same way as if the values were missing
		return http.ErrNoCookie
	}
	return nil
}

/*
authHandler wraps another http.Handler inside of it. It checks client's "username" and "auth"
cookies and, if they aren't present / have invalid values, redirects to login page.
 */
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

/* Wrap a handler into authHandler
 */
func mustAuth(next http.Handler)http.Handler {
	return authHandler{next: next, redirect: true}
}

func mustAuthAPI(next http.Handler) http.Handler {
	return authHandler{next: next, redirect: false}
}

/*
templateHandler handles an html template specified by filename.
If a request-dependent data is needed for template execution, getter func must be specified
*/

type tmplGetter func(* http.Request)map[string]interface{}
var globGetters = make(map[string]tmplGetter)

type templateHandler struct {
	filename string
	once sync.Once
	tmpl *template.Template
	getter tmplGetter
	globGetters []string
}
func (h * templateHandler) ServeHTTP(w http.ResponseWriter, r * http.Request) {
	// parse template only once
	h.once.Do(func(){
		var err error
		h.tmpl, err = template.ParseFiles(
			filepath.Join("html_templates", h.filename),
			filepath.Join("html_templates", "elements.html"),
			)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			aaLogger.Errorf("Error parsing template: %s", err)
			return
		}
		h.tmpl.Funcs(template.FuncMap{
			"formatJSON": func(data interface{})string{
				formatted, err := json.MarshalIndent(data, "", "    ")
				if err != nil {
					return "error: " + err.Error()
				}
				return string(formatted)
			},
		})
	})

	// retrieve data for execution via getter (if present)
	var data = make(map[string]interface{})
	if h.getter != nil {
		data = h.getter(r)
	}

	for _, getterName := range h.globGetters {
		if getter, found := globGetters[getterName]; found {
			data[getterName] = getter(r)
		} else {
			aaLogger.Warningf("Unknown global getter: \"%s\"", getterName)
		}
	}

	// execute the template
	var err error
	if h.tmpl != nil {
		err = h.tmpl.Execute(w, data)
	} else {
		err = errors.New("template is nil")
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		aaLogger.Errorf("Error executing a template: %s", err)
	}
}

// ======== auth token stuff =========

var authTokenKey = []byte("can be literally anything")
/* Create auth cookie value for user session
 */
func createAuthToken(username string)[]byte {
	hash := sha1.New()
	hash.Write([]byte(username))
	// TODO: figure out a better way to add key
	return hash.Sum(authTokenKey)
}

/* Check whether username-authToken pair is a valid one
*/
func checkAuthToken(token, username string)bool {
	tokenBytes := make([]byte, hex.DecodedLen(len(token)))
	_, err := hex.Decode(tokenBytes, []byte(token))
	if err != nil {
		aaLogger.Warningf("Error decoding auth token (possibly, bad token): %s", err)
		return false
	}
	return bytes.Equal(tokenBytes, createAuthToken(username))
}


func methodAuthCheck(r * http.Request)(map[string]interface{}, error) {
	username := r.Form.Get("username")
	password := r.Form.Get("password")

	err := checkAdmin(username, password)
	if err != nil {
		return respondError(err)
	}

	token := createAuthToken(username)
	tokenHex := make([]byte, hex.EncodedLen(len(token)))
	hex.Encode(tokenHex, token)

	return map[string]interface{}{
		"ok": true,
		"token": string(tokenHex),
	}, nil
}