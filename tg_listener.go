package main

import (
	"io/ioutil"
	"net/http"
)

func tgHandler(w http.ResponseWriter, r *http.Request)error {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	tgLogger.Debugf("Got a %s request.", r.Method)
	tgLogger.Logf(VERBOSE,"	body: %s", string(body))

	w.WriteHeader(200)
	return nil
}
