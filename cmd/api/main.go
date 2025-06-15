package main

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"time"

	"github.com/obynonwane/broker-service/cmd/redis_client"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
)

const webPort = "8080"

type Config struct {
	cache  *redis.Client
	Rabbit *amqp.Connection
}

// Ensure Config implements the Handler interface (all methods in interface)
// this is a compile time check, just for safety
// the _ is to tell go compiler i wont be needing to use the Handler variable
var _ Handler = &Config{}

func main() {

	// try to connect to rabbitmq
	rabbitConn, err := connect()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	defer rabbitConn.Close()

	cache, err := redis_client.NewRedisClient()
	if err != nil {
		log.Fatalf("Could not connect to Redis: %v", err)
	}
	app := Config{
		cache:  cache,
		Rabbit: rabbitConn,
	}

	// websocket- chat handling
	go app.HandleMessages()

	// Start collecting system metrics in the background
	go CollectSystemMetrics()

	log.Printf("starting broker service on port %s\n", webPort)
	//define http server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", webPort),
		Handler: app.routes(),
	}

	//start the server
	err = srv.ListenAndServe()
	if err != nil {
		log.Panic(err)
	}
}

func connect() (*amqp.Connection, error) {
	var counts int64
	var backOff = 1 * time.Second
	var connection *amqp.Connection

	// don't continue until rabbit is ready
	for {
		c, err := amqp.Dial("amqp://user:password@rabbitmq:5672")
		if err != nil {
			fmt.Println("RabbitMQ not yet ready...")
			counts++
		} else {
			log.Println("Connected to RabbitMQ!")
			connection = c
			break
		}

		if counts > 5 {
			fmt.Println(err)
			return nil, err
		}

		backOff = time.Duration(math.Pow(float64(counts), 2)) * time.Second
		log.Println("backing off...")
		time.Sleep(backOff)
		continue
	}

	return connection, nil
}
