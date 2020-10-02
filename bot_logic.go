package main

import (
	"fmt"
	"regexp"

	db "github.com/xopoww/korm/database"
	. "github.com/xopoww/korm/types"
)

func AddHandlers(bots ...BotHandle) {
	for _, bot := range bots {
		// on start
		bot.CommandHandler("start",
			func(bot BotHandle, text string, sender *User, newUser bool, messages *messageTemplates){
				bot.Debugf("/start used by user (id %d)", sender.ID)

				var reply string
				switch {
				case sender.FirstName == "":
					reply = messages.Error
				case newUser:
					reply = fmt.Sprintf(messages.Hello, sender.FirstName)
				default:
					reply = fmt.Sprintf(messages.HelloAgain, sender.FirstName)
				}

				err := bot.SendText(sender.ID, reply, nil)
				if err != nil {
					bot.Errorf("Error sending a message to user (id %d): %s", sender.ID, err)
				}
			})

		// menu
		bot.CommandHandler("menu",
			func(bot BotHandle, text string, sender *User, newUser bool, messages *messageTemplates) {
				bot.Debugf("/menu used by user (id %d)", sender.ID)
				var reply string

				dishes, err := db.GetDishes()
				if err != nil {
					reply = messages.Error
				} else {
					reply = "Наши блюда:"
					for index, dish := range dishes {
						reply += fmt.Sprintf("\n%d. %s", index+1, dish)
					}
				}

				err = bot.SendText(sender.ID, reply, nil)
				if err != nil {
					bot.Errorf("Error sending a message to user (id %d): %s", sender.ID, err)
				}
			})

		bot.CommandHandler("test",
			func(bot BotHandle, text string, sender *User, newUser bool, messages *messageTemplates) {
				if text == "arg" {
					err := bot.SendText(sender.ID, "Got your arg!", nil)
					if err != nil {
						bot.Errorf("Error sending a message to user (id %d): %s", sender.ID, err)
					}
					return
				}
				err := bot.SendText(sender.ID, "Look at my nice keyboard!", &TestKeyboard)
				if err != nil {
					bot.Errorf("Error sending a message to user (id %d): %s", sender.ID, err)
				}
			})

		bot.CallbackHandler(func(d string)bool{ return len(d) == 3 },
			func(bot BotHandle, text string, sender *User, newUser bool, messages *messageTemplates) {
				err := bot.SendText(sender.ID, fmt.Sprintf("You've clicked short button: %s", text), nil)
				if err != nil {
					bot.Errorf("Error sending a message to user (id %d): %s", sender.ID, err)
				}
			})

		bot.CallbackHandler(func(d string)bool{ return d == "b a z" },
			func(bot BotHandle, text string, sender *User, newUser bool, messages *messageTemplates) {
				err := bot.SendText(sender.ID, "B A Z", nil)
				if err != nil {
					bot.Errorf("Error sending a message to user (id %d): %s", sender.ID, err)
				}
			})

		// on text/unknown command
		bot.DefaultHandler(
			func(bot BotHandle, text string, sender *User, newUser bool, messages *messageTemplates) {
				bot.Debugf("Message from user (id %d): %s", sender.ID, text)

				var reply string
				com, _ := parseCommand(text)
				if com == "" {
					reply = randEmoji()
				} else {
					reply = fmt.Sprintf(messages.UnknownCommand, com)
				}


				err := bot.SendText(sender.ID, reply, nil)
				if err != nil {
					bot.Errorf("Error sending a message to user (id %d): %s", sender.ID, err)
				}
			})
	}
}

var TestKeyboard = Keyboard{[][]KeyboardButton{
	{KeyboardButton{Caption: "foo", Data: "foo"}, KeyboardButton{Caption: "bar", Data: "bar"}},
	{KeyboardButton{Caption: "baz", Data: "b a z"}},
}}

// utils
var (
	reCommand = regexp.MustCompile("/[^ ]+")
	reArgument = regexp.MustCompile(" .+")
)

// 	Parse the text of the message to extract command and argument.
// The supposed format is "/{command} {argument}".
// Returns empty string (strings) if something wasn't found.
func parseCommand(text string)(command, argument string) {
	textBytes := []byte(text)
	commandBytes := reCommand.Find(textBytes)
	if len(commandBytes) == 0 {
		return "", ""
	}
	command = string(commandBytes[1:])
	argumentBytes := reArgument.Find(textBytes)
	if len(argumentBytes) == 0 {
		argument = ""
	} else {
		argument = string(argumentBytes[1:])
	}
	return
}
