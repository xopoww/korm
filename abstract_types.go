package main

import (
	"fmt"
	"github.com/sirupsen/logrus"

	vk "github.com/xopoww/vk_min_api"
	tb "gopkg.in/tucnak/telebot.v2"

	db "github.com/xopoww/korm/database"
	. "github.com/xopoww/korm/types"
)

// Message handler is a function for handling text messages by bot.
// It must be passed as an argument to either DefaultHandler or CommandHandler methods of BotHandle.
// If it is a default handler, text will be a text of the message. If it is a CommandHandler, text will
// be a payload of the specified command.
type messageHandler func(bot BotHandle, text string, sender *User, newUser bool, messages *messageTemplates)

type BotHandle interface {
	SendText(id int, msg string) error

	//  Add on-text handler to the bot
	// Before passing the message to the message handler, it checks if the user is added to the database
	// (and passes the result of the check as a newUser parameter to the messageHandler), and adds him if necessary.
	DefaultHandler(action messageHandler)

	// 	Add on-command handler to the bot
	// Performs the same actions before passing the message to the messageHandler as DefaultHandler
	CommandHandler(command string, action messageHandler)

	Debug(...interface{})
	Info(...interface{})
	Warn(...interface{})
	Error(...interface{})
	Debugf(string, ...interface{})
	Infof(string, ...interface{})
	Warnf(string, ...interface{})
	Errorf(string, ...interface{})
}

type tgBot struct {
	*tb.Bot
	*logrus.Logger
}

type tgRecipient int
func (tr tgRecipient) Recipient() string {
	return fmt.Sprint(tr)
}
func (b * tgBot) SendText(id int, msg string)error {
	_, err := b.Send(tgRecipient(id), msg)
	return err
}

func (b *tgBot) DefaultHandler(action messageHandler) {
	b.Handle(tb.OnText, func(m * tb.Message){
		sender := &User{
			FirstName: m.Sender.FirstName,
			LastName: m.Sender.LastName,
			ID: m.Sender.ID,
		}
		uid, err := db.CheckUser(sender.ID, false)
		newUser := true
		switch {
		case err != nil:
			b.Errorf("Cannot check user (id %d): %s", sender.ID, err)
		case uid == 0:
			// new user
			_, err = db.AddUser(sender, false)
			if err != nil {
				b.Errorf("Cannot add new user (id %d): %s", sender.ID, err)
			}
		default:
			// old user
			newUser = false
		}
		action(b, m.Text, sender, newUser, locales[db.GetUserLocale(sender.ID, false)].Messages)
	})
}

func (b *tgBot) CommandHandler(command string, action messageHandler) {
	b.Handle("/"+command, func(m * tb.Message){
		sender := &User{
			FirstName: m.Sender.FirstName,
			LastName: m.Sender.LastName,
			ID: m.Sender.ID,
		}
		uid, err := db.CheckUser(sender.ID, false)
		newUser := true
		switch {
		case err != nil:
			b.Errorf("Cannot check user (id %d): %s", sender.ID, err)
		case uid == 0:
			// new user
			_, err = db.AddUser(sender, false)
			if err != nil {
				b.Errorf("Cannot add new user (id %d): %s", sender.ID, err)
			}
		default:
			// old user
			newUser = false
		}
		action(b, m.Payload, sender, newUser, locales[db.GetUserLocale(sender.ID, false)].Messages)
	})
}


type vkBot struct {
	*vk.Bot
	*logrus.Logger
}

func (b *vkBot) SendText(id int, msg string) error {
	return b.SendMessage(id, msg)
}

func (b *vkBot) DefaultHandler(action messageHandler) {
	b.HandleDefault(func(m * vk.Message){
		uid, err := db.CheckUser(m.FromID, true)
		newUser := true
		sender := &User{ID: m.FromID}
		switch {
		case err != nil:
			b.Errorf("Cannot check user (id %d): %s", m.FromID, err)
		case uid == 0:
			// new user
			vkUser, err := b.GetUserByID(m.FromID)
			if err != nil {
				b.Errorf("Cannot get user (id %d) via API: %s", m.FromID, err)
				break
			}
			sender.FirstName = vkUser.FirstName
			sender.LastName = vkUser.LastName
			_, err = db.AddUser(sender, false)
			if err != nil {
				b.Errorf("Cannot add new user (id %d): %s", sender.ID, err)
			}
		default:
			// old user
			newUser = false
			sender, err = db.GetVkUser(uid)
			if err != nil {
				b.Errorf("Cannot get user (id %d) from DB: %s", m.FromID, err)
				sender = &User{ID: m.FromID}
			}
		}
		action(b, m.Text, sender, newUser, locales[db.GetUserLocale(sender.ID, true)].Messages)
	})
}

func (b *vkBot) CommandHandler(command string, action messageHandler) {
	b.HandleOnCommand(command, func(m * vk.Message){
		uid, err := db.CheckUser(m.FromID, true)
		newUser := true
		sender := &User{ID: m.FromID}
		switch {
		case err != nil:
			b.Errorf("Cannot check user (id %d): %s", sender.ID, err)
		case uid == 0:
			// new user
			vkUser, err := b.GetUserByID(m.FromID)
			if err != nil {
				b.Errorf("Cannot get user (id %d) via API: %s", sender.ID, err)
				break
			}
			sender.FirstName = vkUser.FirstName
			sender.LastName = vkUser.LastName
			_, err = db.AddUser(sender, false)
			if err != nil {
				b.Errorf("Cannot add new user (id %d): %s", sender.ID, err)
			}
		default:
			// old user
			newUser = false
			// TODO: fix this ugly bit
			sender, err = db.GetVkUser(uid)
			if err != nil {
				b.Errorf("Cannot get user (id %d) from DB: %s", m.FromID, err)
				sender = &User{ID: m.FromID}
			}
		}
		action(b, m.CommandArg(), sender, newUser, locales[db.GetUserLocale(sender.ID, true)].Messages)
	})
}
