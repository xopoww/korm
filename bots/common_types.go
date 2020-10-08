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

	// Add a handler for CallbackQuery with specified action label. Answer will be sent
	// to a callback query.
	AddCallbackHandler(action, answer string, handler func(BotHandle, *CallbackQuery))
}

// An object that is sent to the KeyboardButton Action when the button is pressed.
// If an optional argument was provided bu callback query issuer (e.g. a button),
// it will be in Argument field
type CallbackQuery struct{
	From		*User
	MessageID	int
	Argument	string
}

// Inline keyboard with all buttons being callback ones.
type Keyboard struct {
	keys [][]KeyboardButton
}

// Single button for Keyboard.
// When the button is pressed, a CallbackQuery is formed and sent to the callback handler
// assigned to Action label.
type KeyboardButton struct {
	// Text of the button
	Label		string
	// VK only
	Color		string
	// Unique action label that will be put into callback data.
	// The shorter - the better.
	Action		string
	// Optional argument that can be also put into callback data.
	Argument	string
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