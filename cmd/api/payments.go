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

type BillingCycle string

const (
	BillingCycleMonthly BillingCycle = "monthly"
	BillingCycleAnnual  BillingCycle = "annual"
)

type PaystackTransactionInitializationPayload struct {
	UserId       string       `json:"user_id"`
	PlanId       string       `json:"plan_id"`
	UserEmail    string       `json:"user_email"`
	BillingCycle BillingCycle `json:"billing_cycle"`
}

func (app *Config) PaystackTransactionInitialization(w http.ResponseWriter, r *http.Request) {

	// verify the user token
	user, err := app.getToken(r)
	if err != nil {
		app.errorJSON(w, err, user.Data, http.StatusUnauthorized)
		return
	}

	if user.Error {
		app.errorJSON(w, errors.New(user.Message), user.Data, user.StatusCode)
		return
	}

	//extract the request body
	var requestPayload PaystackTransactionInitializationPayload

	//extract the request body
	err = app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	userId := user.Data.(map[string]interface{})["user"].(map[string]interface{})["id"].(string)
	userEmail := user.Data.(map[string]interface{})["user"].(map[string]interface{})["email"].(string)
	requestPayload.UserId = userId
	requestPayload.UserEmail = userEmail

	//create some json we will send to authservice
	jsonData, _ := json.MarshalIndent(requestPayload, "", "\t")

	invServiceUrl := fmt.Sprintf("%s%s", os.Getenv("PAYMENT_SERVICE_URL"), "paystack-transaction-initialization")

	// call the service by creating a request
	request, err := http.NewRequest("POST", invServiceUrl, bytes.NewBuffer(jsonData))

	if err != nil {
		log.Println(err)
		app.errorJSON(w, err, nil)
		return
	}

	// Set the Content-Type header
	request.Header.Set("Content-Type", "application/json")
	//create a http client
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		log.Println(err)
		app.errorJSON(w, err, nil)
		return
	}
	defer response.Body.Close()

	// create a varabiel we'll read response.Body into
	var jsonFromService jsonResponse

	// decode the json from the auth service
	err = json.NewDecoder(response.Body).Decode(&jsonFromService)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	if response.StatusCode != http.StatusAccepted {
		app.errorJSON(w, errors.New(jsonFromService.Message), nil, response.StatusCode)
		return
	}

	var payload jsonResponse
	payload.Error = jsonFromService.Error
	payload.StatusCode = jsonFromService.StatusCode
	payload.Message = jsonFromService.Message
	payload.Data = jsonFromService.Data

	app.writeJSON(w, http.StatusOK, payload)
}

type CancelSubscriptionPayload struct {
	UserId string `json:"user_id"`
}

func (app *Config) CancelSubscription(w http.ResponseWriter, r *http.Request) {

	// verify the user token
	user, err := app.getToken(r)
	if err != nil {
		app.errorJSON(w, err, user.Data, http.StatusUnauthorized)
		return
	}

	if user.Error {
		app.errorJSON(w, errors.New(user.Message), user.Data, user.StatusCode)
		return
	}

	//extract the request body
	var requestPayload CancelSubscriptionPayload

	userId := user.Data.(map[string]interface{})["user"].(map[string]interface{})["id"].(string)
	requestPayload.UserId = userId

	//create some json we will send to authservice
	jsonData, _ := json.MarshalIndent(requestPayload, "", "\t")

	invServiceUrl := fmt.Sprintf("%s%s", os.Getenv("PAYMENT_SERVICE_URL"), "cancel-subscription")

	// call the service by creating a request
	request, err := http.NewRequest("POST", invServiceUrl, bytes.NewBuffer(jsonData))

	if err != nil {
		log.Println(err)
		app.errorJSON(w, err, nil)
		return
	}

	// Set the Content-Type header
	request.Header.Set("Content-Type", "application/json")
	//create a http client
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		log.Println(err)
		app.errorJSON(w, err, nil)
		return
	}
	defer response.Body.Close()

	// create a varabiel we'll read response.Body into
	var jsonFromService jsonResponse

	// decode the json from the auth service
	err = json.NewDecoder(response.Body).Decode(&jsonFromService)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	if response.StatusCode != http.StatusAccepted {
		app.errorJSON(w, errors.New(jsonFromService.Message), nil, response.StatusCode)
		return
	}

	var payload jsonResponse
	payload.Error = jsonFromService.Error
	payload.StatusCode = jsonFromService.StatusCode
	payload.Message = jsonFromService.Message
	payload.Data = jsonFromService.Data

	app.writeJSON(w, http.StatusOK, payload)
}

type SubscriptionHistoryPayload struct {
	UserId string `json:"user_id"`
}

func (app *Config) GetSubscriptionHistory(w http.ResponseWriter, r *http.Request) {

	// verify the user token
	user, err := app.getToken(r)
	if err != nil {
		app.errorJSON(w, err, user.Data, http.StatusUnauthorized)
		return
	}

	if user.Error {
		app.errorJSON(w, errors.New(user.Message), user.Data, user.StatusCode)
		return
	}

	//extract the request body
	var requestPayload SubscriptionHistoryPayload

	userId := user.Data.(map[string]interface{})["user"].(map[string]interface{})["id"].(string)
	requestPayload.UserId = userId

	//create some json we will send to authservice
	jsonData, _ := json.MarshalIndent(requestPayload, "", "\t")

	invServiceUrl := fmt.Sprintf("%s%s", os.Getenv("PAYMENT_SERVICE_URL"), "subscription-history")

	// call the service by creating a request
	request, err := http.NewRequest("POST", invServiceUrl, bytes.NewBuffer(jsonData))

	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	// Set the Content-Type header
	request.Header.Set("Content-Type", "application/json")
	//create a http client
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}
	defer response.Body.Close()

	// create a varabiel we'll read response.Body into
	var jsonFromService jsonResponse

	// decode the json from the auth service
	err = json.NewDecoder(response.Body).Decode(&jsonFromService)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	if response.StatusCode != http.StatusAccepted {
		app.errorJSON(w, errors.New(jsonFromService.Message), nil, response.StatusCode)
		return
	}

	var payload jsonResponse
	payload.Error = jsonFromService.Error
	payload.StatusCode = jsonFromService.StatusCode
	payload.Message = jsonFromService.Message
	payload.Data = jsonFromService.Data

	app.writeJSON(w, http.StatusOK, payload)
}

type VerifyPaystackTransactionPayload struct {
	Reference string `json:"reference"`
}

func (app *Config) VerifyPaystackTransaction(w http.ResponseWriter, r *http.Request) {

	// verify the user token
	user, err := app.getToken(r)
	if err != nil {
		app.errorJSON(w, err, user.Data, http.StatusUnauthorized)
		return
	}

	if user.Error {
		app.errorJSON(w, errors.New(user.Message), user.Data, user.StatusCode)
		return
	}

	//extract the request body
	var requestPayload VerifyPaystackTransactionPayload

	//extract the request body
	err = app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	//create some json we will send to authservice
	jsonData, _ := json.MarshalIndent(requestPayload, "", "\t")

	invServiceUrl := fmt.Sprintf("%s%s", os.Getenv("PAYMENT_SERVICE_URL"), "verify-paystack-transaction")

	// call the service by creating a request
	request, err := http.NewRequest("POST", invServiceUrl, bytes.NewBuffer(jsonData))

	if err != nil {
		log.Println(err)
		app.errorJSON(w, err, nil)
		return
	}

	// Set the Content-Type header
	request.Header.Set("Content-Type", "application/json")
	//create a http client
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		log.Println(err)
		app.errorJSON(w, err, nil)
		return
	}
	defer response.Body.Close()

	// create a varabiel we'll read response.Body into
	var jsonFromService jsonResponse

	// decode the json from the auth service
	err = json.NewDecoder(response.Body).Decode(&jsonFromService)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	if response.StatusCode != http.StatusAccepted {
		app.errorJSON(w, errors.New(jsonFromService.Message), nil, response.StatusCode)
		return
	}

	var payload jsonResponse
	payload.Error = jsonFromService.Error
	payload.StatusCode = jsonFromService.StatusCode
	payload.Message = jsonFromService.Message
	payload.Data = jsonFromService.Data

	app.writeJSON(w, http.StatusOK, payload)
}

func (app *Config) GetPlans(w http.ResponseWriter, r *http.Request) {

	invServiceUrl := fmt.Sprintf("%s%s", os.Getenv("PAYMENT_SERVICE_URL"), "plans")

	// call the service by creating a request
	request, err := http.NewRequest("GET", invServiceUrl, nil)

	if err != nil {
		log.Println(err)
		app.errorJSON(w, err, nil)
		return
	}

	// Set the Content-Type header
	request.Header.Set("Content-Type", "application/json")
	//create a http client
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		log.Println(err)
		app.errorJSON(w, err, nil)
		return
	}
	defer response.Body.Close()

	// create a varabiel we'll read response.Body into
	var jsonFromService jsonResponse

	// decode the json from the auth service
	err = json.NewDecoder(response.Body).Decode(&jsonFromService)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	if response.StatusCode != http.StatusAccepted {
		app.errorJSON(w, errors.New(jsonFromService.Message), nil, response.StatusCode)
		return
	}

	var payload jsonResponse
	payload.Error = jsonFromService.Error
	payload.StatusCode = jsonFromService.StatusCode
	payload.Message = jsonFromService.Message
	payload.Data = jsonFromService.Data

	app.writeJSON(w, http.StatusOK, payload)
}
