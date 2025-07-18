package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/obynonwane/broker-service/event"
)

type RabbitMQPayload struct {
	Name string `json:"name"`
	// Data map[string]interface{} `json:"data"`
	// Data Message `json:"data"`
	Data json.RawMessage `json:"data"` // Raw JSON until decoded
}

// Message struct defines the message payload
type Message struct {
	Content      string `json:"content"`
	Sender       string `json:"sender"`
	ReplyTo      string `json:"reply_to"`
	Receiver     string `json:"receiver"`
	SentAt       int64  `json:"sent_at"`
	Content_Type string `json:"content_type"`
	MessageID    string `json:"message_id"`
}

// Map of userID to websocket.Conn
var clients = make(map[string]*websocket.Conn)
var clientsMu sync.Mutex // for safe concurrent access

var broadcast = make(chan Message, 128)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // allow all origins for testing
	},
}

func GenerateUUID() string {
	return uuid.NewString()
}

// Main entry point

// chatHandler handles new WebSocket connections
func (app *Config) ChatHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade failed: %v", err)
		return
	}

	// Register user
	clientsMu.Lock()
	clients[userID] = conn
	clientsMu.Unlock()

	log.Printf("[CONNECT] User %s connected", userID)

	// Start pinging the connection
	go keepAlive(conn, userID)

	// Listen for incoming messages
	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("[DISCONNECT] %s: %v", userID, err)
			break
		}
		msg.SentAt = time.Now().UnixMilli()
		msg.MessageID = GenerateUUID()
		broadcast <- msg
	}

	// Cleanup
	clientsMu.Lock()
	delete(clients, userID)
	clientsMu.Unlock()
	conn.Close()
	log.Printf("[CLEANUP] %s disconnected", userID)
}

// keepAlive sends periodic ping messages to keep the connection alive
func keepAlive(conn *websocket.Conn, userID string) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			log.Printf("[PING FAILED] %s: %v", userID, err)
			break
		}
	}
}

// handleMessages routes messages to the intended receiver
func (app *Config) HandleMessages() {
	for msg := range broadcast {
		log.Printf("[MESSAGE] %s → %s: %s -> %s -> %s", msg.Sender, msg.Receiver, msg.Content, msg.ReplyTo, msg.MessageID)

		app.saveToDatabase(msg)


		// Send to receiver
		clientsMu.Lock()
		receiverConn, ok := clients[msg.Receiver]
		senderConn, senderOnline := clients[msg.Sender]
		clientsMu.Unlock()

		if ok {
			go safeSend(receiverConn, msg)
		}

		// Optional: echo back to sender
		if senderOnline {
			go safeSend(senderConn, msg)
		}
	}
}

// safeSend handles errors while writing to connections
func safeSend(conn *websocket.Conn, msg Message) {
	if err := conn.WriteJSON(msg); err != nil {
		log.Printf("[SEND ERROR] %v", err)
		conn.Close()
	}
}

func (app *Config) saveToDatabase(msg Message) {

	// Example: log to console
	log.Printf("[DB SAVE] From %s to %s at %d: %s %s", msg.Sender, msg.Receiver, msg.SentAt, msg.Content, msg.MessageID)

	rawData, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal message: %v", err)
		return
	}

	data := RabbitMQPayload{
		Name: "persist_chat",
		Data: json.RawMessage(rawData),
	}

	go app.pushEventViaRabbit(data)
}

func (app *Config) GetChatHistory(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	userA := queryParams.Get("userA")
	userB := queryParams.Get("userB")

	// Verify the user token
	user, err := app.getToken(r)
	if err != nil {
		app.errorJSON(w, err, user.Data, http.StatusUnauthorized)
		return
	}

	if user.Error {
		app.errorJSON(w, errors.New(user.Message), user.Data, user.StatusCode)
		return
	}

	// Define the payload structure
	type ChatHistoryRequest struct {
		UserA string `json:"userA"`
		UserB string `json:"userB"`
	}

	requestPayload := ChatHistoryRequest{
		UserA: userA,
		UserB: userB,
	}

	// Marshal request payload to JSON
	jsonData, err := json.Marshal(requestPayload)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	// Construct inventory service URL
	invServiceUrl := fmt.Sprintf("%s%s", os.Getenv("INVENTORY_SERVICE_URL"), "chat-history")

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

func (app *Config) GetChatList(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	userID := queryParams.Get("userId")

	// Verify the user token
	user, err := app.getToken(r)
	if err != nil {
		app.errorJSON(w, err, user.Data, http.StatusUnauthorized)
		return
	}

	if user.Error {
		app.errorJSON(w, errors.New(user.Message), user.Data, user.StatusCode)
		return
	}

	// Define the payload structure

	type ChatListRequest struct {
		UserID string `json:"user_id"`
	}

	requestPayload := ChatListRequest{
		UserID: userID,
	}

	// Marshal request payload to JSON
	jsonData, err := json.Marshal(requestPayload)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	// Construct inventory service URL
	invServiceUrl := fmt.Sprintf("%s%s", os.Getenv("INVENTORY_SERVICE_URL"), "chat-list")

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

func (app *Config) GetUnreadChat(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	userID := queryParams.Get("userId")

	// Verify the user token
	user, err := app.getToken(r)
	if err != nil {
		app.errorJSON(w, err, user.Data, http.StatusUnauthorized)
		return
	}

	if user.Error {
		app.errorJSON(w, errors.New(user.Message), user.Data, user.StatusCode)
		return
	}

	// Define the payload structure

	type UnreadChatRequest struct {
		UserID string `json:"user_id"`
	}

	requestPayload := UnreadChatRequest{
		UserID: userID,
	}

	// Marshal request payload to JSON
	jsonData, err := json.Marshal(requestPayload)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	// Construct inventory service URL
	invServiceUrl := fmt.Sprintf("%s%s", os.Getenv("INVENTORY_SERVICE_URL"), "unread-chat")

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

func (app *Config) MarkChatAsRead(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	userID := queryParams.Get("senderId")

	// Verify the user token
	user, err := app.getToken(r)
	if err != nil {
		app.errorJSON(w, err, user.Data, http.StatusUnauthorized)
		return
	}

	if user.Error {
		app.errorJSON(w, errors.New(user.Message), user.Data, user.StatusCode)
		return
	}

	replierID, err := app.returnLoggedInUserID(user)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	// Define the payload structure

	type MarkChatRequest struct {
		UserID   string `json:"user_id"`
		SenderID string `json:"sender_id"`
	}

	requestPayload := MarkChatRequest{
		UserID:   replierID,
		SenderID: userID,
	}

	// Marshal request payload to JSON
	jsonData, err := json.Marshal(requestPayload)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	// Construct inventory service URL
	invServiceUrl := fmt.Sprintf("%s%s", os.Getenv("INVENTORY_SERVICE_URL"), "mark-chat-as-read")

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

type DeleteChatPayload struct {
	ID     string `json:"id" binding:"required"`
	UserId string `json:"user_id"`
}

func (app *Config) DeleteChat(w http.ResponseWriter, r *http.Request) {

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
	var requestPayload DeleteChatPayload

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

	invServiceUrl := fmt.Sprintf("%s%s", os.Getenv("INVENTORY_SERVICE_URL"), "delete-chat")

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

//==============================================================================================================================================//

// pushEventViaRabbit logs an event using the logger-service. It makes the call by pushing the data to RabbitMQ.
func (app *Config) pushEventViaRabbit(l RabbitMQPayload) {
	err := app.pushToQueue(l.Name, l.Data)
	if err != nil {
		// app.errorJSON(w, err, nil)
		return
	}
}

// pushToQueue pushes a message into RabbitMQ
func (app *Config) pushToQueue(name string, msg json.RawMessage) error {
	emitter, err := event.NewEventEmitter(app.Rabbit)
	if err != nil {
		return err
	}

	payload := RabbitMQPayload{
		Name: name,
		Data: msg,
	}

	j, _ := json.MarshalIndent(&payload, "", "\t")
	err = emitter.Push(string(j), "log.INFO")
	if err != nil {
		return err
	}
	return nil
}
