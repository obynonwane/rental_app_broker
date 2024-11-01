package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"reflect"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-redis/redis"
)

var ctx = context.Background()

type MailPayload struct {
	From    string                 `json:"from"`
	To      string                 `json:"to"`
	Subject string                 `json:"subject"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data"`
}

type SignupPayload struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	Password  string `json:"password"`
}

type AssignPermissionPayload struct {
	UserID       string `json:"user_id"`
	PermissionID string `json:"permission_id"`
}

type ChooseRolePayload struct {
	UserType string `json:"user_type"`
}

type LoginPayload struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

func (app *Config) Signup(w http.ResponseWriter, r *http.Request) {

	//extract the request body
	var requestPayload SignupPayload

	//extract the requestbody
	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	// Validate the request payload
	if err := app.ValidataSignupInput(requestPayload); len(err) > 0 {
		app.errorJSON(w, errors.New("error trying to sign-up user"), err, http.StatusBadRequest)
		return
	}

	//create some json we will send to authservice
	jsonData, _ := json.MarshalIndent(requestPayload, "", "\t")

	authServiceUrl := fmt.Sprintf("%s%s", os.Getenv("AUTH_URL"), "signup")

	// call the service by creating a request
	request, err := http.NewRequest("POST", authServiceUrl, bytes.NewBuffer(jsonData))

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

	log.Println("response from auth service", jsonFromService)
	if response.StatusCode != http.StatusAccepted {
		log.Println(jsonFromService.Message, jsonFromService)
		app.errorJSON(w, errors.New(jsonFromService.Message), nil)
		return
	}

	var payload jsonResponse
	payload.Error = jsonFromService.Error
	payload.StatusCode = http.StatusOK
	payload.Message = jsonFromService.Message
	payload.Data = jsonFromService.Data

	app.writeJSON(w, http.StatusOK, payload)
}

func (app *Config) Login(w http.ResponseWriter, r *http.Request) {
	log.Println("hit the login endpoint")

	TrackFunctionCall("Login", func() {

		// Extract the request body
		var requestPayload LoginPayload
		err := app.readJSON(w, r, &requestPayload)
		if err != nil {
			app.errorJSON(w, err, nil)
			return
		}

		// Validate the request payload
		if err := app.ValidateLoginInput(requestPayload); len(err) > 0 {
			app.errorJSON(w, errors.New("error trying to sign-in user"), err, http.StatusBadRequest)
			return
		}

		value, err := app.cache.Get(r.Context(), "login_detail").Result()
		if err != nil {
			if errors.Is(err, redis.Nil) || err.Error() == "redis: nil" {
				log.Println("This is a cache miss : logging in")
				//create some json we will send to authservice
				jsonData, _ := json.MarshalIndent(requestPayload, "", "\t")

				authServiceUrl := fmt.Sprintf("%s%s", os.Getenv("AUTH_URL"), "login")

				// call the service by creating a request
				request, err := http.NewRequest("POST", authServiceUrl, bytes.NewBuffer(jsonData))

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
					app.errorJSON(w, errors.New("error signingup user"), nil)
					return
				}

				var payload jsonResponse
				payload.Error = jsonFromService.Error
				payload.StatusCode = http.StatusOK
				payload.Message = jsonFromService.Message
				payload.Data = jsonFromService.Data

				//convert the payload into string
				b, err := json.Marshal(payload)
				if err != nil {
					app.errorJSON(w, errors.New("error marshalling payload into string for saving to redis"), nil)
					return
				}
				//set the value in redis
				err = app.cache.Set(ctx, "login_detail", bytes.NewBuffer(b).Bytes(), time.Second*15).Err()
				if err != nil {
					fmt.Printf("error setting data & key to redis cache: %v\n", err)
				}
				app.writeJSON(w, http.StatusOK, payload)
				return
			}

		} else {

			var data jsonResponse
			err := json.Unmarshal(bytes.NewBufferString(value).Bytes(), &data)
			if err != nil {
				app.errorJSON(w, errors.New("error unmarshalling data from redis"), nil)
			}

			log.Println("This is a cache hit : logging in")
			app.writeJSON(w, http.StatusOK, data)
			return
		}
		app.errorJSON(w, err, nil)

	})

}

func (app *Config) GetMe(w http.ResponseWriter, r *http.Request) {

	value, err := app.cache.Get(r.Context(), "getme_detail").Result()
	if err != nil {
		if errors.Is(err, redis.Nil) || err.Error() == "redis: nil" {
			log.Println("This is a cache miss : getting getme detail")

			//get authorization hearder
			authorizationHeader := r.Header.Get("Authorization")

			authServiceUrl := fmt.Sprintf("%s%s", os.Getenv("AUTH_URL"), "get-me")

			// call the service by creating a request
			request, err := http.NewRequest("GET", authServiceUrl, nil)

			// Set the "Authorization" header with your Bearer token
			request.Header.Set("authorization", authorizationHeader)

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
				app.errorJSON(w, errors.New("error signingup user"), nil)
				return
			}

			var payload jsonResponse
			payload.Error = jsonFromService.Error
			payload.StatusCode = http.StatusOK
			payload.Message = jsonFromService.Message
			payload.Data = jsonFromService.Data

			//convert the payload into string
			b, err := json.Marshal(payload)
			if err != nil {
				app.errorJSON(w, errors.New("error marshalling payload into string for saving to redis"), nil)
				return
			}
			//set the value in redis
			err = app.cache.Set(ctx, "getme_detail", bytes.NewBuffer(b).Bytes(), time.Second*15).Err()
			if err != nil {
				fmt.Printf("error setting data & key to redis cache: %v\n", err)
			}
			app.writeJSON(w, http.StatusOK, payload)
			return
		}

	} else {

		var data jsonResponse
		err := json.Unmarshal(bytes.NewBufferString(value).Bytes(), &data)
		if err != nil {
			app.errorJSON(w, errors.New("error unmarshalling data from redis"), nil)
		}

		log.Println("This is a cache hit : getting getme detail")
		app.writeJSON(w, http.StatusOK, data)
		return
	}
	app.errorJSON(w, err, nil)

}

func (app *Config) VerifyToken(w http.ResponseWriter, r *http.Request) {
	value, err := app.cache.Get(r.Context(), "verify_token_detail").Result()

	if err != nil {
		if errors.Is(err, redis.Nil) || err.Error() == "redis: nil" {
			log.Println("This is a cache miss : getting verify_token_detail")
			//get authorization hearder
			authorizationHeader := r.Header.Get("Authorization")

			authServiceUrl := fmt.Sprintf("%s%s", os.Getenv("AUTH_URL"), "verify-token")

			// call the service by creating a request
			request, err := http.NewRequest("GET", authServiceUrl, nil)

			// Set the "Authorization" header with your Bearer token
			request.Header.Set("authorization", authorizationHeader)

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
				app.errorJSON(w, errors.New("error verifying token"), nil)
				return
			}

			var payload jsonResponse
			payload.Error = jsonFromService.Error
			payload.StatusCode = http.StatusOK
			payload.Message = jsonFromService.Message
			payload.Data = jsonFromService.Data

			//convert the payload into string
			b, err := json.Marshal(payload)
			if err != nil {
				app.errorJSON(w, errors.New("error marshalling payload into string for saving to redis"), nil)
				return
			}

			//set the value in redis
			err = app.cache.Set(ctx, "verify_token_detail", bytes.NewBuffer(b).Bytes(), time.Second*15).Err()
			if err != nil {
				fmt.Printf("error setting data & key to redis cache: %v\n", err)
			}
			app.writeJSON(w, http.StatusOK, payload)
			return
		}
	} else {

		var data jsonResponse
		err := json.Unmarshal(bytes.NewBufferString(value).Bytes(), &data)
		if err != nil {
			app.errorJSON(w, errors.New("error unmarshalling data from redis"), nil)
		}

		log.Println("This is a cache hit : getting verify_token_detail ")
		app.writeJSON(w, http.StatusOK, data)
		return
	}
	app.errorJSON(w, err, nil)
}

func (app *Config) Logout(w http.ResponseWriter, r *http.Request) {

	// retrieve authorization token
	authorizationHeader := r.Header.Get("Authorization")

	// contruct the url
	authServiceUrl := fmt.Sprintf("%s%s", os.Getenv("AUTH_URL"), "log-out")

	// call the service by creating a request
	request, err := http.NewRequest("POST", authServiceUrl, nil)

	// Set the "Authorization" header with your Bearer token
	request.Header.Set("authorization", authorizationHeader)

	// check for error
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
		app.errorJSON(w, errors.New("error logging out"), nil)
		return
	}

	var payload jsonResponse
	payload.Error = jsonFromService.Error
	payload.StatusCode = http.StatusOK
	payload.Message = jsonFromService.Message
	payload.Data = jsonFromService.Data

	app.writeJSON(w, http.StatusOK, payload)

}

func (app *Config) VerifyEmail(w http.ResponseWriter, r *http.Request) {

	token := r.FormValue("token")

	// contruct the url
	authServiceUrl := fmt.Sprintf("%sverify-email?token=%s", os.Getenv("AUTH_URL"), token)

	// call the service by creating a request
	request, err := http.NewRequest("GET", authServiceUrl, nil)

	// check for error
	if err != nil {
		log.Println(err, "1")
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
		app.errorJSON(w, errors.New("error verifying email"), nil)
		return
	}

	var payload jsonResponse
	payload.Error = jsonFromService.Error
	payload.StatusCode = http.StatusOK
	payload.Message = jsonFromService.Message
	payload.Data = jsonFromService.Data

	app.writeJSON(w, http.StatusOK, payload)
}

func (app *Config) Subscription(w http.ResponseWriter, r *http.Request) {
	log.Println("hit the subscription endpoint")
	payload := jsonResponse{
		Error:   false,
		Message: "hit the broker change",
		Data:    nil,
	}

	_ = app.writeJSON(w, http.StatusOK, payload)
}

func (app *Config) TestEmail(w http.ResponseWriter, r *http.Request) {

	//extract the requestbody
	var requestPayload MailPayload

	//extract the requestbody
	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	//create some json we will send to authservice
	jsonData, _ := json.MarshalIndent(requestPayload, "", "\t")

	// call the service by creating a request
	request, err := http.NewRequest("POST", os.Getenv("MAIL_URL"), bytes.NewBuffer(jsonData))

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

	if response.StatusCode != http.StatusAccepted {
		app.errorJSON(w, errors.New("error sending mail"), nil)
		return
	}

	var payload jsonResponse
	payload.Error = false
	payload.Message = "mail sent succesfully"

	app.writeJSON(w, http.StatusOK, payload)
}

func (app *Config) GetUsers(w http.ResponseWriter, r *http.Request) {

	response, err := app.getToken(r)
	if err != nil {
		app.errorJSON(w, err, response.Data, http.StatusUnauthorized)
		return
	}

	if response.Error {
		app.errorJSON(w, errors.New(response.Message), response.Data, response.StatusCode)
		return
	}

	app.proceedGetUser(w)
}

func (app *Config) proceedGetUser(w http.ResponseWriter) {

	authServiceUrl := fmt.Sprintf("%s%s", os.Getenv("INVENTORY_SERVICE_URL"), "getusers")
	log.Println("The endpoint:", authServiceUrl)

	// Call the service by creating a request
	request, err := http.NewRequest("GET", authServiceUrl, nil)
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
		app.errorJSON(w, errors.New("unexpected status code received from service"), nil)
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

func (app *Config) ChooseRole(w http.ResponseWriter, r *http.Request) {
	//extract the request body
	var requestPayload ChooseRolePayload

	//extract the requestbody
	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	// retrieve authorization token
	authorizationHeader := r.Header.Get("Authorization")

	// contruct the url
	authServiceUrl := fmt.Sprintf("%s%s", os.Getenv("AUTH_URL"), "choose-role")

	// create some json we will send to authservice
	jsonData, _ := json.MarshalIndent(requestPayload, "", "\t")

	// call the service by creating a request
	request, err := http.NewRequest("POST", authServiceUrl, bytes.NewBuffer(jsonData))

	// Set the "Authorization" header with your Bearer token
	request.Header.Set("authorization", authorizationHeader)

	// check for error
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
		log.Println(err, "one")
		app.errorJSON(w, err, nil)
		return
	}

	if response.StatusCode != http.StatusAccepted {
		app.errorJSON(w, errors.New(jsonFromService.Message), nil, response.StatusCode)
		return
	}

	var payload jsonResponse
	payload.Error = jsonFromService.Error
	payload.StatusCode = http.StatusOK
	payload.Message = jsonFromService.Message
	payload.Data = jsonFromService.Data

	app.writeJSON(w, http.StatusOK, payload)

}

func (app *Config) ProductOwnerPermission(w http.ResponseWriter, r *http.Request) {

	// retrieve authorization token
	authorizationHeader := r.Header.Get("Authorization")

	// contruct the url
	authServiceUrl := fmt.Sprintf("%s%s", os.Getenv("AUTH_URL"), "product-owner-permissions")

	// call the service by creating a request
	request, err := http.NewRequest("GET", authServiceUrl, nil)

	// Set the "Authorization" header with your Bearer token
	request.Header.Set("authorization", authorizationHeader)

	// check for error
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
		log.Println(err, "one")
		app.errorJSON(w, err, nil)
		return
	}

	if response.StatusCode != http.StatusAccepted {
		app.errorJSON(w, errors.New(jsonFromService.Message), nil, response.StatusCode)
		return
	}

	var payload jsonResponse
	payload.Error = jsonFromService.Error
	payload.StatusCode = http.StatusOK
	payload.Message = jsonFromService.Message
	payload.Data = jsonFromService.Data

	app.writeJSON(w, http.StatusOK, payload)

}

func (app *Config) ProductOwnerCreateStaff(w http.ResponseWriter, r *http.Request) {

	//extract the request body
	var requestPayload SignupPayload

	//extract the requestbody
	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	// Validate the request payload
	if err := app.ValidataSignupInput(requestPayload); len(err) > 0 {
		app.errorJSON(w, errors.New("error creating user"), err, http.StatusBadRequest)
		return
	}

	// retrieve authorization token
	authorizationHeader := r.Header.Get("Authorization")

	//create some json we will send to authservice
	jsonData, _ := json.MarshalIndent(requestPayload, "", "\t")

	authServiceUrl := fmt.Sprintf("%s%s", os.Getenv("AUTH_URL"), "product-owner-create-staff")

	// call the service by creating a request
	request, err := http.NewRequest("POST", authServiceUrl, bytes.NewBuffer(jsonData))

	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	// Set the "Authorization" header with your Bearer token
	request.Header.Set("authorization", authorizationHeader)

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

	log.Println("response from auth service", jsonFromService.Message)
	if response.StatusCode != http.StatusAccepted {
		log.Println(jsonFromService.Message, jsonFromService)
		app.errorJSON(w, errors.New(jsonFromService.Message), nil)
		return
	}

	var payload jsonResponse
	payload.Error = jsonFromService.Error
	payload.StatusCode = http.StatusOK
	payload.Message = jsonFromService.Message
	payload.Data = jsonFromService.Data

	app.writeJSON(w, http.StatusOK, payload)
}

func (app *Config) ProductOwnerAssignPermission(w http.ResponseWriter, r *http.Request) {

	//extract the request body
	var requestPayload AssignPermissionPayload

	//extract the requestbody
	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	// Validate the request payload
	if err := app.ValidataAssignPermission(requestPayload); len(err) > 0 {
		app.errorJSON(w, errors.New("error assignin user permission"), err, http.StatusBadRequest)
		return
	}

	// retrieve authorization token
	authorizationHeader := r.Header.Get("Authorization")

	//create some json we will send to authservice
	jsonData, _ := json.MarshalIndent(requestPayload, "", "\t")

	authServiceUrl := fmt.Sprintf("%s%s", os.Getenv("AUTH_URL"), "assign-permission")

	// call the service by creating a request
	request, err := http.NewRequest("POST", authServiceUrl, bytes.NewBuffer(jsonData))

	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	// Set the "Authorization" header with your Bearer token
	request.Header.Set("authorization", authorizationHeader)

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

	log.Println("response from auth service", jsonFromService.Message)
	if response.StatusCode != http.StatusAccepted {
		log.Println(jsonFromService.Message, jsonFromService)
		app.errorJSON(w, errors.New(jsonFromService.Message), nil)
		return
	}

	var payload jsonResponse
	payload.Error = jsonFromService.Error
	payload.StatusCode = http.StatusOK
	payload.Message = jsonFromService.Message
	payload.Data = jsonFromService.Data

	app.writeJSON(w, http.StatusOK, payload)
}

func (app *Config) GetCountries(w http.ResponseWriter, r *http.Request) {

	// retrieve authorization token
	authorizationHeader := r.Header.Get("Authorization")

	// contruct the url
	authServiceUrl := fmt.Sprintf("%s%s", os.Getenv("AUTH_URL"), "countries")

	// call the service by creating a request
	request, err := http.NewRequest("GET", authServiceUrl, nil)

	// Set the "Authorization" header with your Bearer token
	request.Header.Set("authorization", authorizationHeader)

	// check for error
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
	payload.StatusCode = http.StatusOK
	payload.Message = jsonFromService.Message
	payload.Data = jsonFromService.Data

	app.writeJSON(w, http.StatusOK, payload)

}

func (app *Config) GetStates(w http.ResponseWriter, r *http.Request) {

	// retrieve authorization token
	authorizationHeader := r.Header.Get("Authorization")

	// contruct the url
	authServiceUrl := fmt.Sprintf("%s%s", os.Getenv("AUTH_URL"), "states")

	// call the service by creating a request
	request, err := http.NewRequest("GET", authServiceUrl, nil)

	// Set the "Authorization" header with your Bearer token
	request.Header.Set("authorization", authorizationHeader)

	// check for error
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
	payload.StatusCode = http.StatusOK
	payload.Message = jsonFromService.Message
	payload.Data = jsonFromService.Data

	app.writeJSON(w, http.StatusOK, payload)

}

func (app *Config) GetLgas(w http.ResponseWriter, r *http.Request) {

	// retrieve authorization token
	authorizationHeader := r.Header.Get("Authorization")

	// contruct the url
	authServiceUrl := fmt.Sprintf("%s%s", os.Getenv("AUTH_URL"), "lgas")

	// call the service by creating a request
	request, err := http.NewRequest("GET", authServiceUrl, nil)

	// Set the "Authorization" header with your Bearer token
	request.Header.Set("authorization", authorizationHeader)

	// check for error
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
	payload.StatusCode = http.StatusOK
	payload.Message = jsonFromService.Message
	payload.Data = jsonFromService.Data

	app.writeJSON(w, http.StatusOK, payload)

}

func (app *Config) GetCountryState(w http.ResponseWriter, r *http.Request) {

	id := chi.URLParam(r, "id")

	if id == "" {
		app.errorJSON(w, errors.New("id parameter is missing"), nil)
		return
	}

	// Retrieve authorization token
	authorizationHeader := r.Header.Get("Authorization")
	if authorizationHeader == "" {
		app.errorJSON(w, errors.New("authorization token is missing"), nil)
		return
	}

	// Construct the URL
	authServiceUrl := fmt.Sprintf("%s%s%s", os.Getenv("AUTH_URL"), "country/state/", id)

	log.Println(authServiceUrl, "url")

	// Call the service by creating a request
	request, err := http.NewRequest("GET", authServiceUrl, nil)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	// Set the Authorization and Content-Type headers
	request.Header.Set("Authorization", authorizationHeader)
	request.Header.Set("Content-Type", "application/json")

	// Create an HTTP client and execute the request
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}
	defer response.Body.Close()

	// Decode the JSON response
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

	// Send the successful response
	var payload jsonResponse
	payload.Error = jsonFromService.Error
	payload.StatusCode = http.StatusOK
	payload.Message = jsonFromService.Message
	payload.Data = jsonFromService.Data

	app.writeJSON(w, http.StatusOK, payload)
}

func (app *Config) GetStateLga(w http.ResponseWriter, r *http.Request) {

	id := chi.URLParam(r, "id")

	if id == "" {
		app.errorJSON(w, errors.New("id parameter is missing"), nil)
		return
	}

	// Retrieve authorization token
	authorizationHeader := r.Header.Get("Authorization")
	if authorizationHeader == "" {
		app.errorJSON(w, errors.New("authorization token is missing"), nil)
		return
	}

	// Construct the URL
	authServiceUrl := fmt.Sprintf("%s%s%s", os.Getenv("AUTH_URL"), "state/lgas/", id)
	log.Println(authServiceUrl, "url")

	// Call the service by creating a request
	request, err := http.NewRequest("GET", authServiceUrl, nil)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	// Set the Authorization and Content-Type headers
	request.Header.Set("Authorization", authorizationHeader)
	request.Header.Set("Content-Type", "application/json")

	// Create an HTTP client and execute the request
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}
	defer response.Body.Close()

	// Decode the JSON response
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

	// Send the successful response
	var payload jsonResponse
	payload.Error = jsonFromService.Error
	payload.StatusCode = http.StatusOK
	payload.Message = jsonFromService.Message
	payload.Data = jsonFromService.Data

	app.writeJSON(w, http.StatusOK, payload)
}

func (app *Config) KycRenter(w http.ResponseWriter, r *http.Request) {
	// Parse the multipart form
	err := r.ParseMultipartForm(10 << 20) // Limit to 10 MB
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	// Retrieve the file from the form data
	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}
	defer file.Close()

	// Copy file data into a buffer
	var fileBuffer bytes.Buffer
	_, err = io.Copy(&fileBuffer, file)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	// Retrieve additional form fields
	address := r.FormValue("address")
	idNumber := r.FormValue("id_number")
	idType := r.FormValue("id_type")
	addressCountry := r.FormValue("address_country")
	addressState := r.FormValue("address_state")
	addressLga := r.FormValue("address_lga")

	// Retrieve the Bearer token from the Authorization header
	bearerToken := r.Header.Get("Authorization")
	if bearerToken == "" {
		app.errorJSON(w, errors.New("authorization header missing"), nil, http.StatusUnauthorized)
		return
	}

	// Forward the file, form fields, and token to the NestJS service
	app.forwardDataToNestJS(w, &fileBuffer, fileHeader.Filename, address, idNumber, idType, addressCountry, addressState, addressLga, bearerToken)
}

func (app *Config) forwardDataToNestJS(
	w http.ResponseWriter,
	fileData *bytes.Buffer,
	originalFileName string,
	address string,
	idNumber string,
	idType string,
	addressCountry string,
	addressState string,
	addressLga string,
	bearerToken string,
) {
	// URL of the auth service
	authServiceUrl := fmt.Sprintf("%s%s", os.Getenv("AUTH_URL"), "renter-kyc")

	// Create a new multipart writer
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// Add the file to the request body using the original file name
	fileWriter, err := writer.CreateFormFile("file", originalFileName)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}
	_, err = io.Copy(fileWriter, fileData)
	if err != nil {

		app.errorJSON(w, err, nil)
		return
	}

	// Add the additional form fields
	_ = writer.WriteField("address", address)
	_ = writer.WriteField("id_number", idNumber)
	_ = writer.WriteField("id_type", idType)
	_ = writer.WriteField("address_country", addressCountry)
	_ = writer.WriteField("address_state", addressState)
	_ = writer.WriteField("address_lga", addressLga)
	_ = writer.WriteField("bearer_token", bearerToken)

	// Close the writer to finalize the request body
	writer.Close()

	// Create a new POST request
	req, err := http.NewRequest("POST", authServiceUrl, &requestBody)
	if err != nil {

		app.errorJSON(w, err, nil)
		return
	}

	// Set headers
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", bearerToken) // Include the Bearer token

	// Create an HTTP client and execute the request
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}
	defer response.Body.Close()

	// Decode the JSON response
	var jsonFromService jsonResponse
	err = json.NewDecoder(response.Body).Decode(&jsonFromService)
	if err != nil {
		log.Println(err, 2)
		app.errorJSON(w, err, nil)
		return
	}

	// Check if the response status is not accepted
	if response.StatusCode != http.StatusAccepted {
		app.errorJSON(w, errors.New(jsonFromService.Message), nil, response.StatusCode)
		return
	}

	// If successful, send the JSON response back to the caller using writeJSON
	payload := jsonResponse{
		Error:      jsonFromService.Error,
		StatusCode: http.StatusOK,
		Message:    jsonFromService.Message,
		Data:       jsonFromService.Data,
	}

	// Use the writeJSON utility function to return the response
	err = app.writeJSON(w, http.StatusOK, payload)
	if err != nil {
		log.Println(err, "0")
		app.errorJSON(w, err, nil)
	}
}

func (app *Config) RetriveIdentificationTypes(w http.ResponseWriter, r *http.Request) {

	log.Println("I reached here too")
	// retrieve authorization token
	authorizationHeader := r.Header.Get("Authorization")

	// contruct the url
	authServiceUrl := fmt.Sprintf("%s%s", os.Getenv("AUTH_URL"), "retrieve-identification-types")

	// call the service by creating a request
	request, err := http.NewRequest("GET", authServiceUrl, nil)

	// Set the "Authorization" header with your Bearer token
	request.Header.Set("authorization", authorizationHeader)

	// check for error
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
	payload.StatusCode = http.StatusOK
	payload.Message = jsonFromService.Message
	payload.Data = jsonFromService.Data

	app.writeJSON(w, http.StatusOK, payload)

}

// logStructFields logs all fields of a struct dynamically using reflection
func logStructFields(v interface{}) {
	val := reflect.ValueOf(v).Elem() // get the underlying value of the pointer
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		value := val.Field(i).Interface()

		log.Printf("%s: %v", field.Name, value)
	}
}

//TODO
// 1. format role & permission into separate arrays - done
// 2. Try adding additional permission to a user - done
// 3. also extract the additional permission into the permissions array - done
// 4. Renters staff relationship (table renters_staff (renter_id, user_id, )) - done
// 5. Extract the user who added you

// Ask user if they have company
// if the company is registered or not
// Ask for CAC number if registered
// Ask company phone number
// Ask Company address (state & LGA)
