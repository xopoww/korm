package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"path/filepath"

	"flag"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/xopoww/korm/admin"
	db "github.com/xopoww/korm/database"
)

var locales map[string]*locale

func main() {
	rand.Seed(time.Now().Unix())

	trace := flag.Bool("trace", false, "set logger level to trace")
	//vkVerbose := flag.Bool("vk_verb", false, "set vk bot VerboseLogging option")
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

	//// messages from JSON
	//var err error
	//locales, err = loadMessages()
	//if err != nil {
	//	panic(err)
	//}

	// main router
	router := mux.NewRouter()

	// VK initialization
	//vbot, err := vk.NewBot(vk.Properties{
	//		Token: os.Getenv("VK_TOKEN"),
	//		Version: "5.95",
	//		Secret: "testing",
	//		VerboseLogging: *vkVerbose,
	//	},false)
	//if err != nil {
	//	panic(err)
	//}
	//router.HandleFunc("/vk", vbot.HTTPHandler())
	//
	//// TG initialization
	//tbot, err := NewTgBot(os.Getenv("TG_TOKEN"), logger)
	//if err != nil {
	//	logger.Fatalf("Error initializing TG bot: %s", err)
	//} else {
	//	logger.Info("Initialized a TG bot.")
	//}
	//
	//// abstract bot inits
	//AddHandlers(
	//	&vkBot{vbot, logger},
	//	tbot,
	//	)

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

	//go tbot.Start()
	//go func(){
	//	vbot.Start()
	//	waitGroup.Done()
	//}()

	waitGroup.Wait()
}

// utils

func randEmoji()string {
	emojis := []string{"\U0001f643","\U0001f609","\U0001f914","\U0001f596","\U0001f60a","\U0001f642","\U0000261d"}
	return emojis[rand.Int() % len(emojis)]
}