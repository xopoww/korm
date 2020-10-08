package bots

import (
	"encoding/json"
	"errors"
	"fmt"
	tg "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/sirupsen/logrus"
	"net/url"
	"sync/atomic"

	. "github.com/xopoww/korm/types"
)

// Telegram implementation of BotHandle interface
type tgBot struct {
	*tg.BotAPI

	commandHandlers		map[string]func(*tg.Message)
	callbackHandlers	map[string]callbackHandler
	defaultHandler		func(*tg.Message)

	logger				*logrus.Logger

	buttonCounter		uint32
}

// Create a new TgBot
func NewTgBot(token string, logger *logrus.Logger) (BotHandle, error) {
	bot, err := tg.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("bot api: %w", err)
	}

	return &tgBot{
		BotAPI:          	bot,
		commandHandlers:	make(map[string]func(*tg.Message)),
		callbackHandlers:	make(map[string]callbackHandler),
		logger:				logger,
		buttonCounter:		0,
	}, nil
}

type callbackHandler struct {
	// answer that will be sent after the query is received
	answer		string
	action		func(*tg.CallbackQuery)
}

// create a unique id for keyboard button
func (bot * tgBot) createButtonID() string {
	return fmt.Sprint(atomic.AddUint32(&bot.buttonCounter, 1))
}

// Convert Keyboard to telegram reply markup and register all needed handlers.
func (bot * tgBot) processKeyboard(keyboard * Keyboard) *tg.InlineKeyboardMarkup {
	if keyboard == nil {
		return nil
	}
	keys := make([][]tg.InlineKeyboardButton, len(keyboard.keys))
	for i, row := range keyboard.keys {
		keys[i] = make([]tg.InlineKeyboardButton, len(row))
		for j, button := range row {
			// get a unique id for a button
			id := bot.createButtonID()
			// add the button with id as callback data to the keyboard
			keys[i][j] = tg.NewInlineKeyboardButtonData(button.Label, id)
			// register a handler
			bot.callbackHandlers[id] = callbackHandler{
				answer: button.Answer,
				action: func(cq * tg.CallbackQuery) {
					if action := button.Action; action != nil {
						action(bot, &CallbackQuery{
							From:      stripTgUser(cq.From),
							MessageID: cq.Message.MessageID,
						})
					}

				},
			}
			bot.logger.Tracef("Processed button (%s) and assigned it id %s", button.Label, id)
		}
	}
	markup := tg.NewInlineKeyboardMarkup(keys...)
	return &markup
}

// ==== bot interface implementation ====

func (bot * tgBot) Start() error {
	uCfg := tg.UpdateConfig{
		Offset:  0,
		Limit:   0,
		Timeout: 60,
	}
	updates, err := bot.GetUpdatesChan(uCfg)
	if err != nil {
		return fmt.Errorf("get updates chan: %w", err)
	}

	for upd := range updates {
		// text message
		if m := upd.Message; m != nil {
			// command
			if m.IsCommand() {
				com := m.Command()
				bot.logger.Tracef("Got a command: %s", com)
				if hand, found := bot.commandHandlers[com]; found {
					hand(m)
					continue
				}
				// ! unhandled command
			}

			// simple text message
			if hand := bot.defaultHandler; hand != nil {
				hand(m)
				continue
			}
			// ! unhandled message
		}

		// callback query
		if cq := upd.CallbackQuery; cq != nil {
			if hand, found := bot.callbackHandlers[cq.Data]; found {
				_, err := bot.AnswerCallbackQuery(tg.NewCallback(cq.ID, hand.answer))
				if err != nil {
					bot.logger.Errorf("Error answering callback query: %s", err)
					continue
				}
				hand.action(cq)
				continue
			}
			// ! unhandled callback
		}
	}

	return errors.New("updates chan is closed")
}

func (bot *tgBot) SendMessage(text string, to *User, keyboard *Keyboard) (int, error) {
	message := tg.NewMessage(int64(to.ID), text)
	message.ReplyMarkup = bot.processKeyboard(keyboard)
	resp, err := bot.Send(message)
	if err != nil {
		return 0, err
	}
	return resp.MessageID, nil
}

func (bot *tgBot) RegisterCommands(commands ...Command) error {
	for _, com := range commands {
		bot.commandHandlers[com.Label] = func(m *tg.Message){
			com.Action(bot, stripTgUser(m.From))
		}
		bot.logger.Tracef("Registered a command: %s", com.Label)
	}

	data, err := json.Marshal(commands)
	if err != nil {
		return fmt.Errorf("marshal list of commands: %w", err)
	}
	vals := url.Values{}
	vals.Set("commands", string(data))
	resp, err := bot.MakeRequest("setMyCommands", vals)
	if err != nil {
		return fmt.Errorf("make request: %w", err)
	}
	if !resp.Ok {
		return fmt.Errorf("API error (%d): %s", resp.ErrorCode, resp.Description)
	}
	return nil
}

func (bot * tgBot) EditMessage(to *User, id int, text string, keyboard *Keyboard) error {
	var cfg tg.Chattable
	if text == "" {
		cfg = tg.NewDeleteMessage(int64(to.ID), id)
	} else {
		cfg = tg.EditMessageTextConfig{
			BaseEdit:              tg.BaseEdit{
				ChatID:          int64(to.ID),
				MessageID:       id,
				ReplyMarkup:     bot.processKeyboard(keyboard),
			},
			Text:                  text,
		}
	}
	_, err := bot.Send(cfg)
	return err
}

// ======== utils ========

// Convert telegram user to *types.User
func stripTgUser(user * tg.User) *User {
	return &User{
		FirstName: user.FirstName,
		LastName:  user.LastName,
		ID:        user.ID,
	}
}