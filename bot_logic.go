package main

import (
	"fmt"
	"github.com/xopoww/korm/bots"
	db "github.com/xopoww/korm/database"
	. "github.com/xopoww/korm/types"
	"strconv"
)

const menuText = "Наше меню:"

//type UserSession struct {
//	MenuKeys		*bots.Keyboard
//	KindKeys		map[int]*bots.Keyboard
//	CurrentOrder	*Order
//}
//
//var sessions map[int]UserSession

//func createMenuKeyboard() (*bots.Keyboard, error) {
//	kinds, err := db.GetDishKinds()
//	if err != nil {
//		return nil, err
//	}
//
//	keyboard := &bots.Keyboard{}
//	for _, kind := range kinds {
//		button := bots.KeyboardButton{
//			Label: kind.Repr,
//			Action: "menu",
//			Argument: fmt.Sprint(kind.ID),
//		}
//		keyboard.AddRow(button)
//	}
//	return keyboard, nil
//}

//func createDishesKeyboard(kindID int) (*bots.Keyboard, error) {
//	dishes, err := db.GetDishesByKind(DishKind{ID: kindID})
//	if err != nil {
//		return nil, err
//	}
//
//	keyboard := &bots.Keyboard{}
//	for _, dish := range dishes {
//		button := bots.KeyboardButton{
//			Label: fmt.Sprintf("%s (%d)", dish.Name, dish.Quantity),
//			Action: "doodle",
//		}
//		keyboard.AddRow(button)
//	}
//	back := bots.KeyboardButton{
//		Label: "назад",
//		Action: "menu",
//		Argument: "back",
//	}
//	keyboard.AddRow(back)
//	return keyboard, nil
//}

//func menuCallback(bot bots.BotHandle, cq * bots.CallbackQuery) {
//	var (
//		keys *bots.Keyboard
//		err error
//	)
//	switch cq.Argument {
//	case "back":
//		keys, err = createMenuKeyboard()
//		if err != nil {
//			bot.Errorf("Create menu keyboard: %s", err)
//			return
//		}
//	default:
//		kindID, err := strconv.Atoi(cq.Argument)
//		if err != nil {
//			bot.Errorf("Atoi (string %s): %s", cq.Argument, err)
//			return
//		}
//		keys, err = createDishesKeyboard(kindID)
//		if err != nil {
//			bot.Errorf("Create dishes keyboard (id %d): %s", kindID, err)
//			return
//		}
//	}
//	err = bot.EditMessage(cq.From, cq.MessageID, menuText, keys)
//	if err != nil {
//		bot.Errorf("Edit message: %s", err)
//	}
//}

func createMenuKeyboard() (*bots.Keyboard, error) {
	keys := bots.Keyboard{};
	keys.AddRow(bots.KeyboardButton{
		Label:    "Кормы",
		Action:   "menu",
		Argument: "0",
	})
	keys.AddRow(bots.KeyboardButton{
		Label:    "Супы",
		Action:   "menu",
		Argument: "1",
	})
	keys.AddRow(bots.KeyboardButton{
		Label:		"Напики",
		Action:		"menu",
		Argument:	"2",
	})
	keys.AddRow(bots.KeyboardButton{
		Label:		"Заказать",
		Action:		"order",
	}, bots.KeyboardButton{
		Label: "Сбросить",
		Action: "back",
		Argument: "cancel",
	})
	return &keys, nil
}

type DishMock struct {
	name string
	price int
	id	int
}

var (
	korms = []DishMock{
	{name: "Курица терияки", price: 185, id: 1},
	{name: "Паста карбонара", price: 225, id: 2},
	{name: "Жаркое из свинины", price: 185, id: 3},
	}

	soups = []DishMock{
		{name: "Борщ", price: 105, id: 4},
	}

	drinks = []DishMock{
		{name: "Апельсин-имбирь", price: 50, id: 5},
		{name: "Ягодный пунш", price: 50, id: 6},
	}
)

func getDishMockByID(id int) DishMock {
	if id > 0 && id < 4 {
		return korms[id - 1]
	}
	if id == 4 {
		return soups[0]
	}
	if id > 4 && id < 7 {
		return drinks[id - 5]
	}
	return DishMock{}
}

func createDishKeyboard(kind string) *bots.Keyboard {
	var dishes []DishMock
	switch kind {
	case "0":
		dishes = korms
	case "1":
		dishes = soups
	case "2":
		dishes = drinks
	}
	keys := bots.Keyboard{}
	for _, dish := range dishes {
		keys.AddRow(bots.KeyboardButton{
			Label: fmt.Sprintf("%s - %dр.", dish.name, dish.price),
			Action: "add",
			Argument: fmt.Sprint(dish.id),
		})
	}
	keys.AddRow(bots.KeyboardButton{Label: "назад", Action: "back"})
	return &keys
}

var OrderMock map[int]int

func ListOrderMock() string {
	if len(OrderMock) == 0 {
		return "Ваш заказ пока что пуст. Добавьте блюда при помощи клавиатуры:"
	}
	msg := ""
	price := 0
	for id, quantity := range OrderMock {
		dish := getDishMockByID(id)
		msg += fmt.Sprintf("%s - %d шт.\n", dish.name, quantity)
		price += dish.price
	}
	msg += fmt.Sprintf("\nСтоимость заказа: %dр.", price)
	return msg
}

func InitializeBots(handles ...bots.BotHandle) error {

	menuKeys, _ := createMenuKeyboard()

	startCommand := bots.Command{
		Name:	"начать общение с ботом",
		Label:	"start",
		Action: func(bot bots.BotHandle, user *User) {
			msg := fmt.Sprintf("Здравствуй, %s! Я - бот КОРМа. Напиши /order, чтобы сделать заказ.",
				user.FirstName)
			_, _ = bot.SendMessage(msg, user, nil)
		},
	}

	menuCommand := bots.Command{
		Name:   "сделать заказ",
		Label:  "order",
		Action: func(bot bots.BotHandle, user *User) {
			keys, err := createMenuKeyboard()
			if err != nil {
				bot.Errorf("Create menu keyboard: %s", err)
				return
			}
			OrderMock = nil
			_, err = bot.SendMessage(ListOrderMock(), user, keys)
			if err != nil {
				bot.Errorf("Send message: %s", err)
				return
			}
		},
	}

	for _, bot := range handles {
		err := bot.RegisterCommands(startCommand, menuCommand)
		if err != nil {
			return err
		}

		bot.AddCallbackHandler("menu", "",
			func(bot bots.BotHandle, cq *bots.CallbackQuery){
				_ = bot.EditMessage(cq.From, cq.MessageID, ListOrderMock(), createDishKeyboard(cq.Argument))
			})

		bot.AddCallbackHandler("add", "Добавлено в заказ",
			func(bot bots.BotHandle, cq *bots.CallbackQuery) {
				if OrderMock == nil {
					OrderMock = make(map[int]int)
				}
				id, _ := strconv.Atoi(cq.Argument)
				if _, found := OrderMock[id]; found {
					OrderMock[id]++
				} else {
					OrderMock[id] = 1
				}
				_ = bot.EditMessage(cq.From, cq.MessageID, ListOrderMock(), menuKeys)
			})

		bot.AddCallbackHandler("back", "",
			func(bot bots.BotHandle, cq *bots.CallbackQuery){
				if cq.Argument == "cancel" {
					OrderMock = nil
				}
				_ = bot.EditMessage(cq.From, cq.MessageID, ListOrderMock(), menuKeys)
			})

		bot.AddCallbackHandler("order", "",
			func(bot bots.BotHandle, cq *bots.CallbackQuery){
				_ = bot.EditMessage(cq.From, cq.MessageID, "", nil)
				_, _ = bot.SendMessage("Ваш заказ успешно оформлен! Ожидайте, наш курьер с вами свяжется.",
					cq.From, nil)
			})
	}

	return nil
}

// Wrap a command handler to check if the user is in the database
// (and add him if he is not) and populate user.UID field.
func CheckOrAddUser(action func(bots.BotHandle, *User)) func(bots.BotHandle, *User) {
	return func(bot bots.BotHandle, user *User) {
		// TODO: figure out the best way to connect bot handle to database
		vk := false

		uid, err := db.CheckUser(user.ID, vk)
		if err != nil {
			bot.Errorf("Check user (id %d): %s", user.ID, err)
			return
		}
		if uid == 0 {
			uid, err = db.AddUser(user, vk)
			if err != nil {
				bot.Errorf("Add user (id %d): %s", user.ID, err)
				return
			}
		}
		user.UID = uid
		action(bot, user)
	}
}
