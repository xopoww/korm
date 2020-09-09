package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

const (
	VK_SECRET = "testing"
)

var vkBotInstance *VkBot

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

// main handler for VK Callback API requests
func vkHandler(w http.ResponseWriter, r *http.Request)error {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	vkLogger.Debugf("Got a %s request.", r.Method)
	vkLogger.Logf(VERBOSE,"\t body: %s", string(body))

	requestChan <- body
	vkLogger.Debug("Sent a request to the channel.")

	_, err = fmt.Fprint(w, "ok")
	return err
}

// request processing
var requestChan = make(chan []byte, REQUEST_CHAN_SIZE)
const (
	REQUEST_PROCESSERS = 5
	REQUEST_CHAN_SIZE = 25
)

func processRequest(bot * VkBot, body []byte)error {

	reqType, err := getRequestType(body)
	if err != nil {
		return err
	}

	switch reqType {
	case "message_new":
		var obj struct{Object vkMessage `json:"object"`}
		err = json.Unmarshal(body, &obj)
		if err != nil {
			return err
		}
		return handleNewMessage(bot, obj.Object)

	case "bad_request":
		vkLogger.Info("Got a request with a wrong secret.")

	default:
		vkLogger.Warningf("Got an unsupported request type: %s\n", reqType)
	}

	return nil
}

// conditional handlers
func onStart(bot *VkBot, msg vkMessage)error {
	uid, err := checkUser(msg.FromID, true)
	if err != nil {
		return err
	}
	var (
		user vkUser
		reply string
	)
	if uid != 0 {
		user, err = getVkUser(uid)
		if err != nil {
			return err
		}
		reply = fmt.Sprintf("Снова здравствуй, %s!", user.FirstName)
	} else {
		user, err = bot.getUser(msg.FromID)
		if err != nil {
			return err
		}
		uid, err = addVkUser(&user)
		if err != nil {
			return err
		}
		reply = fmt.Sprintf("Привет, %s!", user.FirstName)
	}
	vkLogger.Debugf("/start used by %s %s", user.FirstName, user.LastName)
	err = bot.sendMessage(msg.FromID, fmt.Sprintf(reply))
	return err
}

func handleNewMessage(bot *VkBot, msg vkMessage)error {
	if msg.Text[0:1] == "/" {
		command, _/*payload*/ := getCommand(msg.Text)
		switch command {
		case "start":
			return onStart(bot, msg)
		default:
			return bot.sendMessage(msg.FromID, fmt.Sprintf("Неизвестная команда: %s", command))
		}
	}

	err := bot.sendMessage(msg.FromID, randEmoji())
	if err != nil {
		return err
	}
	vkLogger.Infof("Message from user (id: %d): %s", msg.FromID, msg.Text)
	return nil
}

// utils

// parse message text in the format
// "/{COMMAND} {PAYLOAD}"
func getCommand(text string)(string, string) {
	output := strings.SplitN(text[1:], " ", 2)
	if len(output) > 1 {
		return output[0], output[1]
	}
	return output[0], ""
}