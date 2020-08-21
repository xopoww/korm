package main

import (
	"fmt"
	"github.com/xopoww/gologs"
	"net/http"
	"os"
)

var vkLogger = gologs.NewLogger("VK handler")
//var dbLogger = gologs.NewLogger("SQL handler")

var VERBOSE = gologs.LogLevel{5, "VERBOSE"}

func main() {
	fmt.Println("Would you like a verbose debug output? (\"yes\" to set verbose, anything else to skip it)")
	var (
		ans string
		lvl gologs.LogLevel
	)
	fmt.Scan(&ans)
	if ans == "yes" {
		lvl = VERBOSE
	} else {
		lvl = gologs.DEBUG
	}

	vkLogger.AddWriter(os.Stdout, lvl)
	//dbLogger.AddWriter(os.Stdout, gologs.DEBUG)


	VK_TOKEN := os.Getenv("VK_TOKEN")
	vkBotInstance = &VkBot{VK_TOKEN, VK_API_VERSION}
	vkLogger.Infof("Initialized a VK bot with a token %s", VK_TOKEN)

	http.HandleFunc("/vk", wrapHandler(vkHandler))
	go vkLogger.Fatalf("Server failed: %s", http.ListenAndServe("", nil))

	for body := range requestChan {
		vkLogger.Debug("Got a request from channel.")
		if err := processRequest(vkBotInstance, body); err != nil {
			vkLogger.Errorf("Error processing a request: %s", err)
		}
	}
}
