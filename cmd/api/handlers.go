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
	"net/rpc"
	"os"
	"reflect"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-redis/redis"
	"github.com/obynonwane/broker-service/utility"
	"github.com/obynonwane/rental-service-proto/inventory"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Define a custom type for the context key

var ctx = context.Background()

type MailPayload struct {
	From    string                 `json:"from"`
	To      string                 `json:"to"`
	Subject string                 `json:"subject"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data"`
}

type SignupPayload struct {
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	Email      string `json:"email"`
	Phone      string `json:"phone"`
	Password   string `json:"password"`
	IsBusiness string `json:"is_business"`
}

type CreateStaffPayload struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	Password  string `json:"password"`
	Role      string `json:"role"`
}

type ChooseRolePayload struct {
	UserType string `json:"user_type"`
}

type LoginPayload struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type BusinessKycPayload struct {
	BusinessRegistered string `json:"business_registered"`
	CacNumber          string `json:"cac_number"`
	DisplayName        string `json:"display_name"`
	AddressCountry     string `json:"address_country"`
	AddressState       string `json:"address_state"`
	AddressLga         string `json:"address_lga"`
	AddressStreet      string `json:"address_street"`
	Description        string `json:"description"`
	KeyBonus           string `json:"key_bonus"`
	Subdomain          string `json:"subdomain"`
	Industries         string `json:"industries"`
}

type LogPayload struct {
	Name string `json:"name"`
	Data string `json:"data"`
}
type RPCPayload struct {
	Name string
	Data string
}

type ReplyRatingPayload struct {
	RatingID      string `json:"rating_id,omitempty"`
	ReplierID     string `json:"replier_id,omitempty"`
	Comment       string `json:"comment"`
	ParentReplyID string `json:"parent_reply_id"`
}

type SearchPayload struct {
	CountryID     string `json:"country_id"`
	StateID       string `json:"state_id"`
	LgaID         string `json:"lga_id"`
	Text          string `json:"text"`
	Limit         string `json:"limit"`
	Offset        string `json:"offset"`
	CategoryID    string `json:"category_id"`
	SubcategoryID string `json:"subcategory_id"`
	Ulid          string `json:"ulid"`

	StateSlug       string `json:"state_slug"`
	CountrySlug     string `json:"country_slug"`
	LgaSlug         string `json:"lga_slug"`
	CategorySlug    string `json:"category_slug"`
	SubcategorySlug string `json:"subcategory_slug"`
	UserID          string `json:"user_id"`
	ProductPurpose  string `json:"product_purpose"`
}

type GetCategoryByIDPayload struct {
	CategoryID   string `json:"category_id"`
	CategorySlug string `json:"category_slug"`
}

type ResetPasswordEmailPayload struct {
	Email string `json:"email"`
}

type ChangePasswordPayload struct {
	Token           string `json:"token"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
}

type IndexInventoryPayload struct {
	Id string `json:"id"`
}

type RequestPasswordVerificationEmailPayload struct {
	Email string `json:"email"`
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
					app.errorJSON(w, err, nil, jsonFromService.StatusCode)
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

				//convert the payload into string
				b, err := json.Marshal(payload)
				if err != nil {
					app.errorJSON(w, errors.New("error marshalling payload into string for saving to redis"), payload.StatusCode)
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
				app.errorJSON(w, err, nil, jsonFromService.StatusCode)
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
				app.errorJSON(w, errors.New("error verifying token"), nil, response.StatusCode)
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
		app.errorJSON(w, errors.New("error logging out"), nil, response.StatusCode)
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
		app.errorJSON(w, errors.New("error sending mail"), nil, response.StatusCode)
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

func (app *Config) ParticipantCreateStaff(w http.ResponseWriter, r *http.Request) {

	//extract the request body
	var requestPayload CreateStaffPayload

	//extract the requestbody
	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	// Validate the request payload
	if err := app.ValidateCreateStaffInput(requestPayload); len(err) > 0 {
		app.errorJSON(w, errors.New("error creating user"), err, http.StatusBadRequest)
		return
	}

	// retrieve authorization token
	authorizationHeader := r.Header.Get("Authorization")

	//create some json we will send to authservice
	jsonData, _ := json.MarshalIndent(requestPayload, "", "\t")

	authServiceUrl := fmt.Sprintf("%s%s", os.Getenv("AUTH_URL"), "participant-create-staff")

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
	// authorizationHeader := r.Header.Get("Authorization")
	// if authorizationHeader == "" {
	// 	app.errorJSON(w, errors.New("authorization token is missing"), nil)
	// 	return
	// }

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
	// request.Header.Set("Authorization", authorizationHeader)
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
	authServiceUrl := fmt.Sprintf("%s%s", os.Getenv("AUTH_URL"), "kyc-renter")

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
func (app *Config) RetriveIndustries(w http.ResponseWriter, r *http.Request) {

	// retrieve authorization token
	authorizationHeader := r.Header.Get("Authorization")

	// contruct the url
	authServiceUrl := fmt.Sprintf("%s%s", os.Getenv("AUTH_URL"), "retrieve-industries")

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
func (app *Config) ListUserTypes(w http.ResponseWriter, r *http.Request) {

	// retrieve authorization token
	authorizationHeader := r.Header.Get("Authorization")

	// contruct the url
	authServiceUrl := fmt.Sprintf("%s%s", os.Getenv("AUTH_URL"), "list-user-type")

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

func (app *Config) KycBusiness(w http.ResponseWriter, r *http.Request) {

	//extract the request body
	var requestPayload BusinessKycPayload

	//extract the requestbody
	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	//create some json we will send to authservice
	jsonData, _ := json.MarshalIndent(requestPayload, "", "\t")

	authServiceUrl := fmt.Sprintf("%s%s", os.Getenv("AUTH_URL"), "kyc-business")

	//get authorization hearder
	authorizationHeader := r.Header.Get("Authorization")

	// call the service by creating a request
	request, err := http.NewRequest("POST", authServiceUrl, bytes.NewBuffer(jsonData))

	// Set the "Authorization" header with your Bearer token
	request.Header.Set("authorization", authorizationHeader)

	if err != nil {
		log.Println("error 1")
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

func (app *Config) SignupAdmin(w http.ResponseWriter, r *http.Request) {

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

	authServiceUrl := fmt.Sprintf("%s%s", os.Getenv("AUTH_URL"), "admin/signup")

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

func (app *Config) testRPC(w http.ResponseWriter, r *http.Request) {

	data := LogPayload{
		Name: "testing",
		Data: "The data",
	}
	app.logItemViaRPC(w, data)
}

func (app *Config) logItemViaRPC(w http.ResponseWriter, l LogPayload) {
	// 1. get the RPC client
	client, err := rpc.Dial("tcp", "logging-service:5001")
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	// 2. create a payload exact as the type the rpc remote server expects
	rpcPayload := RPCPayload{
		Name: l.Name,
		Data: l.Data,
	}

	// 3. result to be received from the remote rpc call
	var result string
	err = client.Call("RPCServer.LogInfo", rpcPayload, &result)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: result,
	}

	app.writeJSON(w, http.StatusOK, payload)
}

// UserDTO is a data transfer object to match the server's User structure
type UserDTO struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name,omitempty"`
	LastName  string    `json:"last_name,omitempty"`
	Verified  bool      `json:"verified"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (app *Config) CreateInventory(w http.ResponseWriter, r *http.Request) {

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

	//=========================================Extracting form details===============================================================

	name := r.FormValue("name")
	product_purpose := ProductPurpose(r.FormValue("product_purpose"))
	quantity := utility.ParseStringToDouble(r.FormValue("quantity"))
	is_available := AvailabilityStatus(r.FormValue("is_available"))
	rental_duration := RentalDuration(r.FormValue("rental_duration"))
	offer_price := utility.ParseStringToDouble(r.FormValue("offer_price"))
	minimum_price := utility.ParseStringToDouble(r.FormValue("minimum_price"))
	security_deposit := utility.ParseStringToDouble(r.FormValue("security_deposit"))
	country_id := r.FormValue("country_id")
	state_id := r.FormValue("state_id")
	lga_id := r.FormValue("lga_id")
	category_id := r.FormValue("category_id")
	sub_category_id := r.FormValue("sub_category_id")
	description := r.FormValue("description")
	tags := r.FormValue("tags")
	metadata := r.FormValue("metadata")
	negotiable := NegotiableStatus(r.FormValue("negotiable"))
	condition := r.FormValue("condition")
	usage_guide := r.FormValue("usage_guide")
	included := r.FormValue("included")
	//================================================================================================================================

	// check the inputs
	if !product_purpose.IsValid() {
		app.errorJSON(w, errors.New("invalid product_purpose: either rental or sale"), http.StatusBadRequest)
		return
	}
	if !is_available.IsValid() {
		app.errorJSON(w, errors.New("invalid is_available: either yes or no"), http.StatusBadRequest)
		return
	}
	if minimum_price > offer_price {
		app.errorJSON(w, errors.New("invalid: minimum price can not be greater than offer price"), http.StatusBadRequest)
		return
	}
	if product_purpose == ProductPurposeRental {
		if !rental_duration.IsValid() {
			app.errorJSON(w, errors.New("invalid rental duration: either hourly, daily, monthly or  annually"), http.StatusBadRequest)
			return
		}

		if security_deposit <= 0 {
			security_deposit = 0
		}
	}
	if !negotiable.IsValid() {
		app.errorJSON(w, errors.New("invalid negotiable value: eithe yes or no"), http.StatusBadRequest)
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

	//=========================================Working on  Image Array ===============================================================
	var images []*inventory.ImageData

	// 3. Validate the request payload
	if err := app.ValidateCreateInventoryInput(category_id, sub_category_id, name, description, country_id, state_id, lga_id, offer_price); len(err) > 0 {
		app.errorJSON(w, errors.New("error trying to create inventory"), err, http.StatusBadRequest)
		return
	}

	// Iterate over all files with the key "images"
	files := r.MultipartForm.File["images"]
	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			http.Error(w, "Error reading image file", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Read file content into a buffer
		var imageData bytes.Buffer
		_, err = io.Copy(&imageData, file)
		if err != nil {
			app.errorJSON(w, errors.New("error reading image data"), nil)
			return
		}

		// Detect MIME type of the image
		imageType := http.DetectContentType(imageData.Bytes()[:512]) // Inspect the first 512 bytes

		// Log detected MIME type for debugging purposes
		log.Printf("Detected MIME type: %s\n", imageType)

		// Append to images array for gRPC request
		images = append(images, &inventory.ImageData{
			ImageData: imageData.Bytes(),
			ImageType: imageType, // Example MIME type; adjust as necessary
		})
	}
	//================================================================================================================================

	//=========================================Working on Primary Image ===============================================================

	var primaryImage *inventory.ImageData

	// Retrieve the primary image from the form data
	primary_image, _, err := r.FormFile("primary_image")
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}
	defer primary_image.Close()

	// Copy primary image data into a buffer
	var primaryImageDataBuffer bytes.Buffer
	_, err = io.Copy(&primaryImageDataBuffer, primary_image)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	// Get the image type safely
	imageBytes := primaryImageDataBuffer.Bytes()
	sampleSize := len(imageBytes)
	if sampleSize > 512 {
		sampleSize = 512
	}
	primaryImageType := http.DetectContentType(primaryImageDataBuffer.Bytes()[:512])

	primaryImage = &inventory.ImageData{
		ImageData: primaryImageDataBuffer.Bytes(),
		ImageType: primaryImageType,
	}

	//====================================================================================================================================

	//========================================================Make Call Via gRPC to Inventory service======================================
	conn, err := grpc.Dial("inventory-service:50001", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	// leave connection open forever
	defer conn.Close()

	c := inventory.NewInventoryServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // Increased timeout
	defer cancel()

	data, err := c.CreateInventory(ctx, &inventory.CreateInventoryRequest{
		CategoryId:      category_id,
		SubCategoryId:   sub_category_id,
		CountryId:       country_id,
		StateId:         state_id,
		LgaId:           lga_id,
		Name:            name,
		Description:     description,
		Images:          images,
		UserId:          userID,
		OfferPrice:      offer_price,
		ProductPurpose:  string(product_purpose),
		Quantity:        quantity,
		IsAvailable:     string(is_available),
		RentalDuration:  string(rental_duration),
		SecurityDeposit: security_deposit,
		Tags:            tags,
		Metadata:        metadata,
		Negotiable:      string(negotiable),
		PrimaryImage:    primaryImage,
		MinimumPrice:    minimum_price,
		Condition:       condition,
		UsageGuide:      usage_guide,
		Included:        included,
	})

	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}
	//====================================================================================================================================

	var payload jsonResponse
	payload.Error = data.Error
	payload.Message = data.Message
	payload.StatusCode = 200
	payload.Data = data

	app.writeJSON(w, http.StatusAccepted, payload)

}

func (app *Config) GetUsersViaGrpc(w http.ResponseWriter, r *http.Request) {
	// get a gRPC client and dial using tcp
	conn, err := grpc.Dial("inventory-service:50001", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}
	defer conn.Close()

	c := inventory.NewInventoryServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // Increased timeout
	defer cancel()

	// Channel to receive the response from the goroutine
	responseChannel := make(chan *inventory.UserListResponse)
	errorChannel := make(chan error)

	// Call the GetUsers method asynchronously in a goroutine
	go func() {
		data, err := c.GetUsers(ctx, &inventory.EmptyRequest{})
		if err != nil {
			errorChannel <- err // Send error to the error channel
			return
		}
		responseChannel <- data // Send the response to the response channel
	}()

	// Wait for either the response, error, or timeout to be sent through the channels
	select {
	case data := <-responseChannel:
		// Successfully received the data, prepare the response
		var payload jsonResponse
		payload.Error = false
		payload.Message = "User details retrieved successfully"
		payload.Data = data.Users

		app.writeJSON(w, http.StatusAccepted, payload)

	case err := <-errorChannel:
		// If there was an error calling the gRPC method, handle it
		log.Println("Error retrieving users:", err)
		app.errorJSON(w, err, nil)

	case <-ctx.Done():
		// If the operation timed out, handle the timeout error
		log.Println("Error: gRPC request timed out")
		app.errorJSON(w, fmt.Errorf("gRPC request timed out"), nil)
	}
}

func (app *Config) AllCategories(w http.ResponseWriter, r *http.Request) {

	// get a gRPC client and dial using tcp
	conn, err := grpc.Dial("inventory-service:50001", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}
	defer conn.Close()

	c := inventory.NewInventoryServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second) // Increased timeout
	defer cancel()

	// channel to receive response from go routince

	responseChannel := make(chan *inventory.AllCategoryResponse)
	errorChannel := make(chan error)

	go func() {
		data, err := c.GetCategories(ctx, &inventory.EmptyRequest{})
		if err != nil {
			errorChannel <- err
			return
		}

		responseChannel <- data
	}()

	select {

	case data := <-responseChannel:
		var payload jsonResponse
		payload.Error = false
		payload.Message = "Categories retrieved successfully"
		payload.Data = data.Categories
		payload.StatusCode = int(data.StatusCode)

		app.writeJSON(w, http.StatusAccepted, payload)

	case err := <-errorChannel:
		log.Println("Error retrieving users:", err)
		app.errorJSON(w, err, nil)

	case <-ctx.Done():
		// If the operation timed out, handle the timeout error
		log.Println("Error: gRPC request timed out")
		app.errorJSON(w, fmt.Errorf("gRPC request timed out"), nil)

	}
}

func (app *Config) AllSubcategories(w http.ResponseWriter, r *http.Request) {
	// get a gRPC client and dial using tcp
	conn, err := grpc.Dial("inventory-service:50001", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}
	defer conn.Close()

	c := inventory.NewInventoryServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // Increased timeout
	defer cancel()

	// channel to receive response from go routince

	responseChannel := make(chan *inventory.AllSubCategoryResponse)
	errorChannel := make(chan error)

	go func() {
		data, err := c.GetSubCategories(ctx, &inventory.EmptyRequest{})
		if err != nil {
			errorChannel <- err
			return
		}

		responseChannel <- data
	}()

	select {

	case data := <-responseChannel:
		var payload jsonResponse
		payload.Error = false
		payload.Message = "Subcategories retrieved successfully"
		payload.Data = data.Subcategories
		payload.StatusCode = int(data.StatusCode)

		app.writeJSON(w, http.StatusAccepted, payload)

	case err := <-errorChannel:
		log.Println("Error retrieving subcategories:", err)
		app.errorJSON(w, err, nil)

	case <-ctx.Done():
		// If the operation timed out, handle the timeout error
		log.Println("Error: gRPC request timed out")
		app.errorJSON(w, fmt.Errorf("gRPC request timed out"), nil)

	}

}
func (app *Config) GetCategorySubcategories(w http.ResponseWriter, r *http.Request) {
	// get a gRPC client and dial using tcp

	categoryId := chi.URLParam(r, "id")

	conn, err := grpc.Dial("inventory-service:50001", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())

	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}
	defer conn.Close()

	c := inventory.NewInventoryServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // Increased timeout
	defer cancel()

	// channel to receive response from go routince

	responseChannel := make(chan *inventory.AllSubCategoryResponse)
	errorChannel := make(chan error)

	go func() {
		data, err := c.GetCategorySubcategories(ctx, &inventory.ResourceId{Id: categoryId})
		if err != nil {
			errorChannel <- err
			return
		}

		responseChannel <- data
	}()

	select {

	case data := <-responseChannel:
		var payload jsonResponse
		payload.Error = false
		payload.Message = "Subcategories retrieved successfully"
		payload.Data = data.Subcategories
		payload.StatusCode = int(data.StatusCode)

		app.writeJSON(w, http.StatusAccepted, payload)

	case err := <-errorChannel:
		log.Println("Error retrieving subcategories:", err)
		app.errorJSON(w, err, nil)

	case <-ctx.Done():
		// If the operation timed out, handle the timeout error
		log.Println("Error: gRPC request timed out")
		app.errorJSON(w, fmt.Errorf("gRPC request timed out"), nil)

	}

}

func (app *Config) GetCategoryByID(w http.ResponseWriter, r *http.Request) {

	category_id := r.FormValue("category_id")
	category_slug := r.FormValue("category_slug")

	// establish connection via grpc
	conn, err := grpc.Dial("inventory-service:50001", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}
	defer conn.Close() // defer closing connection untill function execution is complete

	// instantiate a new instnnce of inventory service from proto definition
	c := inventory.NewInventoryServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // Increased timeout
	defer cancel()

	// declare response and error channel
	responseChannel := make(chan *inventory.CategoryResponse)
	errorChannel := make(chan error)

	// declare a go routine to initiate asynchronour process
	go func(category_id, category_slug string) {
		data, err := c.GetCategory(ctx, &inventory.GetCategoryByIDPayload{
			CategoryId:   category_id,
			CategorySlug: category_slug})
		if err != nil {
			errorChannel <- err
			return
		}
		responseChannel <- data

	}(category_id, category_slug)

	select {
	case data := <-responseChannel:
		var payload jsonResponse
		payload.Error = false
		payload.Message = "Category retrieved successfully"
		payload.Data = data
		payload.StatusCode = int(data.StatusCode)

		app.writeJSON(w, http.StatusAccepted, payload)

	case err := <-errorChannel:
		log.Println("Error retrieving category:", err)
		app.errorJSON(w, err, nil)

	case <-ctx.Done():
		// If the operation timed out, handle the timeout error
		log.Println("Error: gRPC request timed out")
		app.errorJSON(w, fmt.Errorf("gRPC request timed out"), nil)

	}

}

func (app *Config) ExtractLoggedInUser(w http.ResponseWriter, r *http.Request) (string, error, int) {
	// verify the user token
	response, err := app.getToken(r)
	if err != nil {
		log.Println(response, "The response HERE")
		return "", fmt.Errorf("%s", response.Message), response.StatusCode
	}

	if response.Error {
		log.Println(response, "The response HERE")
		return "", fmt.Errorf("%s", response.Message), response.StatusCode
	}

	// Extract user ID from response.Data
	var userID string
	if response.Data != nil {
		// Assert response.Data is a map
		dataMap, ok := response.Data.(map[string]any)
		if !ok {
			return "", fmt.Errorf("invalid data format"), response.StatusCode
		}

		// Extract "user" field and assert it is a map
		userData, ok := dataMap["user"].(map[string]any)
		if !ok {
			return "", fmt.Errorf("missing or invalid user data"), response.StatusCode
		}

		// Extract "id" field and assert it is a string
		userID, ok = userData["id"].(string)
		if !ok {
			return "", fmt.Errorf("missing or invalid user ID"), response.StatusCode
		}
	}

	return userID, nil, response.StatusCode
}

func (app *Config) RateInventory(w http.ResponseWriter, r *http.Request) {
	// extract the request params
	queryParams := r.URL.Query()
	comment := queryParams.Get("comment")
	if comment == "" {
		app.errorJSON(w, errors.New("comment on rating not supplied"), nil)
		return
	}
	inventory_id := queryParams.Get("inventory_id")
	if inventory_id == "" {
		app.errorJSON(w, errors.New("inventory id must be supplied"), nil)
		return
	}

	rating := queryParams.Get("rating")
	if rating == "" {
		app.errorJSON(w, errors.New("inventory id must be supplied"), nil)
		return
	}

	// convert rating into int32
	ratingInt, err := strconv.ParseInt(rating, 10, 25)
	if err != nil {
		app.errorJSON(w, errors.New("error convering rrating to int"), nil)
		return
	}
	ratingInt32 := int32(ratingInt)

	// the user
	userId, err, statusCodeRes := app.ExtractLoggedInUser(w, r)
	if err != nil {
		app.errorJSON(w, err, nil, statusCodeRes)
		return
	}

	// establish connection via grpc
	conn, err := grpc.Dial("inventory-service:50001", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}
	defer conn.Close() // defer closing connection untill function execution is complete

	c := inventory.NewInventoryServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // Increased timeout
	defer cancel()

	resultCh := make(chan *inventory.InventoryRatingResponse, 1)
	errorChannel := make(chan error, 1)
	go func(ratingInt32 int32, comment string, inventory_id string) {
		result, err := c.RateInventory(ctx, &inventory.InventoryRatingRequest{
			InventoryId: inventory_id,
			Rating:      ratingInt32,
			Comment:     comment,
			RaterId:     userId,
		})
		if err != nil {
			errorChannel <- err
		}
		resultCh <- result

	}(ratingInt32, comment, inventory_id)

	select {
	case data := <-resultCh:

		var payload jsonResponse
		payload.Error = false
		payload.Message = "Rating sucessfully submitted"
		payload.Data = data
		payload.StatusCode = 200
		app.writeJSON(w, http.StatusAccepted, payload)

	case err := <-errorChannel:
		log.Println("Error creating rating:", err)
		app.errorJSON(w, err, nil)

	case <-ctx.Done():
		// If the operation timed out, handle the timeout error
		log.Println("Error: gRPC request timed out")
		app.errorJSON(w, fmt.Errorf("gRPC request timed out"), nil)
	}

}

func (app *Config) RateUser(w http.ResponseWriter, r *http.Request) {
	// extract the request params
	queryParams := r.URL.Query()
	comment := queryParams.Get("comment")
	if comment == "" {
		app.errorJSON(w, errors.New("comment on rating not supplied"), nil)
		return
	}
	user_id := queryParams.Get("user_id")
	if user_id == "" {
		app.errorJSON(w, errors.New("user id must be supplied"), nil)
		return
	}

	rating := queryParams.Get("rating")
	if rating == "" {
		app.errorJSON(w, errors.New("inventory id must be supplied"), nil)
		return
	}

	// convert rating into int32
	ratingInt, err := strconv.ParseInt(rating, 10, 25)
	if err != nil {
		app.errorJSON(w, errors.New("error convering rrating to int"), nil)
		return
	}
	ratingInt32 := int32(ratingInt)

	// the user
	raterId, err, statusCodeRes := app.ExtractLoggedInUser(w, r)
	if err != nil {
		app.errorJSON(w, err, nil, statusCodeRes)
		return
	}

	// establish connection via grpc
	conn, err := grpc.Dial("inventory-service:50001", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}
	defer conn.Close() // defer closing connection untill function execution is complete

	c := inventory.NewInventoryServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // Increased timeout
	defer cancel()

	resultCh := make(chan *inventory.UserRatingResponse, 1)
	errorChannel := make(chan error, 1)
	go func(ratingInt32 int32, comment string, user_id string) {
		result, err := c.RateUser(ctx, &inventory.UserRatingRequest{
			UserId:  user_id,
			Rating:  ratingInt32,
			Comment: comment,
			RaterId: raterId,
		})
		if err != nil {
			errorChannel <- err
		}
		resultCh <- result

	}(ratingInt32, comment, user_id)

	select {
	case data := <-resultCh:

		var payload jsonResponse
		payload.Error = false
		payload.Message = "Rating sucessfully submitted"
		payload.Data = data
		payload.StatusCode = 200
		app.writeJSON(w, http.StatusAccepted, payload)

	case err := <-errorChannel:
		log.Println("Error creating rating:", err)
		app.errorJSON(w, err, nil)

	case <-ctx.Done():
		// If the operation timed out, handle the timeout error
		log.Println("Error: gRPC request timed out")
		app.errorJSON(w, fmt.Errorf("gRPC request timed out"), nil)
	}
}

func (app *Config) GetInventoryDetail(w http.ResponseWriter, r *http.Request) {

	// 1. retrieve inventory id
	id := chi.URLParam(r, "id")

	slug_ulid := r.FormValue("slug_ulid")
	inventory_id := r.FormValue("inventory_id")

	//2.  establish connection via grpc
	conn, err := grpc.Dial("inventory-service:50001", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}
	defer conn.Close() // defer closing connection untill function execution is complete

	//3. instantiate a new instnnce of inventory service from proto definition
	c := inventory.NewInventoryServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // Increased timeout
	defer cancel()

	//4. create result & error channel
	resultCh := make(chan *inventory.InventoryResponseDetail, 1)
	errorChannel := make(chan error, 1)

	go func(id string) {
		// make the call via grpc
		result, err := c.GetInventoryByID(ctx, &inventory.SingleInventoryRequestDetail{
			SlugUlid:    slug_ulid,
			InventoryId: inventory_id,
		})
		if err != nil {
			errorChannel <- err
		}
		resultCh <- result

	}(id)

	// 5. select statement to wait
	select {
	case data := <-resultCh:
		var payload jsonResponse
		payload.Error = false
		payload.Message = "Inventory sucessfully retrieved"
		payload.Data = data
		payload.StatusCode = 200
		app.writeJSON(w, http.StatusAccepted, payload)

	case err := <-errorChannel:
		log.Println("Error retrieving inventory:", err)
		app.errorJSON(w, err, nil)

	case <-ctx.Done():
		// If the operation timed out, handle the timeout error
		log.Println("Error: gRPC request timed out")
		app.errorJSON(w, fmt.Errorf("gRPC request timed out"), nil)
	}

}

func (app *Config) GetUserRatings(w http.ResponseWriter, r *http.Request) {

	// 1. retrieve user id
	id := chi.URLParam(r, "id")

	// 2. retrieve query param
	queryParams := r.URL.Query()
	page := queryParams.Get("page")
	if page == "" {
		app.errorJSON(w, errors.New("page not supplied"), nil)
		return
	}
	limit := queryParams.Get("limit")
	if limit == "" {
		app.errorJSON(w, errors.New("limit not supplied"), nil)
		return
	}

	//3.  establish connection via grpc
	conn, err := grpc.Dial("inventory-service:50001", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}
	defer conn.Close()

	//4. instantiate a new instnnce of inventory service from proto definition
	c := inventory.NewInventoryServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // Increased timeout
	defer cancel()

	//5. create result & error channel
	resultCh := make(chan *inventory.UserRatingsResponse, 1)
	errorChannel := make(chan error, 1)

	go func(id string, limit string, page string) {

		// convert string to int
		int64Page, err := strconv.ParseInt(page, 10, 32)
		if err != nil {
			fmt.Println("Error converting string to int32:", err)
			return
		}

		// convert string to int
		int64Limit, err := strconv.ParseInt(limit, 10, 32)
		if err != nil {
			fmt.Println("Error converting string to int32:", err)
			return
		}

		// make the call via grpc
		result, err := c.GetUserRatings(ctx, &inventory.GetResourceWithIDAndPagination{
			Id:         &inventory.ResourceId{Id: id},
			Pagination: &inventory.PaginationParam{Page: int32(int64Page), Limit: int32(int64Limit)},
		})
		if err != nil {
			errorChannel <- err
		}
		resultCh <- result

	}(id, limit, page)

	// 6. select statement to wait
	select {
	case data := <-resultCh:
		var payload jsonResponse
		payload.Error = false
		payload.Message = "User ratings sucessfully retrieved"
		payload.Data = data
		payload.StatusCode = 200
		app.writeJSON(w, http.StatusAccepted, payload)

	case err := <-errorChannel:
		log.Println("Error retrieving inventory:", err)
		app.errorJSON(w, err, nil)

	case <-ctx.Done():
		// If the operation timed out, handle the timeout error
		log.Println("Error: gRPC request timed out")
		app.errorJSON(w, fmt.Errorf("gRPC request timed out"), nil)
	}

}

func (app *Config) GetInventoryRatings(w http.ResponseWriter, r *http.Request) {
	// 1. retrieve user id
	id := chi.URLParam(r, "id")

	// 2. retrieve query param
	queryParams := r.URL.Query()
	page := queryParams.Get("page")
	if page == "" {
		app.errorJSON(w, errors.New("page not supplied"), nil)
		return
	}
	limit := queryParams.Get("limit")
	if limit == "" {
		app.errorJSON(w, errors.New("limit not supplied"), nil)
		return
	}

	//3.  establish connection via grpc
	conn, err := grpc.Dial("inventory-service:50001", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}
	defer conn.Close()

	//4. instantiate a new instnnce of inventory service from proto definition
	c := inventory.NewInventoryServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // Increased timeout
	defer cancel()

	//5. create result & error channel
	resultCh := make(chan *inventory.InventoryRatingsResponse, 1)
	errorChannel := make(chan error, 1)

	go func(id string, limit string, page string) {

		// convert string to int
		int64Page, err := strconv.ParseInt(page, 10, 32)
		if err != nil {
			fmt.Println("Error converting string to int32:", err)
			return
		}

		// convert string to int
		int64Limit, err := strconv.ParseInt(limit, 10, 32)
		if err != nil {
			fmt.Println("Error converting string to int32:", err)
			return
		}

		// make the call via grpc
		result, err := c.GetInventoryRatings(ctx, &inventory.GetResourceWithIDAndPagination{
			Id:         &inventory.ResourceId{Id: id},
			Pagination: &inventory.PaginationParam{Page: int32(int64Page), Limit: int32(int64Limit)},
		})
		if err != nil {
			errorChannel <- err
		}
		resultCh <- result

	}(id, limit, page)

	// 6. select statement to wait
	select {
	case data := <-resultCh:
		var payload jsonResponse
		payload.Error = false
		payload.Message = "Inventory ratings sucessfully retrieved"
		payload.Data = data
		payload.StatusCode = 200
		app.writeJSON(w, http.StatusAccepted, payload)

	case err := <-errorChannel:
		log.Println("Error retrieving inventory:", err)
		app.errorJSON(w, err, nil)

	case <-ctx.Done():
		// If the operation timed out, handle the timeout error
		log.Println("Error: gRPC request timed out")
		app.errorJSON(w, fmt.Errorf("gRPC request timed out"), nil)
	}
}

func (app *Config) ReplyInventoryRating(w http.ResponseWriter, r *http.Request) {

	response, err := app.getToken(r)
	if err != nil {
		app.errorJSON(w, err, response.Data, http.StatusUnauthorized)
		return
	}

	if response.Error {
		app.errorJSON(w, errors.New(response.Message), response.Data, response.StatusCode)
		return
	}

	replierID, err := app.returnLoggedInUserID(response)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	//1. variable of type ReplyRatingPayload
	var requestPayload ReplyRatingPayload

	//2. extract the requestbody
	err = app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	// 3. Validate the request payload
	if err := app.ValidateReplyRatingInput(requestPayload); len(err) > 0 {
		app.errorJSON(w, errors.New("error trying to reply rating"), err, http.StatusBadRequest)
		return
	}

	//4.  establish connection via grpc
	conn, err := grpc.Dial("inventory-service:50001", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}
	defer conn.Close()

	//5. instantiate a new instnnce of inventory service from proto definition
	c := inventory.NewInventoryServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // Increased timeout
	defer cancel()

	//6. create result & error channel
	resultCh := make(chan *inventory.ReplyToRatingResponse, 1)
	errorChannel := make(chan error, 1)

	go func(rating_id, replier_id, comment, parent_reply_id string) {
		// make the call via grpc
		result, err := c.ReplyInventoryRating(ctx, &inventory.ReplyToRatingRequest{
			RatingId:      rating_id,
			ReplierId:     replier_id,
			Comment:       comment,
			ParentReplyId: parent_reply_id,
		})
		if err != nil {
			errorChannel <- err
		}
		resultCh <- result

	}(requestPayload.RatingID, replierID, requestPayload.Comment, requestPayload.ParentReplyID)

	// 7. select statement to wait
	select {
	case data := <-resultCh:
		var payload jsonResponse
		payload.Error = false
		payload.Message = "Inventory rating replied sucessfully"
		payload.Data = data
		payload.StatusCode = 200
		app.writeJSON(w, http.StatusAccepted, payload)

	case err := <-errorChannel:
		log.Println("Error replying rating:", err)
		app.errorJSON(w, err, nil)

	case <-ctx.Done():
		// If the operation timed out, handle the timeout error
		log.Println("Error: gRPC request timed out")
		app.errorJSON(w, fmt.Errorf("gRPC request timed out"), nil)
	}
}

func (app *Config) ReplyUserRating(w http.ResponseWriter, r *http.Request) {

	response, err := app.getToken(r)
	if err != nil {
		app.errorJSON(w, err, response.Data, http.StatusUnauthorized)
		return
	}

	if response.Error {
		app.errorJSON(w, errors.New(response.Message), response.Data, response.StatusCode)
		return
	}

	replierID, err := app.returnLoggedInUserID(response)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	//1. variable of type ReplyRatingPayload
	var requestPayload ReplyRatingPayload

	//2. extract the requestbody
	err = app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	// 3. Validate the request payload
	if err := app.ValidateReplyRatingInput(requestPayload); len(err) > 0 {
		app.errorJSON(w, errors.New("error trying to reply rating"), err, http.StatusBadRequest)
		return
	}

	//4.  establish connection via grpc
	conn, err := grpc.Dial("inventory-service:50001", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}
	defer conn.Close()

	//5. instantiate a new instnnce of inventory service from proto definition
	c := inventory.NewInventoryServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // Increased timeout
	defer cancel()

	//6. create result & error channel
	resultCh := make(chan *inventory.ReplyToRatingResponse, 1)
	errorChannel := make(chan error, 1)

	go func(rating_id, replier_id, comment, parent_reply_id string) {
		// make the call via grpc
		result, err := c.ReplyUserRating(ctx, &inventory.ReplyToRatingRequest{
			RatingId:      rating_id,
			ReplierId:     replier_id,
			Comment:       comment,
			ParentReplyId: parent_reply_id,
		})
		if err != nil {
			errorChannel <- err
		}
		resultCh <- result

	}(requestPayload.RatingID, replierID, requestPayload.Comment, requestPayload.ParentReplyID)

	// 7. select statement to wait
	select {
	case data := <-resultCh:
		var payload jsonResponse
		payload.Error = false
		payload.Message = "User rating replied sucessfully"
		payload.Data = data
		payload.StatusCode = 200
		app.writeJSON(w, http.StatusAccepted, payload)

	case err := <-errorChannel:
		log.Println("Error replying rating:", err)
		app.errorJSON(w, err, nil)

	case <-ctx.Done():
		// If the operation timed out, handle the timeout error
		log.Println("Error: gRPC request timed out")
		app.errorJSON(w, fmt.Errorf("gRPC request timed out"), nil)
	}
}

func (app *Config) SearchInventory(w http.ResponseWriter, r *http.Request) {

	//1. variable of type ReplyRatingPayload
	var requestPayload SearchPayload

	//2. extract the requestbody
	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	// 3. Validate the request payload
	// if err := app.ValidateSearchInput(requestPayload); len(err) > 0 {
	// 	app.errorJSON(w, errors.New("error trying to reply rating"), err, http.StatusBadRequest)
	// 	return
	// }

	//4.  establish connection via grpc
	conn, err := grpc.Dial("inventory-service:50001", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}
	defer conn.Close()

	//5. instantiate a new instnnce of inventory service from proto definition
	c := inventory.NewInventoryServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // Increased timeout
	defer cancel()

	//6. create result & error channel
	resultCh := make(chan *inventory.InventoryCollection, 1)
	errorChannel := make(chan error, 1)

	go func(state_id, country_id, lga_id, text, limit, offset, category_id, subcategory_id, ulid, state_slug, country_slug, lga_slug, category_slug, subcategory_slug, user_id, product_purpose string) {
		// make the call via grpc
		result, err := c.SearchInventory(ctx, &inventory.SearchInventoryRequest{
			StateId:         state_id,
			CountryId:       country_id,
			LgaId:           lga_id,
			Text:            text,
			Limit:           limit,
			Offset:          offset,
			CategoryId:      category_id,
			SubcategoryId:   subcategory_id,
			Ulid:            ulid,
			StateSlug:       state_slug,
			CountrySlug:     country_slug,
			LgaSlug:         lga_slug,
			CategorySlug:    category_slug,
			SubcategorySlug: subcategory_slug,
			UserId:          user_id,
			ProductPurpose:  product_purpose,
		})
		if err != nil {
			errorChannel <- err
		}
		resultCh <- result

	}(
		requestPayload.StateID,
		requestPayload.CountryID,
		requestPayload.LgaID,
		requestPayload.Text,
		requestPayload.Limit,
		requestPayload.Offset,
		requestPayload.CategoryID,
		requestPayload.SubcategoryID,
		requestPayload.Ulid,
		requestPayload.StateSlug,
		requestPayload.CountrySlug,
		requestPayload.LgaSlug,
		requestPayload.CategorySlug,
		requestPayload.SubcategorySlug,
		requestPayload.UserID,
		requestPayload.ProductPurpose,
	)

	// 7. select statement to wait
	select {
	case data := <-resultCh:
		// var payload jsonResponse
		// payload.Error = false
		// payload.Message = "search retrieved succesfully"
		// payload.Data = data
		// payload.StatusCode = 200
		// app.writeJSON(w, http.StatusAccepted, payload)

		// Ensure inventories is not nil to return [] instead of null
		inventories := data.Inventories
		if inventories == nil {
			inventories = []*inventory.Inventory{}
		}

		// Build response payload
		mapped := struct {
			Inventories []*inventory.Inventory `json:"inventories"`
			TotalCount  int32                  `json:"total_count"`
			Offset      int32                  `json:"offset"`
			Limit       int32                  `json:"limit"`
		}{
			Inventories: inventories,
			TotalCount:  data.TotalCount,
			Offset:      data.Offset,
			Limit:       data.Limit,
		}

		payload := jsonResponse{
			Error:      false,
			Message:    "search retrieved successfully",
			StatusCode: 200,
			Data:       mapped,
		}
		app.writeJSON(w, http.StatusOK, payload)

	case err := <-errorChannel:
		log.Println("Error replying rating:", err)
		app.errorJSON(w, err, nil)

	case <-ctx.Done():
		// If the operation timed out, handle the timeout error
		log.Println("Error: gRPC request timed out")
		app.errorJSON(w, fmt.Errorf("gRPC request timed out"), nil)
	}
}

func (app *Config) returnLoggedInUserID(response jsonResponse) (string, error) {

	// Extract user ID from response.Data
	var userID string
	if response.Data != nil {
		// Assert response.Data is a map
		dataMap, ok := response.Data.(map[string]any)
		if !ok {
			return "", errors.New("invalid data format")
		}

		// Extract "user" field and assert it is a map
		userData, ok := dataMap["user"].(map[string]any)
		if !ok {
			return "", errors.New("missing or invalid user data")
		}

		// Extract "id" field and assert it is a string
		userID, ok = userData["id"].(string)
		if !ok {
			return "", errors.New("missing or invalid user ID")
		}
	}

	return userID, nil
}

func (app *Config) SendResetPasswordEmail(w http.ResponseWriter, r *http.Request) {

	//extract the request body
	var requestPayload ResetPasswordEmailPayload

	//extract the requestbody
	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	// Validate the request payload
	if err := app.ValidateResetPasswordEmailInput(requestPayload); len(err) > 0 {
		app.errorJSON(w, errors.New("error doing validation for send reset password email"), err, http.StatusBadRequest)
		return
	}

	//create some json we will send to authservice
	jsonData, _ := json.MarshalIndent(requestPayload, "", "\t")

	authServiceUrl := fmt.Sprintf("%s%s", os.Getenv("AUTH_URL"), "reset-password-email")

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

func (app *Config) ChangePassword(w http.ResponseWriter, r *http.Request) {

	//extract the request body
	var requestPayload ChangePasswordPayload

	//extract the requestbody
	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	// Validate the request payload
	if err := app.ValidateChangePasswordInput(requestPayload); len(err) > 0 {
		app.errorJSON(w, errors.New("error changing password"), err, http.StatusBadRequest)
		return
	}

	//create some json we will send to authservice
	jsonData, _ := json.MarshalIndent(requestPayload, "", "\t")

	authServiceUrl := fmt.Sprintf("%s%s", os.Getenv("AUTH_URL"), "change-password")

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
func (app *Config) RequestVerificationEmail(w http.ResponseWriter, r *http.Request) {

	//extract the request body
	var requestPayload RequestPasswordVerificationEmailPayload

	//extract the requestbody
	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	// Validate the request payload
	if err := app.ValidateEmailRequestInput(requestPayload); len(err) > 0 {
		app.errorJSON(w, errors.New("error validating your email"), err, http.StatusBadRequest)
		return
	}

	//create some json we will send to authservice
	jsonData, _ := json.MarshalIndent(requestPayload, "", "\t")

	authServiceUrl := fmt.Sprintf("%s%s", os.Getenv("AUTH_URL"), "request-verification-email")

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
		app.errorJSON(w, err, jsonFromService.StatusCode)
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

// func (app *Config) EGetUsers(w http.ResponseWriter, r *http.Request) {

// 	response, err := app.getToken(r)
// 	if err != nil {
// 		app.errorJSON(w, err, response.Data, http.StatusUnauthorized)
// 		return
// 	}

// 	if response.Error {
// 		app.errorJSON(w, errors.New(response.Message), response.Data, response.StatusCode)
// 		return
// 	}

// 	app.EproceedGetUser(w)
// }

// func (app *Config) EproceedGetUser(w http.ResponseWriter) {

// 	authServiceUrl := fmt.Sprintf("%s%s", os.Getenv("ELASTIC_SEARCH_SERVICE_URL"), "getusers")
// 	log.Println("The endpoint:", authServiceUrl)

// 	// Call the service by creating a request
// 	request, err := http.NewRequest("GET", authServiceUrl, nil)
// 	if err != nil {
// 		app.errorJSON(w, err, nil)
// 		return
// 	}

// 	// Set the Content-Type header
// 	request.Header.Set("Content-Type", "application/json")

// 	// Create an HTTP client
// 	client := &http.Client{}
// 	response, err := client.Do(request)
// 	if err != nil {
// 		log.Println(err)
// 		app.errorJSON(w, err, nil)
// 		return
// 	}
// 	defer response.Body.Close()

// 	// Create a variable to read response.Body into
// 	var jsonFromService jsonResponse

// 	// Decode the JSON from the service
// 	err = json.NewDecoder(response.Body).Decode(&jsonFromService)
// 	if err != nil {
// 		app.errorJSON(w, err, nil)
// 		return
// 	}

// 	// Check if the status code is Accepted
// 	if response.StatusCode != http.StatusAccepted {
// 		app.errorJSON(w, errors.New("unexpected status code received from service"), nil, response.StatusCode)
// 		return
// 	}

// 	// Prepare the payload
// 	var payload jsonResponse
// 	payload.Error = jsonFromService.Error
// 	payload.StatusCode = http.StatusOK
// 	payload.Message = jsonFromService.Message
// 	payload.Data = jsonFromService.Data

// 	// Write the JSON response
// 	app.writeJSON(w, http.StatusOK, payload)
// }

// func (app *Config) IndexInventory(w http.ResponseWriter, r *http.Request) {

// 	//extract the request body
// 	var requestPayload IndexInventoryPayload

// 	//extract the requestbody
// 	err := app.readJSON(w, r, &requestPayload)
// 	if err != nil {
// 		app.errorJSON(w, err, nil)
// 		return
// 	}

// 	// Construct query parameters for Elasticsearch
// 	elasticQueryParams := url.Values{}
// 	elasticQueryParams.Set("id", requestPayload.Id)

// 	// Build the full Elasticsearch service URL with query parameters
// 	elasticServiceUrl := fmt.Sprintf("%s%s?%s",
// 		os.Getenv("ELASTIC_SEARCH_SERVICE_URL"),
// 		"inventory/index",
// 		elasticQueryParams.Encode(),
// 	)

// 	// call the service by creating a request
// 	request, err := http.NewRequest("GET", elasticServiceUrl, nil)

// 	if err != nil {
// 		app.errorJSON(w, err, nil)
// 		return
// 	}

// 	if err != nil {
// 		log.Println(err)
// 		app.errorJSON(w, err, nil)
// 		return
// 	}

// 	// Set the Content-Type header
// 	request.Header.Set("Content-Type", "application/json")
// 	//create a http client
// 	client := &http.Client{}
// 	response, err := client.Do(request)
// 	if err != nil {
// 		log.Println(err)
// 		app.errorJSON(w, err, nil)
// 		return
// 	}
// 	defer response.Body.Close()

// 	// create a varabiel we'll read response.Body into
// 	var jsonFromService jsonResponse

// 	// decode the json from the auth service
// 	err = json.NewDecoder(response.Body).Decode(&jsonFromService)
// 	if err != nil {
// 		app.errorJSON(w, err, nil)
// 		return
// 	}

// 	if response.StatusCode != http.StatusAccepted {
// 		app.errorJSON(w, errors.New(jsonFromService.Message), nil, response.StatusCode)
// 		return
// 	}

// 	var payload jsonResponse
// 	payload.Error = jsonFromService.Error
// 	payload.StatusCode = http.StatusOK
// 	payload.Message = jsonFromService.Message
// 	payload.Data = jsonFromService.Data

// 	app.writeJSON(w, http.StatusOK, payload)
// }

type SavedInventoryPayload struct {
	UserId      string `json:"user_id"`
	InventoryId string `json:"inventory_id" binding:"required"`
}

func (app *Config) SaveInventory(w http.ResponseWriter, r *http.Request) {

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
	var requestPayload SavedInventoryPayload

	//extract the request body
	err = app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	userId := user.Data.(map[string]interface{})["user"].(map[string]interface{})["id"].(string)
	requestPayload.UserId = userId

	//create some json we will send to authservice
	jsonData, _ := json.MarshalIndent(requestPayload, "", "\t")

	invServiceUrl := fmt.Sprintf("%s%s", os.Getenv("INVENTORY_SERVICE_URL"), "save-inventory")

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

type DeleteSavedInventoryPayload struct {
	ID          string `json:"id"`
	UserId      string `json:"user_id"`
	InventoryId string `json:"inventory_id" binding:"required"`
}

func (app *Config) DeleteSaveInventory(w http.ResponseWriter, r *http.Request) {

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
	var requestPayload DeleteSavedInventoryPayload

	//extract the request body
	err = app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	userId := user.Data.(map[string]interface{})["user"].(map[string]interface{})["id"].(string)
	requestPayload.UserId = userId

	//create some json we will send to authservice
	jsonData, _ := json.MarshalIndent(requestPayload, "", "\t")

	invServiceUrl := fmt.Sprintf("%s%s", os.Getenv("INVENTORY_SERVICE_URL"), "delete-inventory")

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

type GetUserSavedInventoryReq struct {
	UserId string `json:"user_id"`
}

func (app *Config) GetUserSavedInventory(w http.ResponseWriter, r *http.Request) {

	log.Println("GOT TO THE FIRST")
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

	requestPayload := GetUserSavedInventoryReq{
		UserId: userId,
	}

	//create some json we will send to authservice
	jsonData, _ := json.MarshalIndent(requestPayload, "", "\t")

	invServiceUrl := fmt.Sprintf("%s%s", os.Getenv("INVENTORY_SERVICE_URL"), "user-saved-inventory")

	log.Println("URL", invServiceUrl)
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

//TODO
// 1. format role & permission into separate arrays - done
// 2. Try adding additional permission to a user - done
// 3. also extract the additional permission into the permissions array - done
// 4. Renters staff relationship (table renters_staff (renter_id, user_id, )) - done
// 5. Extract the user who added you

// Ask user if they have company - done
// if the company is registered or not - done
// Ask for CAC number if registered - done
// Ask company phone number
// Ask Company address (state & LGA) - done
