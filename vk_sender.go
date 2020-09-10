package main

import (
	"fmt"
	vk "github.com/xopoww/vk_min_api"
)

func vkLogic(bot * vk.Bot) {
	// on Start
	bot.HandleOnCommand("start", func(m * vk.Message){
		Messages := getUserLocale(m.FromID, true)

		uid, err := checkUser(m.FromID, true)
		if err != nil {
			bot.Logger.Errorf("Error checking user: %s", err)
			return
		}
		var (
			user vk.User
			reply string
		)

		if uid != 0 {
			// seen this user
			user, err = getVkUser(uid)
			if err != nil {
				bot.Logger.Errorf("Error getting vk user from DB: %s", err)
				return
			}
			reply = fmt.Sprintf(Messages.HelloAgain, user.FirstName)
		} else {
			// new user
			user, err = bot.GetUserByID(m.FromID)
			if err != nil {
				bot.Logger.Errorf("Error getting vk user: %s", err)
				return
			}
			uid, err = addVkUser(&user)
			if err != nil {
				bot.Logger.Errorf("Error adding vk user to DB: %s", err)
				return
			}
			reply = fmt.Sprintf(Messages.Hello, user.FirstName)
		}
		bot.Logger.Debugf("/start used by %s %s", user.FirstName, user.LastName)
		err = bot.SendMessage(m.FromID, reply)
		if err != nil {
			bot.Logger.Errorf("Error sending a message: %s", err)
		}
	})

	// just text
	bot.HandleOnText(func(m * vk.Message){
		Messages := getUserLocale(m.FromID, true)

		var reply string
		if com := m.Command(); com != "" {
			reply = fmt.Sprintf(Messages.UnknownCommand, com)
		} else {
			reply = randEmoji()
		}
		bot.Logger.Debugf("Message from user (id %d): %s", m.FromID, m.Text)
		bot.SendMessage(m.FromID, reply)
	})
}