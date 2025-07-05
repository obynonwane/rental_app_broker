package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
)

func (app *Config) UploadProfileImage(w http.ResponseWriter, r *http.Request) {

	// verify the user token
	response, err := app.getToken(r)
	if err != nil {
		app.errorJSON(w, err, response.Data, http.StatusUnauthorized)
		return
	}

	if response.Error {
		app.errorJSON(w, errors.New(response.Message), response.Data, response.StatusCode)
		return
	}

	// Extract user ID from response.Data
	var userID string
	if response.Data != nil {
		// Assert response.Data is a map
		dataMap, ok := response.Data.(map[string]any)
		if !ok {
			app.errorJSON(w, errors.New("invalid data format"), nil)
			return
		}

		// Extract "user" field and assert it is a map
		userData, ok := dataMap["user"].(map[string]any)
		if !ok {
			app.errorJSON(w, errors.New("missing or invalid user data"), nil)
			return
		}

		// Extract "id" field and assert it is a string
		userID, ok = userData["id"].(string)
		if !ok {
			app.errorJSON(w, errors.New("missing or invalid user ID"), nil)
			return
		}

	}

	err = r.ParseMultipartForm(20 << 20)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	// Retrieve the primary image from the form data
	primaryImageFile, fileHeader, err := r.FormFile("image")
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}
	defer primaryImageFile.Close()

	// Copy primary image data into a buffer
	var primaryImageDataBuffer bytes.Buffer
	_, err = io.Copy(&primaryImageDataBuffer, primaryImageFile)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	//======================FORWARD TO INVENTORY SERVICE==============================================
	// create a buffer and multipart writer for the outgoing request
	var b bytes.Buffer
	writer := multipart.NewWriter(&b)

	// Add the image field
	formFile, err := writer.CreateFormFile("image", fileHeader.Filename)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	_, err = io.Copy(formFile, bytes.NewReader(primaryImageDataBuffer.Bytes()))
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	// Add user ID field
	err = writer.WriteField("user_id", userID)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	// Close the multipart writer
	writer.Close()

	// create the request to inventory service
	// Construct inventory service URL
	invServiceUrl := fmt.Sprintf("%s%s", os.Getenv("INVENTORY_SERVICE_URL"), "profile-image")

	// Create POST request with JSON body
	req, err := http.NewRequest("POST", invServiceUrl, &b)
	if err != nil {
		log.Println(err)
		app.errorJSON(w, err, nil)
		return
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}
	defer resp.Body.Close()

	// Read and decode the response
	var jsonFromService jsonResponse
	err = json.NewDecoder(resp.Body).Decode(&jsonFromService)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	if resp.StatusCode != http.StatusAccepted {
		app.errorJSON(w, errors.New(jsonFromService.Message), nil, resp.StatusCode)
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

func (app *Config) UploadBanner(w http.ResponseWriter, r *http.Request) {

	// verify the user token
	response, err := app.getToken(r)
	if err != nil {
		app.errorJSON(w, err, response.Data, http.StatusUnauthorized)
		return
	}

	if response.Error {
		app.errorJSON(w, errors.New(response.Message), response.Data, response.StatusCode)
		return
	}

	// Extract user ID from response.Data
	var userID string
	if response.Data != nil {
		// Assert response.Data is a map
		dataMap, ok := response.Data.(map[string]any)
		if !ok {
			app.errorJSON(w, errors.New("invalid data format"), nil)
			return
		}

		// Extract "user" field and assert it is a map
		userData, ok := dataMap["user"].(map[string]any)
		if !ok {
			app.errorJSON(w, errors.New("missing or invalid user data"), nil)
			return
		}

		// Extract "id" field and assert it is a string
		userID, ok = userData["id"].(string)
		if !ok {
			app.errorJSON(w, errors.New("missing or invalid user ID"), nil)
			return
		}

	}

	err = r.ParseMultipartForm(20 << 20)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	// Retrieve the primary image from the form data
	primaryImageFile, fileHeader, err := r.FormFile("image")
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}
	defer primaryImageFile.Close()

	// Copy primary image data into a buffer
	var primaryImageDataBuffer bytes.Buffer
	_, err = io.Copy(&primaryImageDataBuffer, primaryImageFile)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	//======================FORWARD TO INVENTORY SERVICE==============================================
	// create a buffer and multipart writer for the outgoing request
	var b bytes.Buffer
	writer := multipart.NewWriter(&b)

	// Add the image field
	formFile, err := writer.CreateFormFile("image", fileHeader.Filename)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	_, err = io.Copy(formFile, bytes.NewReader(primaryImageDataBuffer.Bytes()))
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	// Add user ID field
	err = writer.WriteField("user_id", userID)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	// Close the multipart writer
	writer.Close()

	// create the request to inventory service
	// Construct inventory service URL
	invServiceUrl := fmt.Sprintf("%s%s", os.Getenv("INVENTORY_SERVICE_URL"), "shop-banner")

	// Create POST request with JSON body
	req, err := http.NewRequest("POST", invServiceUrl, &b)
	if err != nil {
		log.Println(err)
		app.errorJSON(w, err, nil)
		return
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}
	defer resp.Body.Close()

	// Read and decode the response
	var jsonFromService jsonResponse
	err = json.NewDecoder(resp.Body).Decode(&jsonFromService)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	if resp.StatusCode != http.StatusAccepted {
		app.errorJSON(w, errors.New(jsonFromService.Message), nil, resp.StatusCode)
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
