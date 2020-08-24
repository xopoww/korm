package main

import (
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"io/ioutil"

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

const (
	PEM_PATH = "/home/horoshilov_aa/cert/cert.pem"
	KEY_PATH = "/home/horoshilov_aa/cert/key.pem"
	CA_PATH  = "/home/horoshilov_aa/cert/myCA.pem"
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
	err := tgBotInstance.setWebhook("https://35.228.234.83/"+TG_TOKEN,
		PEM_PATH,
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

	// custom CA
	rootCAs, _ := x509.SystemCertPool()
	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}
	certs, err := ioutil.ReadFile(CA_PATH)
	if err != nil {
		panic(err) // TODO FIX
	}
	if ok := rootCAs.AppendCertsFromPEM(certs); !ok {
		fmt.Println("No certs appended, using system certs only")
	} else {
		fmt.Println("Custom CA appended")
	}

	// http and https servers
	cert, err := tls.LoadX509KeyPair(PEM_PATH, KEY_PATH)
	if err != nil {
		panic(err) // TODO FIX
	}
	server := http.Server{
		Addr: "",
		Handler: nil,
		TLSConfig: &tls.Config{
			RootCAs: rootCAs,
			Certificates: []tls.Certificate{ cert }},
	}
	var waitGroup sync.WaitGroup
	waitGroup.Add(1)
	/*go func(){
		defer waitGroup.Done()
		vkLogger.Fatalf("Server failed: %s",
			server.ListenAndServe())
	}()*/
	go func(){
		defer waitGroup.Done()
		tgLogger.Fatalf("Server failed: %s",
			server.ListenAndServeTLS(PEM_PATH, KEY_PATH))
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