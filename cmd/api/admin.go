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

	"github.com/go-chi/chi/v5"
)

type AdminPendingInventoryPayload struct {
	Page  int32 `json:"page"`
	Limit int32 `json:"limit"`
}

func hasRole(roles []interface{}, target string) bool {
	for _, r := range roles {
		if roleStr, ok := r.(string); ok && roleStr == target {
			return true
		}
	}
	return false
}

func (app *Config) AdminGetInventoryPendingApproval(w http.ResponseWriter, r *http.Request) {

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

	// get the user role
	roles, ok := user.Data.(map[string]interface{})["roles"].([]interface{})
	if !ok {
		log.Println("roles is not a slice")
		return
	}

	roleExist := hasRole(roles, "admin")
	if !roleExist {
		app.errorJSON(w, errors.New("user not an admin action denied"), nil, http.StatusUnauthorized)
		return
	}

	//extract the request body
	var requestPayload = AdminPendingInventoryPayload{
		Page:  int32(page),
		Limit: int32(limit),
	}

	//create some json we will send to authservice
	jsonData, _ := json.MarshalIndent(requestPayload, "", "\t")

	invServiceUrl := fmt.Sprintf("%s%s", os.Getenv("INVENTORY_SERVICE_URL"), "pending-inventories")

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

	// create a variable we'll read response.Body into
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

func (app *Config) AdminApproveInventory(w http.ResponseWriter, r *http.Request) {

	id := chi.URLParam(r, "id")

	if id == "" {
		app.errorJSON(w, errors.New("id parameter is missing"), nil)
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

	// get the user role
	roles, ok := user.Data.(map[string]interface{})["roles"].([]interface{})
	if !ok {
		log.Println("roles is not a slice")
		return
	}

	roleExist := hasRole(roles, "admin")
	if !roleExist {
		app.errorJSON(w, errors.New("user not an admin action denied"), nil, http.StatusUnauthorized)
		return
	}

	invServiceUrl := fmt.Sprintf("%s%s", os.Getenv("INVENTORY_SERVICE_URL"), "approve-inventory/"+id)

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

type AdminGetActiveSubscriptionPayload struct {
	Page  int32 `json:"page"`
	Limit int32 `json:"limit"`
}

func (app *Config) AdminGetActiveSubscriptions(w http.ResponseWriter, r *http.Request) {

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
		log.Println(err, "1")
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

	// get the user role
	roles, ok := user.Data.(map[string]interface{})["roles"].([]interface{})
	if !ok {
		log.Println("roles is not a slice")
		return
	}

	roleExist := hasRole(roles, "admin")
	if !roleExist {
		app.errorJSON(w, errors.New("user not an admin action denied"), nil, http.StatusUnauthorized)
		return
	}

	//extract the request body
	var requestPayload = AdminGetActiveSubscriptionPayload{
		Page:  int32(page),
		Limit: int32(limit),
	}

	//create some json we will send to authservice
	jsonData, _ := json.MarshalIndent(requestPayload, "", "\t")

	invServiceUrl := fmt.Sprintf("%s%s", os.Getenv("INVENTORY_SERVICE_URL"), "active-subscriptions")

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

type AdminGetUsersPayload struct {
	Page  int32 `json:"page"`
	Limit int32 `json:"limit"`
}

func (app *Config) AdminGetUsers(w http.ResponseWriter, r *http.Request) {

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
		log.Println(err, "1")
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

	// get the user role
	roles, ok := user.Data.(map[string]interface{})["roles"].([]interface{})
	if !ok {
		log.Println("roles is not a slice")
		return
	}

	roleExist := hasRole(roles, "admin")
	if !roleExist {
		app.errorJSON(w, errors.New("user not an admin action denied"), nil, http.StatusUnauthorized)
		return
	}

	//extract the request body
	var requestPayload = AdminGetUsersPayload{
		Page:  int32(page),
		Limit: int32(limit),
	}

	app.proceedGetUser(w, requestPayload)
}

func (app *Config) proceedGetUser(w http.ResponseWriter, requestPayload AdminGetUsersPayload) {

	authServiceUrl := fmt.Sprintf("%s%s", os.Getenv("INVENTORY_SERVICE_URL"), "getusers")

	//create some json we will send to authservice
	jsonData, _ := json.MarshalIndent(requestPayload, "", "\t")
	// Call the service by creating a request
	request, err := http.NewRequest("POST", authServiceUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	// Set the Content-Type header
	request.Header.Set("Content-Type", "application/json")

	// Create an HTTP client
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		log.Println(err)
		app.errorJSON(w, err, nil)
		return
	}
	defer response.Body.Close()

	// Create a variable to read response.Body into
	var jsonFromService jsonResponse

	// Decode the JSON from the service
	err = json.NewDecoder(response.Body).Decode(&jsonFromService)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	// Check if the status code is Accepted
	if response.StatusCode != http.StatusAccepted {
		app.errorJSON(w, errors.New("unexpected status code received from service"), nil, response.StatusCode)
		return
	}

	// Prepare the payload
	var payload jsonResponse
	payload.Error = jsonFromService.Error
	payload.StatusCode = http.StatusOK
	payload.Message = jsonFromService.Message
	payload.Data = jsonFromService.Data

	// Write the JSON response
	app.writeJSON(w, http.StatusOK, payload)
}

func (app *Config) AdminGetDashboardCard(w http.ResponseWriter, r *http.Request) {

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

	// get the user role
	roles, ok := user.Data.(map[string]interface{})["roles"].([]interface{})
	if !ok {
		log.Println("roles is not a slice")
		return
	}

	roleExist := hasRole(roles, "admin")
	if !roleExist {
		app.errorJSON(w, errors.New("user not an admin action denied"), nil, http.StatusUnauthorized)
		return
	}

	invServiceUrl := fmt.Sprintf("%s%s", os.Getenv("INVENTORY_SERVICE_URL"), "dasboard-card")

	// call the service by creating a request
	request, err := http.NewRequest("GET", invServiceUrl, nil)

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

func (app *Config) AdminGetAmountMadeByDate(w http.ResponseWriter, r *http.Request) {

	date := chi.URLParam(r, "date")

	if date == "" {
		app.errorJSON(w, errors.New("date parameter is missing"), nil)
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

	// get the user role
	roles, ok := user.Data.(map[string]interface{})["roles"].([]interface{})
	if !ok {
		log.Println("roles is not a slice")
		return
	}

	roleExist := hasRole(roles, "admin")
	if !roleExist {
		app.errorJSON(w, errors.New("user not an admin action denied"), nil, http.StatusUnauthorized)
		return
	}

	invServiceUrl := fmt.Sprintf("%s%s", os.Getenv("INVENTORY_SERVICE_URL"), "amount-made-bydate/"+date)

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

func (app *Config) AdminGetUsersJoinedByDate(w http.ResponseWriter, r *http.Request) {

	date := chi.URLParam(r, "date")

	if date == "" {
		app.errorJSON(w, errors.New("date parameter is missing"), nil)
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

	// get the user role
	roles, ok := user.Data.(map[string]interface{})["roles"].([]interface{})
	if !ok {
		log.Println("roles is not a slice")
		return
	}

	roleExist := hasRole(roles, "admin")
	if !roleExist {
		app.errorJSON(w, errors.New("user not an admin action denied"), nil, http.StatusUnauthorized)
		return
	}

	invServiceUrl := fmt.Sprintf("%s%s", os.Getenv("INVENTORY_SERVICE_URL"), "users-joined-bydate/"+date)

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

func (app *Config) AdminGetInventoryCreatedByDate(w http.ResponseWriter, r *http.Request) {

	date := chi.URLParam(r, "date")

	if date == "" {
		app.errorJSON(w, errors.New("date parameter is missing"), nil)
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

	// get the user role
	roles, ok := user.Data.(map[string]interface{})["roles"].([]interface{})
	if !ok {
		log.Println("roles is not a slice")
		return
	}

	roleExist := hasRole(roles, "admin")
	if !roleExist {
		app.errorJSON(w, errors.New("user not an admin action denied"), nil, http.StatusUnauthorized)
		return
	}

	invServiceUrl := fmt.Sprintf("%s%s", os.Getenv("INVENTORY_SERVICE_URL"), "inventory-created-bydate/"+date)

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

type RegistrationStatsRequest struct {
	GroupBy   string // "day", "month", or "year"
	StartDate string // e.g. "2023-01-01"
	EndDate   string // e.g. "2023-12-31"
}

func (app *Config) GetUserRegistrationStats(w http.ResponseWriter, r *http.Request) {

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

	// get the user role
	roles, ok := user.Data.(map[string]interface{})["roles"].([]interface{})
	if !ok {
		log.Println("roles is not a slice")
		return
	}

	roleExist := hasRole(roles, "admin")
	if !roleExist {
		app.errorJSON(w, errors.New("user not an admin action denied"), nil, http.StatusUnauthorized)
		return
	}

	//extract the request body
	var requestPayload RegistrationStatsRequest

	//extract the requestbody
	err = app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	//create some json we will send to authservice
	jsonData, _ := json.MarshalIndent(requestPayload, "", "\t")

	invServiceUrl := fmt.Sprintf("%s%s", os.Getenv("INVENTORY_SERVICE_URL"), "analytics/user-registrations")

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

func (app *Config) GetInventoryCreationStats(w http.ResponseWriter, r *http.Request) {

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

	// get the user role
	roles, ok := user.Data.(map[string]interface{})["roles"].([]interface{})
	if !ok {
		log.Println("roles is not a slice")
		return
	}

	roleExist := hasRole(roles, "admin")
	if !roleExist {
		app.errorJSON(w, errors.New("user not an admin action denied"), nil, http.StatusUnauthorized)
		return
	}

	//extract the request body
	var requestPayload RegistrationStatsRequest

	//extract the requestbody
	err = app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	//create some json we will send to authservice
	jsonData, _ := json.MarshalIndent(requestPayload, "", "\t")

	invServiceUrl := fmt.Sprintf("%s%s", os.Getenv("INVENTORY_SERVICE_URL"), "analytics/inventory-creations")

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

type SubscriptionStatsRequest struct {
	GroupBy   string // "day", "month", or "year"
	StartDate string // e.g., "2025-01-01"
	EndDate   string // e.g., "2025-12-31"
}

func (app *Config) GetSubscriptionAmountStats(w http.ResponseWriter, r *http.Request) {

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

	// get the user role
	roles, ok := user.Data.(map[string]interface{})["roles"].([]interface{})
	if !ok {
		log.Println("roles is not a slice")
		return
	}

	roleExist := hasRole(roles, "admin")
	if !roleExist {
		app.errorJSON(w, errors.New("user not an admin action denied"), nil, http.StatusUnauthorized)
		return
	}

	//extract the request body
	var requestPayload SubscriptionStatsRequest

	//extract the requestbody
	err = app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	//create some json we will send to authservice
	jsonData, _ := json.MarshalIndent(requestPayload, "", "\t")

	invServiceUrl := fmt.Sprintf("%s%s", os.Getenv("INVENTORY_SERVICE_URL"), "analytics/subscription-amount")

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
