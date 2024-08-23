package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

/* returns http.Handler*/
func (app *Config) routes() http.Handler {
	mux := chi.NewRouter()

	//specify who is allowed to connect
	mux.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	mux.Use(middleware.Heartbeat("/ping"))
	mux.Post("/api/v1/authentication/signup", app.Signup)
	mux.Post("/api/v1/authentication/login", app.Login)
	mux.Get("/api/v1/authentication/get-me", app.GetMe)
	mux.Get("/api/v1/authentication/verify-token", app.VerifyToken)
	mux.Post("/api/v1/authentication/log-out", app.Logout)
	mux.Get("/api/v1/authentication/verify-email", app.VerifyEmail)

	mux.Post("/api/v1/send-email", app.TestEmail)
	mux.Post("/", app.Subscription)

	//Inventory routes---------------------------------------------------//
	mux.Get("/api/v1/inventory/getusers", app.GetUsers)

	return mux
}
