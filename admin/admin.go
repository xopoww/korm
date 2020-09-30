package admin

import (
	"encoding/json"
	"errors"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	db "github.com/xopoww/korm/database"
)

const (
	host = "35.228.234.83"
	loginWsEndpoint = "/auth"
)

func SetAdminRoutes(s *mux.Router){
	logger := logrus.Logger{
		Out: os.Stdout,
		Formatter: &logrus.TextFormatter{DisableLevelTruncation: true},
		Level: logrus.DebugLevel,
	}

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
				logger.Errorf("Error parsing id: %s", err)
				return map[string]interface{}{
					"error": err.Error(),
				}
			}
			dish, err := db.GetDishByID(int(id))
			switch {
			case err == nil:
				break
			default:
				logger.Errorf("Error getting dish: %s", err)
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
			dishes, err := db.GetDishes()
			if err != nil {
				logger.Errorf("Error getting list of dishes: %v", err)
				data["dishes_error"] = err.Error()
			} else {
				data["dishes"] = dishes
			}
			return
		},
		globGetters: []string{"header"},
	}
	s.Handle("/order", mustAuth(orderHandler))

	// new dish
	newDishHandler := &templateHandler{
		filename: "new_dish.html",
		getter: func(*http.Request)(data map[string]interface{}){
			data = make(map[string]interface{})

			kinds, err := db.GetDishKinds()
			if err != nil {
				data["error"] = err.Error()
				return
			}
			data["kinds"] = kinds
			return
		},
		globGetters: []string{"header"},
	}
	s.Handle("/new_dish", mustAuth(newDishHandler))

	// home
	homeHandler := &templateHandler{
		filename: "home.html",
		getter: func(r * http.Request)(data map[string]interface{}){
			data = make(map[string]interface{})

			// name of admin
			username, err := r.Cookie("username")
			if err == nil {
				name, err := db.GetAdminName(username.Value)
				if err != nil {
					logger.Errorf("Error getting admin name: %s", err)
				} else {
					data["name"] = name
				}
			}

			// list of dishes
			dishes, err := db.GetDishes()
			if err != nil {
				logger.Errorf("Error getting list of dishes: %v", err)
				data["dishes_error"] = err.Error()
			} else {
				data["dishes"] = dishes
			}
			return
		},
		globGetters: []string{"header"},
	}
	s.Handle("", mustAuth(homeHandler))

	return
}

type tmplGetter func(* http.Request)map[string]interface{}
var globGetters = make(map[string]tmplGetter)

// templateHandler handles an html template specified by filename.
// If a request-dependent data is needed for template execution, getter func must be specified
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
		h.tmpl, err = template.New(h.filename).Funcs(template.FuncMap{
			"formatJSON": func(data interface{})string{
				formatted, err := json.MarshalIndent(data, "", "    ")
				if err != nil {
					return "error: " + err.Error()
				}
				return string(formatted)
			},
		}).ParseFiles(
			filepath.Join("html_templates", h.filename),
			filepath.Join("html_templates", "elements.html"),
			)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
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
			//aaLogger.Warningf("Unknown global getter: \"%s\"", getterName)
		}
	}

	// execute the template
	var err error
	if h.tmpl != nil {
		err = h.tmpl.Execute(w, data)
	} else {
		err = errors.New("template is nil")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	if err != nil {
		//aaLogger.Errorf("Error executing a template: %s", err)
	}
}