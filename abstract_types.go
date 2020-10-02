package main

import (
	"github.com/sirupsen/logrus"

	tg "github.com/go-telegram-bot-api/telegram-bot-api"
	vk "github.com/xopoww/vk_min_api"

	db "github.com/xopoww/korm/database"
	. "github.com/xopoww/korm/types"
)

type KeyboardButton struct {
	Caption		string
	Data		string
	Color		string
}

type Keyboard struct {
	rows		[][]KeyboardButton
}

func (k * Keyboard) AddRow(buttons ...KeyboardButton) {
	k.rows = append(k.rows, buttons)
}

// Message handler is a function for handling text messages by bot.
// It must be passed as an argument to either DefaultHandler or CommandHandler methods of BotHandle.
// If it is a default handler, text will be a text of the message. If it is a CommandHandler, text will
// be a payload of the specified command.
type messageHandler func(bot BotHandle, text string, sender *User, newUser bool, messages *messageTemplates)

type BotHandle interface {
	// 	Send a text message
	// If keyboard is not nil, it will be transformed to corresponding keyboard object and sent
	// as an inline markup.
	SendText(id int, msg string, keyboard *Keyboard) error

	//  Add on-text handler to the bot
	// Before passing the message to the message handler, it checks if the user is added to the database
	// (and passes the result of the check as a newUser parameter to the messageHandler), and adds him if necessary.
	DefaultHandler(action messageHandler)

	// 	Add on-command handler to the bot
	// Performs the same actions before passing the message to the messageHandler as DefaultHandler
	CommandHandler(command string, action messageHandler)

	//  Add callback query handler
	CallbackHandler(condition func(string)bool, action messageHandler)

	// TODO: fix collision with tg.BotAPI.Debug
	//Debug(...interface{})
	Info(...interface{})
	Warn(...interface{})
	Error(...interface{})
	Debugf(string, ...interface{})
	Infof(string, ...interface{})
	Warnf(string, ...interface{})
	Errorf(string, ...interface{})
}
/*
type tgBot struct {
	*tb.Bot
	*logrus.Logger
}

type tgRecipient int
func (tr tgRecipient) Recipient() string {
	return fmt.Sprint(tr)
}
func (b * tgBot) SendText(id int, msg string, keyboard *Keyboard)error {

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
*/

type vkBot struct {
	*vk.Bot
	*logrus.Logger
}

func (b *vkBot) SendText(id int, msg string, keyboard *Keyboard) error {
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
			_, err = db.AddUser(sender, true)
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
			_, err = db.AddUser(sender, true)
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
		action(b, m.CommandArg(), sender, newUser, locales[db.GetUserLocale(sender.ID, true)].Messages)
	})
}

func (b *vkBot) CallbackHandler(condition func(string)bool, action messageHandler) {
	// TODO: implement
}


type tgBot struct {
	*tg.BotAPI
	*logrus.Logger

	callbackHandlers []tgCallbackHandler
	commandHandlers map[string]func(*tg.Message)
	textHandler func(*tg.Message)
}

func NewTgBot(token string, logger *logrus.Logger) (*tgBot, error) {
	tbot, err := tg.NewBotAPI(token)
	if err != nil {
		return nil, err
	}
	return &tgBot{
		BotAPI:           tbot,
		Logger:           logger,
		callbackHandlers: make([]tgCallbackHandler, 0),
		commandHandlers:  make(map[string]func(*tg.Message)),
		textHandler:      nil,
	}, nil
}

type tgCallbackHandler struct {
	condition func(string)bool
	action func(*tg.CallbackQuery)
}

// 	Start receiving updates
func (b * tgBot) Start() {
	updates, err := b.GetUpdatesChan(tg.UpdateConfig{
		Offset: 0,
		Limit: 0,
		Timeout: 60,
	})
	if err != nil {
		b.Fatalf("Could not get updates chan: %s", err)
	}

	for u := range updates {
		// CallbackQuery
		if cq := u.CallbackQuery; cq != nil {
			// answer it with nothing
			_, err = b.AnswerCallbackQuery(tg.NewCallback(cq.ID, ""))
			if err != nil {
				b.Errorf("Could not answer callback query: %s.", err)
				continue
			}
			// handle it
			b.handleCallback(cq)
		}

		// Message
		if m := u.Message; m != nil {
			com, _ := parseCommand(m.Text)
			if handler, found := b.commandHandlers[com]; found {
				handler(m)
			} else {
				b.textHandler(m)
			}
		}
	}
}

func (b * tgBot) handleCallback(cq *tg.CallbackQuery) {
	for _, handler := range b.callbackHandlers {
		if handler.condition(cq.Data) {
			handler.action(cq)
			return
		}
	}
	b.Warningf("Uncaught callback query with data: %s", cq.Data)
}

func (b * tgBot) DefaultHandler(action messageHandler) {
	b.textHandler = func(m * tg.Message) {
		sender := &User{
			FirstName: m.From.FirstName,
			LastName: m.From.LastName,
			ID: m.From.ID,
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
	}
}

func (b *tgBot) CommandHandler(command string, action messageHandler) {
	b.commandHandlers[command] = func(m * tg.Message) {
		sender := &User{
			FirstName: m.From.FirstName,
			LastName: m.From.LastName,
			ID: m.From.ID,
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
		_, arg := parseCommand(m.Text)
		action(b, arg, sender, newUser, locales[db.GetUserLocale(sender.ID, false)].Messages)
	}
}

func (b *tgBot) CallbackHandler(condition func(string)bool, action messageHandler) {
	b.callbackHandlers = append(b.callbackHandlers,
		tgCallbackHandler{
			condition: condition,
			action: func(q * tg.CallbackQuery) {
				sender := &User{
					FirstName: q.From.FirstName,
					LastName: q.From.LastName,
					ID: q.From.ID,
				}
				action(b, q.Data, sender, false, locales[db.GetUserLocale(sender.ID, false)].Messages)
			},
		})
}

func (b *tgBot) SendText(id int, msg string, keyboard *Keyboard) error {
	message := tg.NewMessage(int64(id), msg)
	if keyboard != nil {
		tgRows := make([][]tg.InlineKeyboardButton, len(keyboard.rows))
		for _, row := range keyboard.rows {
			tgRow := make([]tg.InlineKeyboardButton,len(row))
			for _, button := range row {
				tgRow = append(tgRow, tg.NewInlineKeyboardButtonData(button.Caption, button.Data))
			}
			tgRows = append(tgRows, tgRow)
		}
		message.ReplyMarkup = tg.NewInlineKeyboardMarkup(tgRows...)
	}
	_, err := b.Send(message)
	return err
}