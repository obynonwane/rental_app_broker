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
