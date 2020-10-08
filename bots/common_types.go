package bots

import (
	"encoding/json"
	. "github.com/xopoww/korm/types"
)

// Interface for communicating with bot API
type BotHandle interface{
	// Start receiving and handling the updates from the bot. Blocking function.
	// Returns only fatal errors.
	Start() error

	// Send a text message to user.
	// If keyboard is not nil, it is attached to the message.
	// On success returns the conversation id of the sent message.
	SendMessage(text string, to *User, keyboard * Keyboard) (int, error)

	// Edit the message previously sent to "to".
	// If text is an empty string, message is deleted.
	EditMessage(to *User, id int, text string, keyboard *Keyboard) error

	// Register a set of static commands to be available for users.
	RegisterCommands(commands ...Command) error
}

// An object that is sent to the KeyboardButton Action when the button is pressed.
type CallbackQuery struct{
	From		*User
	MessageID	int
}

// Inline keyboard with all buttons being callback ones.
type Keyboard struct {
	keys [][]KeyboardButton
}

// Single button for Keyboard.
// When the button is pressed, a CallbackQuery is formed and sent to Action
// (all handlers are set when the message with the corresponding keyboard is sent).
type KeyboardButton struct {
	Label		string
	// VK only
	Color		string
	// Answer will be sent to user after they press the button
	Answer		string
	Action		func(BotHandle, *CallbackQuery)
}

// Command represents a static (i.e. without variable arguments) bot command.
type Command struct {
	// Command name. In TG will serve as a command description.
	Name		string
	// TG only. Will be used as a command (in "/{command}" format). ASCII characters only.
	Label		string
	Action		func(BotHandle, *User)
}

// For telegram method setMyCommands
func (c Command) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{
		"command": c.Label,
		"description": c.Name,
	})
}