package main

import (
	"fmt"
	tb "gopkg.in/tucnak/telebot.v2"
	"time"
)

func tgInit(token string)(*tb.Bot, error) {
	bot, err := tb.NewBot(tb.Settings{
		Token:  token,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		return nil, err
	}
	tgLogger.Info("Initialized a TG bot.")
	tgLogger.Logf(VERBOSE, "\ttoken: %s", token)

	// on start
	bot.Handle("/start", func(m *tb.Message) {
		Messages := getUserLocale(m.Sender.ID, false)

		var reply string
		uid, err := checkUser(m.Sender.ID, false)
		if err != nil {
			tgLogger.Errorf("Error checking a TG user (id %d): %s", m.Sender.ID, err)
			reply = Messages.Error
		} else if uid == 0 {
			// new user
			_, err = addTgUser(m.Sender)
			if err != nil {
				tgLogger.Errorf("Error adding a TG user: %s", err)
			}
			reply = fmt.Sprintf(Messages.Hello, m.Sender.FirstName)
		} else {
			// already seen user
			reply = fmt.Sprintf(Messages.HelloAgain, m.Sender.FirstName)
		}
		_, err = bot.Send(m.Sender, reply)
		if err != nil {
			tgLogger.Errorf("Error while sending reply: %s", err)
		}
		tgLogger.Debugf("/start used by %s %s", m.Sender.FirstName, m.Sender.LastName)
	})

	// synchronize profiles
	bot.Handle("/sync", func(m *tb.Message) {
		Messages := getUserLocale(m.Sender.ID, false)

		// check if the accounts are already synchronized
		already, err := isSynced(m.Sender.ID, false)
		if err != nil {
			tgLogger.Errorf("Error checking if user (id %d) is synced: %s", m.Sender.ID, err)
			bot.Send(m.Sender, Messages.Error)
			return
		}
		if already {
			bot.Send(m.Sender, Messages.AlreadySynced)
			return
		}

		if m.Payload == "" {
			// emit a new key
			genKey := generateKeyString(KEY_LEN)
			err = setSyncKey(m.Sender.ID, false, genKey)
			if err != nil {
				tgLogger.Errorf("Error setting a sync key for user (id %d): %s", m.Sender.ID, err)
				bot.Send(m.Sender, Messages.Error)
				return
			}
			tgLogger.Infof("Emitted a sync key for user (id %d)", m.Sender.ID)
			bot.Send(m.Sender,
				fmt.Sprintf(Messages.EmitKeyTG, genKey), tb.ModeMarkdownV2)
			keysToErase <- genKey
			return
		} else {
			// check the sent key
			id, fromVK, err := getIdByKey(m.Payload)
			if err != nil {
				tgLogger.Errorf("Error checking a key %s: %s", m.Payload, err)
				bot.Send(m.Sender, Messages.Error)
				return
			}
			if id == 0 {
				// unknown key
				bot.Send(m.Sender, Messages.UnknownKey)
				return
			}
			if !fromVK {
				// the key was emitted from TG
				bot.Send(m.Sender,Messages.SendToVK)
				return
			}
			// key is good => merge the accounts
			err = mergeUsers(m.Sender.ID, id)
			if err != nil {
				tgLogger.Errorf("Error merging users (tgID %d; vkID %d): %s", m.Sender.ID, id, err)
				bot.Send(m.Sender, Messages.Error)
			}
			return
		}
	})

	// just text
	bot.Handle(tb.OnText, func(m *tb.Message) {
		Messages := getUserLocale(m.Sender.ID, false)

		if m.Text[0:1] == "/" {
			command, _ := getCommand(m.Text)
			bot.Send(m.Sender, fmt.Sprintf(Messages.UnknownCommand, command))
		} else {
			bot.Send(m.Sender, randEmoji())
		}
		tgLogger.Debugf("Message from %s %s: %s", m.Sender.FirstName, m.Sender.LastName, m.Text)
	})

	return bot, nil
}