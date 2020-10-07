package bots

// Text of the messages loaded from .json file

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

type messageTemplates struct {
	// common
	Error			string		`json:"error"`
	UnknownCommand	string		`json:"unknown_command"`
	// start
	Hello			string		`json:"hello"`
	HelloAgain		string		`json:"hello_again"`
	// sync
	AlreadySynced	string		`json:"already_synced"`
	EmitKeyTG		string		`json:"emit_key_tg"`
	EmitKeyVK		string		`json:"emit_key_vk"`
	SendToVK		string		`json:"send_to_vk"`
	SendToTG		string		`json:"send_to_tg"`
	UnknownKey		string		`json:"unknown_key"`
}

type locale struct {
	Repr			string		`json:"repr"`
	Messages		*messageTemplates	`json:"messages"`
}

const messagesFile = "messages.json"
func loadMessages()(map[string]*locale, error) {
	file, err := os.Open(messagesFile)
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var locales map[string]*locale
	err = json.Unmarshal(data, &locales)
	if err != nil {
		return nil, err
	}
	return locales, nil
}
