package drafts

import (
	"fmt"
	tg "github.com/go-telegram-bot-api/telegram-bot-api"
	. "github.com/xopoww/korm/types"
	vk "github.com/xopoww/vk_min_api"
)

// Registered command for a bot.
//
// For TG bots the command must be activated by sending it in the format "/{Alias} {argument}".
//
// For VK bots, the command must be activated by pressing the button in the menu.
type Command struct {
	Name		string
	// TG bots only.
	//
	// ASCII alias for a command.
	Alias		string
	Action		func(user *User)
}



// An inline keyboard. All buttons will be callback buttons.
type Keyboard struct {
	buttons [][]Button
}

// Callback button.
type Button struct {
	// A text that will be displayed on the button
	Label		string
	// An answer that will be shown to user after pressing the button. Empty string for nothing to be shown.
	Answer		string
	Action		func(user *User)
	// Supported only by VK bots
	Color		string
}

func (b * vkBot) Process(keyboard * Keyboard) vk.Keyboard {
	buttons := make([][]vk.KeyboardButton, len(keyboard.buttons))
	for i, row := range keyboard.buttons {
		buttons[i] = make([]vk.KeyboardButton, len(row))
		for j, button := range row {
			// get unique id for a button
			id := b.newButtonID()
			// add a data button with id as data
			buttons[i][j] = vk.NewDataButton(button.Label, id, button.Color)
			// register callback handler for this id
			b.HandleCallback(
				func(data interface{}) bool {
					if idGot, ok := data.(int); ok {
						return idGot == int(id)
					}
					return false
				},
				func(m *vk.MessageEvent){
					// answer an event
					err := b.SendMessageEventAnswer(m, button.Answer)
					if err != nil {
						b.Errorf("Cannot answer to callback: %s", err)
						return
					}
					// take action
					sender := &User{ID: m.UserID}
					button.Action(sender)
				})
		}
	}
	// make a vk.Keyboard and return it
	return vk.Keyboard{Inline: true, Buttons: buttons}
}

func (b * tgBot) Process(keyboard * Keyboard) interface{} {
	buttons := make([][]tg.InlineKeyboardButton, len(keyboard.buttons))
	for i, row := range keyboard.buttons {
		buttons[i] = make([]tg.InlineKeyboardButton, len(row))
		for j, button := range row {
			id := fmt.Sprint(b.newButtonID())
			buttons[i][j] = tg.NewInlineKeyboardButtonData(button.Label, id)
			b.callbackHandlers = append(b.callbackHandlers,
				tgCallbackHandler{
					condition: func(s string) bool { return s == id },
					action:    func(q *tg.CallbackQuery){
						user := &User{
							FirstName: q.From.FirstName,
							LastName:  q.,
							ID:        0,
						}
					},
			})

		}
	}
}