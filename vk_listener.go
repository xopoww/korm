package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

type VKConfirmation struct {
	Type		string 	`json:"type"`
	GroupID		int		`json:"group_id"`
}

func vkTestHandler(w http.ResponseWriter, r *http.Request)error {
	if r.Method == "POST" {
		var vc VKConfirmation
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return err
		}
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
	VK_TOKEN := os.Getenv("VK_TOKEN")
	vkBotInstance = &VkBot{VK_TOKEN, VK_API_VERSION}
	fmt.Printf("Initialized a VK bot with the access token %s.\n", VK_TOKEN)

	http.HandleFunc("/vk", wrapHandler(vkHandler))
	fmt.Println(http.ListenAndServe("", nil))
}
