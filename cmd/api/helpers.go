package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
)

type jsonResponse struct {
	Error      bool   `json:"error"`
	Message    string `json:"message"`
	StatusCode int    `json:"status_code"`
	Data       any    `json:"data,omitempty"`
}

// read json
func (app *Config) readJSON(w http.ResponseWriter, r *http.Request, data any) error {
	//add a limiation on the uploaded json file
	maxByte := 104876
	//validate to make sure the request body is not more than 1 byte
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxByte))
	//decode the request body
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(data)
	if err != nil {
		return err
	}

	//check that there is only a single json value in the data we received
	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		return errors.New("body must have only a single json value")
	}

	return nil
}

// write json
func (app *Config) writeJSON(w http.ResponseWriter, status int, data any, headers ...http.Header) error {

	//converts the passed data into json representative
	out, err := json.Marshal(data)

	if err != nil {
		return err
	}

	//check if any header is supplied and set the respnse header
	if len(headers) > 0 {
		for key, value := range headers[0] {
			w.Header()[key] = value
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, err = w.Write(out)
	if err != nil {
		return err
	}

	return nil
}

//generate error json response

func (app *Config) errorJSON(w http.ResponseWriter, err error, data any, status ...int) error {
	statusCode := http.StatusBadRequest
	if len(status) > 0 {
		statusCode = status[0]
	}

	var payload jsonResponse
	payload.Error = true
	payload.Message = err.Error()
	payload.StatusCode = statusCode
	payload.Data = data

	return app.writeJSON(w, statusCode, payload)
}

func (app *Config) getToken(r *http.Request) (jsonResponse, error) {
	//get authorization hearder
	authorizationHeader := r.Header.Get("Authorization")

	// call the service by creating a request
	// request, err := http.NewRequest("GET", os.Getenv("FILE_UPLOAD_URL")+"upload", nil)
	// call the service by creating a request
	request, err := http.NewRequest("GET", os.Getenv("AUTH_URL")+"verify-token", nil)

	if err != nil {
		return jsonResponse{Error: true, Message: err.Error(), StatusCode: http.StatusBadRequest, Data: nil}, err

	}

	// Set the "Authorization" header with your Bearer token
	request.Header.Set("authorization", authorizationHeader)

	// Set the Content-Type header
	request.Header.Set("Content-Type", "application/json")
	//create a http client
	client := &http.Client{}
	response, err := client.Do(request)

	if err != nil {
		return jsonResponse{Error: true, Message: err.Error(), StatusCode: http.StatusBadRequest, Data: nil}, err

	}
	defer response.Body.Close()

	//variable to marshal into
	var jsonFromService jsonResponse

	err = json.NewDecoder(response.Body).Decode(&jsonFromService)

	log.Println(jsonFromService, "json from service")
	if err != nil {
		return jsonResponse{Error: true, Message: err.Error(), StatusCode: http.StatusBadRequest, Data: nil}, err
	}

	// make a call to the bank-service
	var payload jsonResponse
	payload.Error = jsonFromService.Error
	payload.Message = jsonFromService.Message
	payload.StatusCode = response.StatusCode
	payload.Data = jsonFromService.Data

	if jsonFromService.Error {
		return payload, err
	}

	return payload, nil
}
