package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
)

type SearchPremiumPartnerPayload struct {
	Text     string `json:"text"`
	Industry string `json:"industry"`
	Limit    string `json:"limit"`
	Offset   string `json:"offset"`
}

func (app *Config) PremiumPartner(w http.ResponseWriter, r *http.Request) {

	var requestPayload SearchPremiumPartnerPayload

	//2. extract the requestbody
	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	//create some json we will send to authservice
	jsonData, _ := json.MarshalIndent(requestPayload, "", "\t")

	// Construct inventory service URL
	invServiceUrl := fmt.Sprintf("%s%s", os.Getenv("INVENTORY_SERVICE_URL"), "premium-partners")

	// Create POST request with JSON body
	request, err := http.NewRequest("POST", invServiceUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Println(err)
		app.errorJSON(w, err, nil)
		return
	}

	// Set Content-Type header
	request.Header.Set("Content-Type", "application/json")

	// Execute the HTTP request
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		log.Println(err)
		app.errorJSON(w, err, nil)
		return
	}
	defer response.Body.Close()

	// Read and decode the response
	var jsonFromService jsonResponse
	err = json.NewDecoder(response.Body).Decode(&jsonFromService)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	if response.StatusCode != http.StatusAccepted {
		app.errorJSON(w, errors.New(jsonFromService.Message), nil, response.StatusCode)
		return
	}

	// Relay the response
	payload := jsonResponse{
		Error:      jsonFromService.Error,
		StatusCode: jsonFromService.StatusCode,
		Message:    jsonFromService.Message,
		Data:       jsonFromService.Data,
	}
	app.writeJSON(w, http.StatusOK, payload)
}
func (app *Config) GetPremiumUsersExtras(w http.ResponseWriter, r *http.Request) {

	// Construct inventory service URL
	invServiceUrl := fmt.Sprintf("%s%s", os.Getenv("INVENTORY_SERVICE_URL"), "premium-extras")

	// Create POST request with JSON body
	request, err := http.NewRequest("GET", invServiceUrl, nil)
	if err != nil {
		log.Println(err)
		app.errorJSON(w, err, nil)
		return
	}

	// Set Content-Type header
	request.Header.Set("Content-Type", "application/json")

	// Execute the HTTP request
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		log.Println(err)
		app.errorJSON(w, err, nil)
		return
	}
	defer response.Body.Close()

	// Read and decode the response
	var jsonFromService jsonResponse
	err = json.NewDecoder(response.Body).Decode(&jsonFromService)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	if response.StatusCode != http.StatusAccepted {
		app.errorJSON(w, errors.New(jsonFromService.Message), nil, response.StatusCode)
		return
	}

	// Relay the response
	payload := jsonResponse{
		Error:      jsonFromService.Error,
		StatusCode: jsonFromService.StatusCode,
		Message:    jsonFromService.Message,
		Data:       jsonFromService.Data,
	}
	app.writeJSON(w, http.StatusOK, payload)
}
