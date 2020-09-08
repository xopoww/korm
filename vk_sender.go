package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
)

const (
	VK_API_ADDRESS = "https://api.vk.com/method"
	VK_API_VERSION = "5.95"
)

func vkSendRequest(method string, params map[string]interface{}, token, version string)(*http.Response, error) {
	paramsString := ""
	for key, value := range params {
		paramsString += fmt.Sprintf("%s=%v&", key, value)
	}
	url := fmt.Sprintf("%s/%s?%saccess_token=%s&v=%s", VK_API_ADDRESS, method, paramsString, token, version)
	vkLogger.Logf(VERBOSE,"Sending a request: %s", url)
	return http.Get(url)
}


// VK bot class
type VkBot struct {
	Token		string
	Version		string
}

func (b *VkBot) sendRequest(method string, params map[string]interface{})(*http.Response, error) {
	return vkSendRequest(method, params, b.Token, b.Version)
}

func (b *VkBot) getUser(userID int)(vkUser, error) {
	params := map[string]interface{}{
		"user_ids": userID,
		"name_case": "nom",
	}
	resp, err := b.sendRequest("users.get", params)
	if err != nil {
		return vkUser{}, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return vkUser{}, err
	}
	respObj := struct{Response []vkUser `json:"response"`}{}
	err = json.Unmarshal(body, &respObj)
	if err != nil {
		return vkUser{}, err
	}
	return respObj.Response[0], nil
}

func (b *VkBot) sendMessage(to int, msg string)error {
	params := map[string]interface{}{
		"user_id": to,
		"random_id": rand.Uint32(),
		"message": msg,//url.QueryEscape(msg),
	}
	resp, err := b.sendRequest("messages.send", params)
	if err != nil {
		return err
	}
	var(
		body []byte
		respObj struct{
			Error string `json:"error"`
		}
	)
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	vkLogger.Logf(VERBOSE, "Response body: %s", body)
	err = json.Unmarshal(body, &respObj)
	if err != nil {
		return err
	}
	if respObj.Error == "" {
		return nil
	} else {
		return errors.New(fmt.Sprintf("vk api error: %s", respObj.Error))
	}
}