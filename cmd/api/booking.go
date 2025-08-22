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

type CreateBookingPayload struct {
	InventoryId       string  `json:"inventory_id" binding:"required"`
	RenterId          string  `json:"renter_id"`
	OwnerId           string  `json:"owner_id"`
	RentalType        string  `json:"rental_type" binding:"required"`     // e.g., "hourly", "daily"
	RentalDuration    float64 `json:"rental_duration" binding:"required"` // number of units (hours, days, etc.)
	SecurityDeposit   float64 `json:"security_deposit"`                   // can be zero
	OfferPricePerUnit float64 `json:"offer_price_per_unit" binding:"required"`
	Quantity          float64 `json:"quantity" binding:"required"`

	StartDate   string  `json:"start_date" binding:"required"` // e.g., "2025-06-15"
	EndDate     string  `json:"end_date" binding:"required"`   // e.g., "2025-06-15"
	EndTime     string  `json:"end_time" binding:"required"`   // e.g., "18:00:00", optional for daily+ rentals
	StartTime   string  `json:"start_time" binding:"required"` // e.g., "18:00:00", optional for daily+ rentals
	TotalAmount float64 `json:"total_amount" binding:"required"`
}

func (app *Config) CreateBooking(w http.ResponseWriter, r *http.Request) {

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
	var requestPayload CreateBookingPayload

	//extract the request body
	err = app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	log.Println(requestPayload, "THE payload")

	// Validate the request payload
	if err := app.ValidateBookingInput(requestPayload); len(err) > 0 {
		app.errorJSON(w, errors.New("error trying to create booking"), err, http.StatusBadRequest)
		return
	}

	userId := user.Data.(map[string]interface{})["user"].(map[string]interface{})["id"].(string)
	requestPayload.RenterId = userId

	//create some json we will send to authservice
	jsonData, _ := json.MarshalIndent(requestPayload, "", "\t")

	invServiceUrl := fmt.Sprintf("%s%s", os.Getenv("INVENTORY_SERVICE_URL"), "create-booking")

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

type MyBookingPayload struct {
	UserId string `json:"user_id"`
	Page   int32  `json:"page"`
	Limit  int32  `json:"limit"`
}

func (app *Config) MyBookings(w http.ResponseWriter, r *http.Request) {

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
	var requestPayload = MyBookingPayload{
		UserId: userId,
		Page:   int32(page),
		Limit:  int32(limit),
	}

	

	//create some json we will send to authservice
	jsonData, _ := json.MarshalIndent(requestPayload, "", "\t")

	invServiceUrl := fmt.Sprintf("%s%s", os.Getenv("INVENTORY_SERVICE_URL"), "my-booking")


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
