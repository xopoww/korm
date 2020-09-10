package main

import (
	"database/sql"
	"encoding/json"
	_ "github.com/mattn/go-sqlite3"
	vk "github.com/xopoww/vk_min_api"
	"io/ioutil"
	"math/rand"

	"fmt"
	"github.com/xopoww/gologs"
	"net/http"
	"os"
	"sync"
)

var (
	vkLogger = gologs.NewLogger("VK handler")
	dbLogger = gologs.NewLogger("SQL handler")
	tgLogger = gologs.NewLogger("TG handler")
)

var locales map[string]locale

var VERBOSE = gologs.LogLevel{Value: 5, Label: "VERBOSE"}

func main() {
	var lvl gologs.LogLevel
	if getAnswer("Would you like a verbose debug logging?") {
		lvl = VERBOSE
	} else {
		lvl = gologs.DEBUG
	}
	vkLogger.AddWriter(os.Stdout, lvl)
	tgLogger.AddWriter(os.Stdout, lvl)
	dbLogger.AddWriter(os.Stdout, lvl)

	// messages from JSON
	var err error
	locales, err = loadMessages()
	if err != nil {
		panic(err)
	}

	// VK initialization
	VK_TOKEN := os.Getenv("VK_TOKEN")
	vkBot, err := vk.NewBot(
		vk.Properties{
			Token: VK_TOKEN,
			Version: "5.95",
			Secret: "testing",
		},
		false, &vkLogger)
	if err != nil {
		vkLogger.Fatalf("error initializing vk bot: %s", err)
	}
	vkLogic(vkBot)
	http.HandleFunc("/vk", vkBot.HTTPHandler())

	// TG initialization
	TG_TOKEN := os.Getenv("TG_TOKEN")
	tgBot, err := tgInit(TG_TOKEN)
	if err != nil {
		tgLogger.Fatalf("Error initializing telebot: %s", err)
	}

	// database initialization
	db, err = sql.Open("sqlite3", dbname)
	if err != nil {
		dbLogger.Fatalf("Error opening a database: %s", err)
		return
	}
	defer db.Close()
	_, err = db.Exec(dbCreation)
	if err != nil {
		dbLogger.Fatalf("Error initializing a database: %s", err)
		return
	}
	oldKeysEraser()

	var waitGroup sync.WaitGroup
	waitGroup.Add(1)
	go func(){
		defer waitGroup.Done()
		vkLogger.Fatalf("Server failed: %s",
			http.ListenAndServe("", nil))
	}()

	go tgBot.Start()
	go func(){
		vkBot.Start()
		waitGroup.Done()
	}()

	waitGroup.Wait()
}

// utils

func debugBlock() {
	var test string
	fmt.Println("Blocked.")
	_, _ = fmt.Scanln(&test)
	fmt.Printf("Unblocked: %s\n", test)
}

func getAnswer(prompt string)bool {
	for {
		fmt.Println(prompt)
		fmt.Println("[y/n]")
		var ans string
		fmt.Scan(&ans)

		switch ans {
		case "y":
			return true
		case "n":
			return false
		default:
			fmt.Println("Unrecognized input.")
		}
	}
}

func getHeader(r *http.Request, key string)string {
	if values := r.Header["key"]; len(values) > 0 {
		return values[0]
	} else {
		return ""
	}
}

func wrapHandler(handler func(http.ResponseWriter, *http.Request)error, logger gologs.Logger)func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := handler(w, r); err != nil {
			logger.Errorf("Error handling a request: %s", err)
		}
	}
}

const (
	alphabet = "qwertyuiopasdfghjklzxcvbnm1234567890"
	KEY_LEN = 15
)
func generateKeyString(length int)string {
	if length <= 0 {
		panic("length must be positive")
	}
	key := make([]rune, length)
	for i := 0; i < length; i++ {
		key[i] = []rune(alphabet)[rand.Int() % len(alphabet)]
	}
	return string(key)
}

func randEmoji()string {
	emojis := []string{"\U0001f643","\U0001f609","\U0001f914","\U0001f596","\U0001f60a","\U0001f642","\U0000261d"}
	return emojis[rand.Int() % len(emojis)]
}

// messages

type messages struct {
	// common
	Error			string		`json:"error"`
	UnknownCommand	string		`json:"unknown_command"`
	// start
	Hello			string		`json:"hello"`
	HelloAgain		string		`json:"hello_again"`
	// sync
	AlreadySynced	string		`json:"already_synced"`
	EmitKeyTG		string		`json:"emit_key_tg"`
	EmitKeyVK		string		`json:"emit_key_vk"`
	SendToVK		string		`json:"send_to_vk"`
	SendToTG		string		`json:"send_to_tg"`
	UnknownKey		string		`json:"unknown_key"`
}

type locale struct {
	Repr			string		`json:"repr"`
	Messages		messages	`json:"messages"`
}

const messagesFile = "messages.json"
func loadMessages()(map[string]locale, error) {
	file, err := os.Open(messagesFile)
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var locales map[string]locale
	err = json.Unmarshal(data, &locales)
	if err != nil {
		return nil, err
	}
	return locales, nil
}