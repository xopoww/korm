package main

import (
	"github.com/xopoww/gologs"
	"net/http"
	"os"
)

var vkLogger = gologs.NewLogger("VK handler")

func main() {
	vkLogger.AddWriter(os.Stdout, gologs.DEBUG)


	VK_TOKEN := os.Getenv("VK_TOKEN")
	vkBotInstance = &VkBot{VK_TOKEN, VK_API_VERSION}
	vkLogger.Infof("Initialized a VK bot with a token %s", VK_TOKEN)

	http.HandleFunc("/vk", wrapHandler(vkHandler))
	vkLogger.Fatalf("Server failed: %s", http.ListenAndServe("", nil))
}
