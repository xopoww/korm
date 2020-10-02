package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/xopoww/gologs"
	vk "github.com/xopoww/vk_min_api"
	tb "gopkg.in/tucnak/telebot.v2"
	"path/filepath"

	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"
	"flag"

	"github.com/xopoww/korm/admin"
	db "github.com/xopoww/korm/database"
)

var locales map[string]*locale

func main() {
	rand.Seed(time.Now().Unix())

	trace := flag.Bool("trace", false, "set logger level to trace")
	flag.Parse()
	lvl := logrus.DebugLevel
	if *trace {
		lvl = logrus.TraceLevel
	}

	// loggers
	logger := &logrus.Logger{
		Out: os.Stdout,
		Formatter: &logrus.TextFormatter{DisableLevelTruncation: true},
		Level: lvl,
	}

	// messages from JSON
	var err error
	locales, err = loadMessages()
	if err != nil {
		panic(err)
	}

	// main router
	router := mux.NewRouter()

	// VK initialization
	VK_TOKEN := os.Getenv("VK_TOKEN")
	vbot, err := vk.NewBot(
		vk.Properties{
			Token: VK_TOKEN,
			Version: "5.95",
			Secret: "testing",
		},
		false, &gologs.Logger{})
	if err != nil {
		panic(err)
	}
	router.HandleFunc("/vk", vbot.HTTPHandler())

	// TG initialization
	TG_TOKEN := os.Getenv("TG_TOKEN")
	tbot, err := tb.NewBot(tb.Settings{
		Token:  TG_TOKEN,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		logger.Fatalf("Error initializing telebot: %s", err)
	} else {
		logger.Info("Initialized a TG bot.")
	}

	// abstract bot inits
	AddHandlers(
		&vkBot{vbot, logger},
		&tgBot{tbot, logger},
		)

	// Init a database
	db.Start(&db.Config{
		Filename: "korm.db",
		InitScript: filepath.Join("database", "database_creation.sql"),
		Logger: logger,
	})
	go db.StartWorkers()

	// admin app
	admin.SetAdminRoutes(router.PathPrefix("/admin").Subrouter())
	admin.SetApiRoutes(router.PathPrefix("/api").Subrouter())
	// TODO: get rid of this nonsense
	_ = db.AddAdmin("admin", "admin", "Arseny")

	var waitGroup sync.WaitGroup
	waitGroup.Add(1)
	go func(){
		defer waitGroup.Done()
		fmt.Printf("Server failed: %s\n",
			http.ListenAndServe("", router))
	}()

	go tbot.Start()
	go func(){
		vbot.Start()
		waitGroup.Done()
	}()

	waitGroup.Wait()
}

// utils


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

type messageTemplates struct {
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
	Messages		*messageTemplates	`json:"messages"`
}

const messagesFile = "messages.json"
func loadMessages()(map[string]*locale, error) {
	file, err := os.Open(messagesFile)
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var locales map[string]*locale
	err = json.Unmarshal(data, &locales)
	if err != nil {
		return nil, err
	}
	return locales, nil
}