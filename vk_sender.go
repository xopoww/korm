package main

import (
	"fmt"
	"io/ioutil"

	"encoding/json"
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
	vkLogger.Debugf("Sending a request: %s", url)
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

