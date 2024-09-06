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
