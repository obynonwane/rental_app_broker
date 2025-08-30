package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
)

type CreatePrurchaseOrderPayload struct {
	InventoryId       string  `json:"inventory_id" binding:"required"`
	SellerId          string  `json:"seller_id"`
	BuyerId           string  `json:"buyer_id"`
	OfferPricePerUnit float64 `json:"offer_price_per_unit" binding:"required"`
	Quantity          float64 `json:"quantity" binding:"required"`
	TotalAmount       float64 `json:"total_amount" binding:"required"`
}

func (app *Config) CreatePrurchaseOrder(w http.ResponseWriter, r *http.Request) {

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
	var requestPayload CreatePrurchaseOrderPayload

	//extract the requestbody
	err = app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	// Validate the request payload
	if err := app.ValidatePuchaseOrderInput(requestPayload); len(err) > 0 {
		app.errorJSON(w, errors.New("error trying to create purchase order"), err, http.StatusBadRequest)
		return
	}

	userId := user.Data.(map[string]interface{})["user"].(map[string]interface{})["id"].(string)
	requestPayload.BuyerId = userId

	//create some json we will send to authservice
	jsonData, _ := json.MarshalIndent(requestPayload, "", "\t")

	invServiceUrl := fmt.Sprintf("%s%s", os.Getenv("INVENTORY_SERVICE_URL"), "create-order")

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

type MyPurchasePayload struct {
	UserId string `json:"user_id"`
	Page   int32  `json:"page"`
	Limit  int32  `json:"limit"`
}

func (app *Config) MyPurchase(w http.ResponseWriter, r *http.Request) {

	// 2. retrieve query param
	queryParams := r.URL.Query()
	pageStr := queryParams.Get("page")
	if pageStr == "" {

		app.errorJSON(w, errors.New("page not supplied"), nil)
		return
	}
	limitStr := queryParams.Get("limit")
	if limitStr == "" {
		app.errorJSON(w, errors.New("limit not supplied"), nil)
		return
	}

	// convert to int32
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		app.errorJSON(w, errors.New("invalid page number"), nil)
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		app.errorJSON(w, errors.New("invalid limit number"), nil)
		return
	}

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

	userId := user.Data.(map[string]interface{})["user"].(map[string]interface{})["id"].(string)

	//extract the request body
	var requestPayload = MyPurchasePayload{
		UserId: userId,
		Page:   int32(page),
		Limit:  int32(limit),
	}

	//create some json we will send to authservice
	jsonData, _ := json.MarshalIndent(requestPayload, "", "\t")

	invServiceUrl := fmt.Sprintf("%s%s", os.Getenv("INVENTORY_SERVICE_URL"), "my-purchase")

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

func (app *Config) GetPurchaseRequest(w http.ResponseWriter, r *http.Request) {

	// 2. retrieve query param
	queryParams := r.URL.Query()
	pageStr := queryParams.Get("page")
	if pageStr == "" {

		app.errorJSON(w, errors.New("page not supplied"), nil)
		return
	}
	limitStr := queryParams.Get("limit")
	if limitStr == "" {
		app.errorJSON(w, errors.New("limit not supplied"), nil)
		return
	}

	// convert to int32
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		app.errorJSON(w, errors.New("invalid page number"), nil)
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		app.errorJSON(w, errors.New("invalid limit number"), nil)
		return
	}

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

	userId := user.Data.(map[string]interface{})["user"].(map[string]interface{})["id"].(string)

	//extract the request body
	var requestPayload = MyPurchasePayload{
		UserId: userId,
		Page:   int32(page),
		Limit:  int32(limit),
	}

	//create some json we will send to authservice
	jsonData, _ := json.MarshalIndent(requestPayload, "", "\t")

	invServiceUrl := fmt.Sprintf("%s%s", os.Getenv("INVENTORY_SERVICE_URL"), "purchase-requests")

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

type MySubscriptionHistoryPayload struct {
	UserId string `json:"user_id"`
	Page   int32  `json:"page"`
	Limit  int32  `json:"limit"`
}

func (app *Config) MySubscriptionHistory(w http.ResponseWriter, r *http.Request) {

	// 2. retrieve query param
	queryParams := r.URL.Query()
	pageStr := queryParams.Get("page")
	if pageStr == "" {

		app.errorJSON(w, errors.New("page not supplied"), nil)
		return
	}
	limitStr := queryParams.Get("limit")
	if limitStr == "" {
		app.errorJSON(w, errors.New("limit not supplied"), nil)
		return
	}

	// convert to int32
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		app.errorJSON(w, errors.New("invalid page number"), nil)
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		app.errorJSON(w, errors.New("invalid limit number"), nil)
		return
	}

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

	userId := user.Data.(map[string]interface{})["user"].(map[string]interface{})["id"].(string)

	//extract the request body
	var requestPayload = MySubscriptionHistoryPayload{
		UserId: userId,
		Page:   int32(page),
		Limit:  int32(limit),
	}

	//create some json we will send to authservice
	jsonData, _ := json.MarshalIndent(requestPayload, "", "\t")

	invServiceUrl := fmt.Sprintf("%s%s", os.Getenv("INVENTORY_SERVICE_URL"), "my-subscription-history")

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

func (app *Config) GetPendingPurchaseCount(w http.ResponseWriter, r *http.Request) {
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

	userId := user.Data.(map[string]interface{})["user"].(map[string]interface{})["id"].(string)

	// Build the full URL with query param
	baseUrl := os.Getenv("INVENTORY_SERVICE_URL") + "pending-purchase-count"
	reqUrl := fmt.Sprintf("%s?userId=%s", baseUrl, userId)

	// Create the GET request with the query param
	request, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	// Set headers
	request.Header.Set("Content-Type", "application/json")

	// Execute the request
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}
	defer response.Body.Close()

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

	payload := jsonResponse{
		Error:      jsonFromService.Error,
		StatusCode: jsonFromService.StatusCode,
		Message:    jsonFromService.Message,
		Data:       jsonFromService.Data,
	}

	app.writeJSON(w, http.StatusOK, payload)
}
