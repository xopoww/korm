package main

import (
	"encoding/json"
	"errors"
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
)



func setApiSubroutes (s *mux.Router) {
	s.Use(mustAuthAPI)
	s.HandleFunc("/add_dish", wrapMethod(methodAddDish))
	s.HandleFunc("/order", wrapMethod(methodOrder))
}

func wrapMethod(method func(*http.Request)(map[string]interface{}, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r * http.Request) {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		response, err := method(r)
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
}

func respondError(err error)(map[string]interface{}, error) {
	return map[string]interface{}{
		"ok": false,
		"error": err.Error(),
	}, nil
}

func methodAddDish(r * http.Request)(map[string]interface{}, error) {
	name := r.Form.Get("name")
	if name == "" {
		return respondError(errors.New("missing parameter: name"))
	}
	description := r.Form.Get("description")
	quantityS := r.Form.Get("quantity")
	if quantityS == "" {
		return respondError(errors.New("missing parameter: quantity"))
	}
	quantity, err := strconv.ParseInt(quantityS, 10, 0)
	if err != nil {
		return respondError(err)
	}

	id, err := addDish(name, description, int(quantity))
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"ok": true,
		"id": id,
	}, nil
}

func methodOrder(r * http.Request)(map[string]interface{}, error) {
	uidS := r.Form.Get("uid")
	if uidS == "" {
		return respondError(errors.New("missing parameter: uid"))
	}
	uid, err := strconv.ParseInt(uidS, 10, 0)
	if err != nil {
		return respondError(err)
	}

	dishIDsS := r.Form["dish_ids"]
	items := make([]OrderItem, len(dishIDsS))
	for index, idS := range dishIDsS {
		id, err := strconv.ParseInt(idS, 10, 0)
		if err != nil {
			return respondError(err)
		}
		items[index].DishID = int(id)
	}

	quantitiesS := r.Form["quantities"]
	if len(quantitiesS) != len(dishIDsS) {
		return respondError(errors.New("dish_ids and quantities must have the same len"))
	}
	for index, qS := range quantitiesS {
		q, err := strconv.ParseInt(qS, 10, 0)
		if err != nil {
			return respondError(err)
		}
		items[index].Quantity = int(q)
	}

	err = registerOrder(Order{int(uid), items})
	if err != nil {
		return respondError(err)
	}

	return map[string]interface{}{"ok": true}, nil
}