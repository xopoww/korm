package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
)

const (
	TG_API_ADDRESS = "https://api.telegram.org/bot"
)

var tgBotInstance *tgBot

func tgSendRequest(method string, params map[string]interface{}, token string)(*http.Response, error){
	paramsString := ""
	for key, value := range params {
		paramsString += fmt.Sprintf("%s=%v&", key, value)
	}
	url := fmt.Sprintf("%s/%s/%s?%s", TG_API_ADDRESS, token, method, paramsString)
	return http.Get(url)
}

type tgBot struct {
	token string
}

type tgResponse struct {
	Ok			bool		`json:"ok"`
	Description	string		`json:"description"`
}

func (bot * tgBot) setWebhook(webhookUrl, pemPath string, maxConns int, allowedUpds []string)error {
	pem, err := os.Open(pemPath)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	wrt := multipart.NewWriter(&buf)
	fw, err := wrt.CreateFormFile("certificate", pem.Name())
	if err != nil {
		return err
	}
	_, err = io.Copy(fw, pem)
	if err != nil {
		return err
	}
	_ = wrt.Close()
	_ = pem.Close()

	auJsoned, err := json.Marshal(allowedUpds)
	if err != nil {
		return err
	}
	paramsString := fmt.Sprintf("url=%s&max_connections=%d&allowed_updates=%s",
		webhookUrl, maxConns, string(auJsoned))
	url := fmt.Sprintf("%s/%s/setWebhook?%s", TG_API_ADDRESS, bot.token, paramsString)
	tgLogger.Logf(VERBOSE, "Webhook request: %s", url)

	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", wrt.FormDataContentType())
	var c http.Client
	resp, err := c.Do(req)
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var obj tgResponse
	err = json.Unmarshal(body, &obj)
	if err != nil {
		return err
	}
	if !obj.Ok {
		return errors.New(obj.Description)
	}
	tgLogger.Infof("TG API: %s", obj.Description)
	return nil
}