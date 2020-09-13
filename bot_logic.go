package main

import "fmt"

func AddHandlers(bots ...Bot) {
	for _, bot := range bots {
		// on start
		bot.CommandHandler("start", func(m interface{}){
			botType := ""
			switch bot.(type) {
			case *tgBot:
				botType = "TG bot"
			case *vkBot:
				botType = "VK bot"
			}
			fmt.Printf("Handling start by %s", botType)
			
			_, fromID := bot.GetContents(m)
			templates := bot.getUserLocale(fromID)
			var reply string

			uid, err := bot.checkUser(fromID)

			if err != nil {
				reply = templates.Error
				// TODO: logging
			} else if uid == 0 {
				// new user
				sender := bot.GetSender(m)
				uid, err = bot.addUser(&sender)
				if err != nil {
					reply = templates.Error
					// TODO: logging
				}
				reply = fmt.Sprintf(templates.Hello, sender.FirstName)
			} else {
				// seen this user
				var sender User
				// TODO: add context or whatever
				switch bot.(type) {
				case *vkBot:
					sender, err = bot.getUser(uid)
				case *tgBot:
					sender = bot.GetSender(m)
				}
				if err != nil {
					reply = templates.Error
					// TODO: logging
				} else {
					reply = fmt.Sprintf(templates.HelloAgain, sender.FirstName)
				}
			}

			// TODO: logging
			_ = bot.SendText(fromID, reply) // TODO: error handling
		})
	}
}
