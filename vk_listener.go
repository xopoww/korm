package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type VKConfirmation struct {
	Type		string 	`json:"type"`
	GroupID		int		`json:"group_id"`
}

func stripBody(old []byte)[]byte {
	new := []byte{}
	for _, b := range old {
		if b != '\\' {
			new = append(new, b)
		}
	}
	return new[1:len(new) - 1]
}

func wrapHandler(handler func(http.ResponseWriter, *http.Request)error)func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := handler(w, r); err != nil {
			fmt.Printf("Error: %s\n", err)
		}
	}
}

func vkTestHandler(w http.ResponseWriter, r *http.Request)error {
	if r.Method == "POST" {
		var vc VKConfirmation
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return err
		}
		body = stripBody(body)
		fmt.Printf("Body: %s\n", string(body))
		err = json.Unmarshal(body, &vc)
		if err != nil {
			return err
		}
		if vc.Type == "confirmation" {
			_, err = fmt.Fprint(w, "df6b734b")
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func main() {
	http.HandleFunc("/vk", wrapHandler(vkTestHandler))
	if err := http.ListenAndServe(":8888", nil); err != nil {
		fmt.Println(err)
	}
}
