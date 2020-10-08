package bots

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"time"

	. "github.com/xopoww/korm/types"
)

func onFoo(bot BotHandle, user * User) {
	logger.Trace("Handling /foo.")
	_, err := bot.SendMessage("Foo is a good command!", user, nil)
	if err != nil {
		logger.Errorf("Error sending message: %s", err)
	}
}

func deleteOnCallback(bot BotHandle, c * CallbackQuery) {
	err := bot.EditMessage(c.From, c.MessageID, "", nil)
	if err != nil {
		logger.Errorf("Error deleting a message: %s", err)
	}
}

func updateOnCallback(bot BotHandle, c * CallbackQuery) {
	msg := time.Now().String()
	err := bot.EditMessage(c.From, c.MessageID, msg, BarKeys)
	if err != nil {
		logger.Errorf("Error editing a message: %s", err)
	}
}

func setOnCallback(bot BotHandle, c * CallbackQuery) {
	err := bot.EditMessage(c.From, c.MessageID, c.Argument, BarKeys)
	if err != nil {
		logger.Errorf("Error editing a message: %s", err)
	}
}

var BarKeys = &Keyboard{
	keys: [][]KeyboardButton{
		{
			KeyboardButton{
				Label:  "Click",
				Action: "click",
			},
			KeyboardButton{
				Label: "Delete",
				Action: "delete",
			},
		},
		{
			KeyboardButton{
				Label: "Update",
				Action: "update",
			},
		},
		{
			KeyboardButton{
				Label: "Set to \"Foo\"",
				Action: "set",
				Argument: "Foo",
			},
			KeyboardButton{
				Label: "Set to \"Bar\"",
				Action: "set",
				Argument: "Bar",
			},
		},
	},
}

func onBar(bot BotHandle, user * User) {
	logger.Trace("Handling /bar.")
	msg := time.Now().String()
	_, err := bot.SendMessage(msg, user, BarKeys)
	if err != nil {
		logger.Errorf("Error sending message: %s", err)
	}
}

var (
	fooCommand = Command{
		Name: "just a message",
		Label:  "foo",
		Action: onFoo,
	}

	barCommand = Command{
		Name:   "multifunctional keyboard",
		Label:  "bar",
		Action: onBar,
	}
)

var logger = &logrus.Logger{
	Out: os.Stdout,
	Formatter: &logrus.TextFormatter{DisableLevelTruncation: true},
	Level: logrus.TraceLevel,
}

func StartTestBots() error {
	tbot, err := NewTgBot(os.Getenv("TG_TOKEN"), logger)
	if err != nil {
		return fmt.Errorf("new tg bot: %w", err)
	}

	err = tbot.RegisterCommands(fooCommand, barCommand)
	if err != nil {
		return fmt.Errorf("register commands: %w", err)
	}

	tbot.AddCallbackHandler("update", "Updated!", updateOnCallback)
	tbot.AddCallbackHandler("delete", "Deleted!", deleteOnCallback)
	tbot.AddCallbackHandler("click", "Clack!", nil)
	tbot.AddCallbackHandler("set", "Set!", setOnCallback)

	return tbot.Start()
}
