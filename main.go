package main

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"

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

	// VK initialization
	VK_TOKEN := os.Getenv("VK_TOKEN")
	vkBotInstance = &VkBot{VK_TOKEN, VK_API_VERSION}
	vkLogger.Info("Initialized a VK bot.")
	vkLogger.Logf(VERBOSE, "\ttoken: %s", VK_TOKEN)
	http.HandleFunc("/vk", wrapHandler(vkHandler, vkLogger))

	// TG initialization
	TG_TOKEN := os.Getenv("TG_TOKEN")
	tgBotInstance = &tgBot{TG_TOKEN}
	tgLogger.Info("Initialized a TG bot.")
	tgLogger.Logf(VERBOSE, "\ttoken: %s", TG_TOKEN)
	http.HandleFunc("/"+TG_TOKEN, wrapHandler(tgHandler, tgLogger))
	err := tgBotInstance.setWebhook("https://35.228.234.83:8443/"+TG_TOKEN,
		"/home/horoshilov_aa/cert/PUBLIC.pem",
		40, []string{})
	if err != nil {
		tgLogger.Fatalf("Error setting a webhook: %s", err)
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

	// http server
	var waitGroup sync.WaitGroup
	waitGroup.Add(1)
	go func(){
		defer waitGroup.Done()
		vkLogger.Fatalf("Server failed: %s", http.ListenAndServe("", nil))
	}()

	// VK request processing
	for i := 0; i < REQUEST_PROCESSERS; i++ {
		go func() {
			for body := range requestChan {
				vkLogger.Debug("Got a request from channel.")
				if err := processRequest(vkBotInstance, body); err != nil {
					vkLogger.Errorf("Error processing a request: %s", err)
				}
			}
		}()
	}

	waitGroup.Wait()
}

// utils

func debugBlock() {
	var test string
	fmt.Println("Blocked.")
	fmt.Scanln(&test)
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