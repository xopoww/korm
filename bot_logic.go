package main

import (
	"fmt"
	"regexp"
)

func AddHandlers(bots ...BotHandle) {
	for _, bot := range bots {
		// on start
		bot.CommandHandler("start", func(bot BotHandle, m interface{}){
			logger := bot.Logger()

			_, fromID := bot.GetContents(m)
			templates := bot.getUserLocale(fromID)
			var reply string

			var sender User
			uid, err := bot.checkUser(fromID)
			if err != nil {
				reply = templates.Error
				logger.Errorf("Error checking user (id %d): %s", fromID, err)
			} else if uid == 0 {
				// new user
				sender = bot.GetSender(m)
				uid, err = bot.addUser(&sender)
				if err != nil {
					reply = templates.Error
					logger.Errorf("Error adding user (id %d) to database: %s", fromID, err)
				}
				reply = fmt.Sprintf(templates.Hello, sender.FirstName)
			} else {
				// seen this user
				// TODO: add context or whatever
				switch bot.(type) {
				case *vkBot:
					sender, err = bot.getUser(uid)
				case *tgBot:
					sender = bot.GetSender(m)
				}
				if err != nil {
					reply = templates.Error
					logger.Errorf("Error getting user (id %d) from database: %s", fromID, err)
				} else {
					reply = fmt.Sprintf(templates.HelloAgain, sender.FirstName)
				}
			}

			logger.Infof("/start used by %s %s.", sender.FirstName, sender.LastName)
			err = bot.SendText(fromID, reply)
			if err != nil {
				logger.Errorf("Error sending a message: %s", err)
			}
		})

		// on text/unknown command
		bot.DefaultHandler(func(bot BotHandle, m interface{}){
			logger := bot.Logger()
			text, fromID := bot.GetContents(m)
			templates := bot.getUserLocale(fromID)
			com, _ := parseCommand(text)
			var reply string

			if com == "" {
				reply = randEmoji()
			} else {
				reply = fmt.Sprintf(templates.UnknownCommand, com)
			}

			logger.Infof("Message from user (id %d): %s", fromID, text)
			err := bot.SendText(fromID, reply)
			if err != nil {
				logger.Errorf("Error sending a message: %s", err)
			}
		})
	}
}

// utils

var (
	reCommand = regexp.MustCompile("/[^ ]+")
	reArgument = regexp.MustCompile(" .+")
)
/* Parse the text of the message to extract command and argument.
The supposed format is "/{command} {argument}".
Returns empty string (strings) if something wasn't found.
*/
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
