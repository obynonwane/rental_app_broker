package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/obynonwane/broker-service/cmd/redis_client"
	"github.com/redis/go-redis/v9"
)

const webPort = "8080"

type Config struct {
	cache *redis.Client
}

// Ensure Config implements the Handler interface (all methods in interface)
// this is a compile time check, just for safety
// the _ is to tell go compiler i wont be needing to use the Handler variable
var _ Handler = &Config{}

func main() {
	cache, err := redis_client.NewRedisClient()
	if err != nil {
		log.Fatalf("Could not connect to Redis: %v", err)
	}
	app := Config{
		cache: cache,
	}

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
