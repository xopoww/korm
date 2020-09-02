package main

import (
	"fmt"
	tb "gopkg.in/tucnak/telebot.v2"
	"time"
)

func tg_init(token string)(*tb.Bot, error) {
	bot, err := tb.NewBot(tb.Settings{
		Token:  token,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		return nil, err
	}
	tgLogger.Info("Initialized a TG bot.")
	tgLogger.Logf(VERBOSE, "\ttoken: %s", token)

	bot.Handle("/start", func(m *tb.Message){
		bot.Send(m.Sender, fmt.Sprintf("Привет, %s!", m.Sender.FirstName))
	})

	return bot, nil
}