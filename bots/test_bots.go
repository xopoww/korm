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
	err := bot.EditMessage(c.From, c.MessageID, msg, nil)
	if err != nil {
		logger.Errorf("Error editing a message: %s", err)
	}
}

var BarKeys = &Keyboard{
	keys: [][]KeyboardButton{
		{
			KeyboardButton{
				Label:  "Click",
				Answer: "Clack!",
				Action: nil,
			},
			KeyboardButton{
				Label: "Delete",
				Answer: "Deleted!",
				Action: deleteOnCallback,
			},
		},
		{
			KeyboardButton{
				Label: "Update",
				Answer: "Updated!",
				Action: updateOnCallback,
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
	fooCommand = &Command{
		Name: "a foo command",
		Label:  "foo",
		Action: onFoo,
	}

	barCommand = &Command{
		Name:   "a bar command",
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

	return tbot.Start()
}
