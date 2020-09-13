package main

import (
	"fmt"
	vk "github.com/xopoww/vk_min_api"
	tb "gopkg.in/tucnak/telebot.v2"
)

type User struct {
	FirstName	string
	LastName	string
	ID			int
}

type Bot interface {
	SendText(id int, msg string) error
	GetContents(message interface{}) (text string, fromID int)
	GetSender(message interface{}) User

	DefaultHandler(action func(bot Bot, m interface{}))
	CommandHandler(command string, action func(bot Bot, m interface{}))

	checkUser(id int)(uid int, err error)
	addUser(user * User)(uid int, err error)
	getUser(uid int)(User, error)
	getUserLocale(id int)*messageTemplates
}

type tgBot struct {
	*tb.Bot
}

type tgRecipient int
func (tr tgRecipient) Recipient() string {
	return fmt.Sprint(tr)
}

func (b * tgBot) SendText(id int, msg string)error {
	_, err := b.Send(tgRecipient(id), msg)
	return err
}

func (b * tgBot) GetContents(m interface{})(text string, fromID int) {
	tgm := m.(tb.Message)
	return tgm.Text, tgm.Sender.ID
}

func (b * tgBot) GetSender(m interface{})User {
	u := (m).(tb.Message).Sender
	return User{u.FirstName, u.LastName, u.ID}
}

func (b *tgBot) DefaultHandler(action func(bot Bot, m interface{})) {
	b.Handle(tb.OnText, func(m * tb.Message){
		action(b, m)
	})
}

func (b *tgBot) CommandHandler(command string, action func(bot Bot, m interface{})) {
	b.Handle("/"+command, func(m * tb.Message){
		action(b, m)
	})
}

func (b *tgBot) checkUser(id int) (uid int, err error) {
	return checkUser(id, false)
}

func (b *tgBot) addUser(user * User) (uid int, err error) {
	return addUser(user, false)
}

func (b *tgBot) getUser(uid int) (User, error) {
	panic("implement me")
}

func (b *tgBot) getUserLocale(id int) *messageTemplates {
	return getUserLocale(id, false)
}




type vkBot struct {
	*vk.Bot
}

func (b *vkBot) SendText(id int, msg string) error {
	return b.SendMessage(id, msg)
}

func (b *vkBot) GetContents(message interface{}) (text string, fromID int) {
	vkm := message.(vk.Message)
	return vkm.Text, vkm.FromID
}

func (b *vkBot) GetSender(message interface{}) User {
	fromID := message.(vk.Message).FromID
	user, err := b.GetUserByID(fromID)
	if err != nil {
		panic(err)
	}
	return User{user.FirstName, user.LastName, user.ID}
}

func (b *vkBot) DefaultHandler(action func(bot Bot, m interface{})) {
	b.HandleDefault(func(m * vk.Message){
		action(b, m)
	})
}

func (b *vkBot) CommandHandler(command string, action func(bot Bot, m interface{})) {
	b.HandleOnCommand(command, func(m * vk.Message){
		action(b, m)
	})
}

func (b *vkBot) checkUser(id int) (uid int, err error) {
	return checkUser(id, true)
}

func (b *vkBot) addUser(user *User) (uid int, err error) {
	return addUser(user, true)
}

func (b *vkBot) getUser(uid int) (User, error) {
	user, err := getVkUser(uid)
	if err != nil {
		return User{}, err
	}
	return User{user.FirstName, user.LastName, user.ID}, nil
}

func (b *vkBot) getUserLocale(id int) *messageTemplates {
	return getUserLocale(id, true)
}