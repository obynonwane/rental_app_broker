// package main

// import (
// 	"log"
// 	"net/http"
// 	"sync"
// 	"time"

// 	"github.com/gorilla/websocket"
// )

// var upgrader = websocket.Upgrader{
// 	ReadBufferSize:  1024,
// 	WriteBufferSize: 1024,
// 	CheckOrigin: func(r *http.Request) bool {
// 		return true
// 	},
// }

// type Message struct {
// 	Content  string `json:"content"`
// 	Sender   string `json:"sender"`
// 	Receiver string `json:"receiver"`
// 	SentAt   int64  `json:"sent_at"` // unix millis
// }

// var clients = make(map[*websocket.Conn]bool)
// var broadcast = make(chan Message, 128)

// func handleMessage() {
// 	for {
// 		msg := <-broadcast // Receive a new message from any client

// 		log.Printf("[BROADCAST] %s: %s: %s", msg.Sender, msg.Receiver, msg.Content)

// 		// Save to Redis list
// 		// err := app.cache.RPush(context.Background(), "chat:messages", fmt.Sprintf("%s: %s", msg.User, msg.Content)).Err()
// 		// if err != nil {
// 		// 	log.Printf("Error saving message to Redis: %v", err)
// 		// }

// 		for client := range clients {
// 			go func(c *websocket.Conn) {
// 				if err := c.WriteJSON(msg); err != nil {
// 					c.Close()
// 					delete(clients, c)
// 				}
// 			}(client)
// 		}
// 	}
// }

// func (app *Config) ChatHandler(w http.ResponseWriter, r *http.Request) {
// 	conn, err := upgrader.Upgrade(w, r, nil)
// 	if err != nil {
// 		log.Printf("upgrade failed: %v", err)
// 		return
// 	}

// 	clients[conn] = true

// 	// Keep‑alive pings
// 	go func(c *websocket.Conn) {
// 		ticker := time.NewTicker(15 * time.Second)
// 		defer ticker.Stop()
// 		for range ticker.C {
// 			if err := c.WriteMessage(websocket.PingMessage, nil); err != nil {
// 				c.Close()
// 				delete(clients, c)
// 				return
// 			}
// 		}
// 	}(conn)

// 	for {
// 		var msg Message
// 		if err := conn.ReadJSON(&msg); err != nil {
// 			delete(clients, conn)
// 			break
// 		}
// 		msg.SentAt = time.Now().UnixMilli()
// 		broadcast <- msg
// 	}
// }

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
	Content  string `json:"content"`
	Sender   string `json:"sender"`
	Receiver string `json:"receiver"`
	SentAt   int64  `json:"sent_at"`
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
		log.Printf("[MESSAGE] %s → %s: %s", msg.Sender, msg.Receiver, msg.Content)

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
	log.Printf("[DB SAVE] From %s to %s at %d: %s", msg.Sender, msg.Receiver, msg.SentAt, msg.Content)

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
