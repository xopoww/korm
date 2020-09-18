package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
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

	// login
	loginHandler := &templateHandler{
		filename: "login.html",
		getter: func(r * http.Request)map[string]interface{}{
			return map[string]interface{}{
				"host": host,
				"wsEndpoint": loginWsEndpoint,
			}
		},
	}
	s.Handle("/login", loginHandler)

	// login websocket
	s.Handle(loginWsEndpoint, authCheckHandler{})

	// dishes list
	dishesHandler := &templateHandler{
		filename: "dishes.html",
		getter: func(r * http.Request)map[string]interface{}{
			dishes, err := getDishes()
			if err != nil {
				aaLogger.Errorf("Error getting the list of dishes: %s", err)
				return map[string]interface{}{
					"error": err.Error(),
				}
			}
			return map[string]interface{}{
				"dishes": dishes,
			}
		},
	}
	s.Handle("/dishes/all", mustAuth(dishesHandler))

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
	}
	s.Handle("/dishes/{id:[0-9]+}", mustAuth(dishHandler))

	// home
	homeHandler := &templateHandler{
		filename: "home.html",
		getter: func(r * http.Request)map[string]interface{}{
			username, err := r.Cookie("username")
			if err != nil {
				return nil
			}
			name, err := getAdminName(username.Value)
			if err != nil {
				aaLogger.Errorf("Error getting admin name: %s", err)
				return nil
			}
			return map[string]interface{}{
				"name": name,
			}
		},
	}
	s.Handle("", mustAuth(homeHandler))

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
		w.Header().Set("location", "/admin/login")
		w.WriteHeader(http.StatusTemporaryRedirect)
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
	return authHandler{next: next}
}



const websocketBufSize = 1024
var upgrader = &websocket.Upgrader{ReadBufferSize: websocketBufSize, WriteBufferSize: websocketBufSize}
/*
authCheckHandler is a websocket handler that reads user credentials and responds accordingly
 */
type authCheckHandler struct {}
/*
Constants for authResponse status values
 */
const (
	authStatusOk = 0
	authStatusBad = 1
	authStatusErr = 2
)
func (h authCheckHandler) ServeHTTP(w http.ResponseWriter, r * http.Request){
	// create a websocket connection
	socket, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		aaLogger.Errorf("Unable to upgrade request to websocket: %s", err)
		return
	}
	defer socket.Close()

	// read a message from a websocket
	_, data, err := socket.ReadMessage()
	if err != nil {
		aaLogger.Errorf("Error reading from a websocket: %s", err)
	}

	// unmarshal the message into user credentials struct
	var creds struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	err = json.Unmarshal(data, &creds)
	if err != nil {
		aaLogger.Errorf("Unmarshal error: %s", err)
		return
	}

	// check the credentials and populate authResponse object
	err = checkAdmin(creds.Username, creds.Password)
	authResponse := make(map[string]interface{})
	if err != nil {
		// TODO: improve or simplify logic in this switch
		switch err.(type) {
		case *wrongPassword:
			authResponse["status"] = authStatusBad
			authResponse["error"] = err.Error()
		case *wrongUsername:
			authResponse["status"] = authStatusBad
			authResponse["error"] = err.Error()
		default:
			authResponse["status"] = authStatusErr
			authResponse["error"] = err.Error()
		}
	} else {
		authResponse["status"] = authStatusOk
		authResponse["token"] = hex.EncodeToString(createAuthToken(creds.Username))
	}

	// send the response to the websocket
	responseData, err := json.Marshal(authResponse)
	if err != nil {
		aaLogger.Errorf("Marshal error: %s", err)
		return
	}
	err = socket.WriteMessage(websocket.TextMessage, responseData)
	if err != nil {
		aaLogger.Errorf("Error writing to a websocket: %s", err)
		return
	}
}

/*
templateHandler handles an html template specified by filename.
If a request-dependent data is needed for template execution, getter func must be specified
*/
type templateHandler struct {
	filename string
	once sync.Once
	tmpl *template.Template
	getter func(* http.Request)map[string]interface{}
}
func (h * templateHandler) ServeHTTP(w http.ResponseWriter, r * http.Request) {
	// parse template only once
	h.once.Do(func(){
		var err error
		h.tmpl, err = template.ParseFiles(filepath.Join("html_templates", h.filename))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			aaLogger.Errorf("Error parsing template: %s", err)
			return
		}
	})

	// retrieve data for execution via getter (if present)
	var data map[string]interface{}
	if h.getter != nil {
		data = h.getter(r)
	}

	// execute the template
	err := h.tmpl.Execute(w, data)
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