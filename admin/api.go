package admin

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"strconv"

	db "github.com/xopoww/korm/database"
	. "github.com/xopoww/korm/types"
)

func SetApiRoutes (s *mux.Router) {

	handler := &apiHandler{
		&templateHandler{
			filename: "api_result.html",
			getter: func(r * http.Request)(data map[string]interface{}){
				data = make(map[string]interface{})

				// get method name from mux.Vars
				methodName, found := mux.Vars(r)["method"]
				if !found {
					data["critical"] = "Cannot parse request."
					return
				}
				data["method"] = methodName

				// get the required method
				method, found := Methods[methodName]
				if !found {
					data["error"] = fmt.Sprintf("unknown method: %s", methodName)
					return
				}

				// execute the method
				response, err := method(r)
				if err != nil {
					// report internal error
					data["error"] = fmt.Sprintf("internal error: %v", err)
					return
				}

				// embed the method response
				data["response"] = response
				return
			},
		},
	}

	s.Handle("/auth", apiMethod(authMethod))
	// TODO: fix mustAuth to check for "serve_html" value
	s.Handle("/{method:[a-zA-Z_]+}", mustAuthAPI(handler))
}

// apiHandler wraps templateHandler. When serving a request, it check for URL Query value "serve_html".
// If it equals true, it uses underlying template handler for the request, and sends response as plain JSON otherwise.
type apiHandler struct {
	tmplH		*templateHandler
}

func (h * apiHandler) ServeHTTP(w http.ResponseWriter, r * http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Form.Get("serve_html") == "true" {
		// serve api_result template
		h.tmplH.ServeHTTP(w, r)
		return
	}

	// serve response as plain text
	methodName, found := mux.Vars(r)["method"]
	if !found {
		http.Error(w, "cannot parse request", http.StatusInternalServerError)
	}
	method, found := Methods[methodName]
	if !found {
		http.Error(w, "unknown method: " + methodName, http.StatusNotFound)
		return
	}

	method.ServeHTTP(w, r)
	return
}

// ======== methods ========

// apiMethod is the underlying function type for all API methods.
// It accepts a request (on which the ParseForm has already been called),
// retrieves the variables it needs from URL Query and executes the actions needed.
// If during one of these steps an internal (server fault) error is encountered, it returns nil map and this error.
// If an error is encountered due to client's fault, it returns a nil error and a map of the following structure:
// 		"ok": false,
//		"error": message, explaining the error,
// After a successful execution, it returns a map with field "ok" set to true and (possibly) other fields containing
// the result of the execution.
type apiMethod func(*http.Request)(map[string]interface{}, error)

// apiMethod can act as an independent http.Handler.
// In this case the method response object is sent as JSON.
func (m apiMethod) ServeHTTP(w http.ResponseWriter, r * http.Request) {
	response, err := m(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, err = w.Write(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	return
}

// Convenience function that takes an error and returns formatted apiMethod response and nil error.
// Must be used inside an apiMethod like this:
//		if err := doSomething(userInput); err != nil {
//			// err is client's fault
//			return respondError(err)
//		}
func respondError(err error)(map[string]interface{}, error) {
	return respondErrMsg(err.Error())
}

// Same as respondError, but accepts a text message instead of error.
func respondErrMsg(msg string)(map[string]interface{}, error) {
	return map[string]interface{}{
		"ok": false,
		"error": msg,
	}, nil
}

// Map of all existing API methods
var Methods = map[string]apiMethod{
	// add dish record to the database
	"new_dish": func(r * http.Request)(map[string]interface{}, error) {
		name := r.Form.Get("name")
		if name == "" {
			return respondErrMsg("missing parameter: name")
		}

		description := r.Form.Get("description")

		quantityS := r.Form.Get("quantity")
		if quantityS == "" {
			return respondErrMsg("missing parameter: quantity")
		}
		quantity, err := strconv.ParseInt(quantityS, 10, 0)
		if err != nil {
			return respondError(err)
		}

		kindS := r.Form.Get("kind")
		if kindS == "" {
			return respondErrMsg("missing parameter: kind")
		}
		kind, err := strconv.ParseInt(kindS, 10, 0)
		if err != nil {
			return respondError(err)
		}

		id, err := db.NewDish(name, description, int(quantity), int(kind))
		if err != nil {
			return nil, err
		}

		return map[string]interface{}{
			"ok": true,
			"id": id,
		}, nil
	},

	// register a new order
	"order": func(r * http.Request)(map[string]interface{}, error) {
		itemsJSON := r.Form.Get("items")
		if itemsJSON == "" {
			return respondErrMsg("missing parameter: items")
		}
		var items []OrderItem
		err := json.Unmarshal([]byte(itemsJSON), &items)
		if err != nil {
			return respondError(err)
		}

		err = db.RegisterOrder(&Order{Items: items})
		switch {
		case err == nil:
			return map[string]interface{}{"ok": true}, nil
		case errors.Is(err, db.ErrBadID), errors.Is(err, db.ErrOutOfStock):
			return respondError(err)
		default:
			return nil, err
		}
	},

	// add portions to an existing dish
	"add_dish": func(r * http.Request)(map[string]interface{}, error) {
		idS := r.Form.Get("id")
		if idS == "" {
			return respondErrMsg("missing parameter: id")
		}
		id, err := strconv.ParseInt(idS, 10, 0)
		if err != nil {
			return respondError(err)
		}

		deltaS := r.Form.Get("delta")
		if deltaS == "" {
			return respondErrMsg("missing parameter: delta")
		}
		delta, err := strconv.ParseInt(deltaS, 10, 0)
		if err != nil {
			return respondError(err)
		}

		err = db.AddDish(int(id), int(delta))
		switch {
		case err == nil:
			return map[string]interface{}{
				"ok": true,
			}, nil
		case errors.Is(err, db.ErrBadID):
			return respondError(err)
		default:
			return nil, err
		}
	},

	// delete a dish record
	"del_dish": func(r * http.Request)(map[string]interface{}, error) {
		idS := r.Form.Get("id")
		if idS == "" {
			return respondErrMsg("missing parameter: id")
		}
		id, err := strconv.ParseInt(idS, 10, 0)
		if err != nil {
			return respondError(err)
		}

		err = db.DelDish(int(id))
		switch {
		case err == nil:
			return map[string]interface{}{
				"ok": true,
			}, nil
		case errors.Is(err, db.ErrBadID):
			return respondError(err)
		default:
			return nil, err
		}
	},
}

// Process authentication request form login form
// Separated from other methods because it mustn't go through auth check middleware.
func authMethod(r * http.Request)(map[string]interface{}, error){
	err := r.ParseForm()
	if err != nil {
		return respondError(err)
	}

	username := r.Form.Get("username")
	password := r.Form.Get("password")

	err = db.CheckAdmin(username, password)
	switch {
	case err == nil:
		break
	case errors.Is(err, db.ErrBadAdmin):
		return respondError(err)
	default:
		return nil, err
	}

	token := createAuthToken(username)
	tokenHex := make([]byte, hex.EncodedLen(len(token)))
	hex.Encode(tokenHex, token)

	return map[string]interface{}{
		"ok": true,
		"token": string(tokenHex),
	}, nil
}