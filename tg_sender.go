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
		var reply string
		uid, err := checkUser(m.Sender.ID, false)
		if err != nil {
			tgLogger.Errorf("Error checking a TG user (id %d): %s", m.Sender.ID, err)
			reply = ReplyOnError
		} else if uid == 0 {
			// new user
			_, err = addTgUser(m.Sender)
			if err != nil {
				tgLogger.Errorf("Error adding a TG user: %s", err)
			}
			reply = fmt.Sprintf("Привет, %s!", m.Sender.FirstName)
		} else {
			// already seen user
			reply = fmt.Sprintf("Снова здравствуй, %s!", m.Sender.FirstName)
		}
		_, err = bot.Send(m.Sender, reply)
		if err != nil {
			tgLogger.Errorf("Error while sending reply: %s", err)
		}
		tgLogger.Debugf("/start used by %s %s", m.Sender.FirstName, m.Sender.LastName)
	})

	/*
	// synchronize profiles
	bot.Handle("/sync", func(m *tb.Message) {
		// check if the accounts are already synchronized
		already, err := isSynced(m.Sender.ID, false)
		if err != nil {
			tgLogger.Errorf("Error checking if user (id %d) is synced: %s", m.Sender.ID, err)
			bot.Send(m.Sender, ReplyOnError)
			return
		}
		if already {
			bot.Send(m.Sender, "Аккаунт уже синхронизован!")
			return
		}

		if m.Payload == "" {
			// emit a new key
			genKey := generateKeyString(KEY_LEN)
			err = setSyncKey(m.Sender.ID, false, genKey)
			if err != nil {
				tgLogger.Errorf("Error setting a sync key for user (id %d): %s", m.Sender.ID, err)
				bot.Send(m.Sender, ReplyOnError)
				return
			}
			tgLogger.Infof("Emitted a sync key for user (id %d)", m.Sender.ID)
			bot.Send(m.Sender,
				fmt.Sprintf("Ключ для синхронизации:\n`%s`\nПришлите этот ключ в течение 24 часов боту вк.",
					genKey), tb.ModeMarkdownV2)
			return
		} else {
			// check the sent key
			id, fromVK, err := getIdByKey(m.Payload)
			if err != nil {
				tgLogger.Errorf("Error checking a key %s: %s", m.Payload, err)
				bot.Send(m.Sender, ReplyOnError)
				return
			}
			if id == 0 {
				// unknown key
				bot.Send(m.Sender, "Неизвестный ключ!")
				return
			}
			if !fromVK {
				// the key was emitted from TG
				bot.Send(m.Sender,"Ключ надо прислать боту ВК!")
				return
			}
			// key is good => merge the accounts
			err = mergeUsers(m.Sender.ID, id)
			if err != nil {
				tgLogger.Errorf("Error merging users (tgID %d; vkID %d): %s", m.Sender.ID, id, err)
				bot.Send(m.Sender, ReplyOnError)
			}
			return
		}
	})
	 */

	return bot, nil
}