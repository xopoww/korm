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

func vkSendRequest(method string, params map[string]string, token, version string)(*http.Response, error) {
	paramsString := ""
	for key, value := range params {
		paramsString += fmt.Sprintf("%s=%s&", key, value)
	}
	url := fmt.Sprintf("%s/%s?%saccess_token=%s&v=%s", VK_API_ADDRESS, method, paramsString, token, version)
	return http.Get(url)
}


// VK bot class
type VkBot struct {
	Token		string
	Version		string
}

func (b *VkBot) sendRequest(method string, params map[string]string)(*http.Response, error) {
	return vkSendRequest(method, params, b.Token, b.Version)
}

func (b *VkBot) getUser(userID int)(vkUser, error) {
	var user vkUser

	params := map[string]string{
		"user_ids": string(userID),
		"name_case": "nom",
	}
	resp, err := b.sendRequest("users.get", params)
	if err != nil {
		return user, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return user, err
	}

	fmt.Println(string(body))

	err = json.Unmarshal(body, &user)
	if err != nil {
		return user, err
	}
	return user, nil
}

