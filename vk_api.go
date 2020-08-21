package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

const (
	VK_SECRET = "testing"
)

var vkBotInstance *VkBot

// utils
func getHeader(r *http.Request, key string)string {
	if values := r.Header["key"]; len(values) > 0 {
		return values[0]
	} else {
		return ""
	}
}

func wrapHandler(handler func(*VkBot, http.ResponseWriter, *http.Request)error)func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := handler(vkBotInstance, w, r); err != nil {
			fmt.Printf("Error: %s\n", err)
		}
	}
}

// struct for getting the type of the Callback API request
type apiRequest struct {
	Type			string					`json:"type"`
	Secret			string					`json:"secret"`
}

func getRequestType(body []byte)(string, error) {
	var ar apiRequest
	err := json.Unmarshal(body, &ar)
	if err != nil {
		return "", err
	}
	if ar.Secret == VK_SECRET {
		return ar.Type, nil
	} else {
		return "bad_request", nil
	}
}

// VK API Types
type vkMessage struct {
	ID				int						`json:"id"`
	Date			int						`json:"date"`

	FromID			int						`json:"from_id"`
	Text			string					`json:"text"`

	Payload			string					`json:"payload"`
	Keyboard		vkKeyboard				`json:"keyboard"`

	//ReplyMessage	vkMessage				`json:"reply_message"` ???
}

type vkKeyboard struct {
	OneTime			bool					`json:"one_time"`
	Buttons			[][]vkKeyboardButton	`json:"buttons"`
	Inline			bool					`json:"inline"`
}

type vkKeyboardButton struct {
	Action			vkKeyboardAction		`json:"action"`
	Color			string					`json:"color"`
}

type vkKeyboardAction struct {
	Type			string					`json:"type"`
	Label			string					`json:"label"`
	Payload			string					`json:"payload"`
}

type vkUser struct {
	ID				int						`json:"id"`
	FirstName		string					`json:"first_name"`
	LastName		string					`json:"last_name"`
}

// VK API wrappers
type vkNewMsgWrapper struct {
	Object			vkMessage				`json:"object"`
}


// main handler for VK Callback API requests
func vkHandler(bot *VkBot, w http.ResponseWriter, r *http.Request)error {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	fmt.Println(string(body))

	_, err = fmt.Fprint(w, "ok")
	if err != nil {
		return err
	}

	reqType, err := getRequestType(body)
	if err != nil {
		return err
	}

	switch reqType {
	case "message_new":
		var obj vkNewMsgWrapper
		err = json.Unmarshal(body, &obj)
		if err != nil {
			return err
		}
		return handleNewMessage(bot, obj.Object)

	case "bad_request":
		fmt.Println("Got a request with a wrong secret.")

	default:
		fmt.Printf("Got an unsupported request type: %s\n", reqType)
	}

	return nil
}

// conditional handlers
func handleNewMessage(bot *VkBot, msg vkMessage)error {
	user, err := bot.getUser(msg.FromID)
	if err != nil {
		return err
	}

	fmt.Printf("Message from %s %s:\n%s\n", user.FirstName, user.LastName, msg.Text)
	return nil
}
