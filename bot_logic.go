package main

import (
	"fmt"
	"strconv"

	"github.com/xopoww/korm/bots"
	db "github.com/xopoww/korm/database"
	. "github.com/xopoww/korm/types"
)

const menuText = "Наше меню:"

func createMenuKeyboard() (*bots.Keyboard, error) {
	kinds, err := db.GetDishKinds()
	if err != nil {
		return nil, err
	}

	keyboard := &bots.Keyboard{}
	for _, kind := range kinds {
		button := bots.KeyboardButton{
			Label: kind.Repr,
			Action: "menu",
			Argument: fmt.Sprint(kind.ID),
		}
		keyboard.AddRow(button)
	}
	return keyboard, nil
}

func createDishesKeyboard(kindID int) (*bots.Keyboard, error) {
	dishes, err := db.GetDishesByKind(DishKind{ID: kindID})
	if err != nil {
		return nil, err
	}

	keyboard := &bots.Keyboard{}
	for _, dish := range dishes {
		button := bots.KeyboardButton{
			Label: fmt.Sprintf("%s (%d)", dish.Name, dish.Quantity),
			Action: "doodle",
		}
		keyboard.AddRow(button)
	}
	back := bots.KeyboardButton{
		Label: "назад",
		Action: "menu",
		Argument: "back",
	}
	keyboard.AddRow(back)
	return keyboard, nil
}

func menuCallback(bot bots.BotHandle, cq * bots.CallbackQuery) {
	var (
		keys *bots.Keyboard
		err error
	)
	switch cq.Argument {
	case "back":
		keys, err = createMenuKeyboard()
		if err != nil {
			bot.Errorf("Create menu keyboard: %s", err)
			return
		}
	default:
		kindID, err := strconv.Atoi(cq.Argument)
		if err != nil {
			bot.Errorf("Atoi (string %s): %s", cq.Argument, err)
			return
		}
		keys, err = createDishesKeyboard(kindID)
		if err != nil {
			bot.Errorf("Create dishes keyboard (id %d): %s", kindID, err)
			return
		}
	}
	err = bot.EditMessage(cq.From, cq.MessageID, menuText, keys)
	if err != nil {
		bot.Errorf("Edit message: %s", err)
	}
}

func InitializeBots(handles ...bots.BotHandle) error {
	menuCommand := bots.Command{
		Name:   "посмотреть меню",
		Label:  "menu",
		Action: func(bot bots.BotHandle, user *User) {
			keys, err := createMenuKeyboard()
			if err != nil {
				bot.Errorf("Create menu keyboard: %s", err)
				return
			}
			_, err = bot.SendMessage(menuText, user, keys)
			if err != nil {
				bot.Errorf("Send message: %s", err)
				return
			}
		},
	}

	for _, bot := range handles {
		err := bot.RegisterCommands(menuCommand)
		if err != nil {
			return err
		}

		bot.AddCallbackHandler("menu", ".", menuCallback)
		bot.AddCallbackHandler("doodle", "doodle", nil)
	}

	return nil
}
