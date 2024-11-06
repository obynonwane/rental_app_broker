package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

/* returns http.Handler*/
func (app *Config) routes() http.Handler {
	mux := chi.NewRouter()
	// Redirect or clean paths with trailing slashes
	// mux.Use(middleware.RedirectSlashes)

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
	mux.Post("/api/v1/authentication/admin/signup", app.SignupAdmin)
	mux.Post("/api/v1/authentication/login", app.Login)
	mux.Get("/api/v1/authentication/get-me", app.GetMe)
	mux.Get("/api/v1/authentication/verify-token", app.VerifyToken)
	mux.Post("/api/v1/authentication/log-out", app.Logout)
	mux.Get("/api/v1/authentication/verify-email", app.VerifyEmail)
	mux.Post("/api/v1/authentication/choose-role", app.ChooseRole)
	mux.Get("/api/v1/authentication/product-owner-permissions", app.ProductOwnerPermission)
	mux.Post("/api/v1/authentication/product-owner-create-staff", app.ProductOwnerCreateStaff)
	mux.Post("/api/v1/authentication/assign-permission", app.ProductOwnerAssignPermission)
	mux.Get("/api/v1/authentication/countries", app.GetCountries)
	mux.Get("/api/v1/authentication/states", app.GetStates)
	mux.Get("/api/v1/authentication/lgas", app.GetLgas)
	mux.Get("/api/v1/authentication/country/state/{id}", app.GetCountryState)
	mux.Get("/api/v1/authentication/state/lgas/{id}", app.GetStateLga)
	mux.Post("/api/v1/authentication/kyc-renter", app.KycRenter)
	mux.Post("/api/v1/authentication/kyc-participant", app.KycBusiness)
	mux.Get("/api/v1/authentication/retrieve-identification-types", app.RetriveIdentificationTypes)
	mux.Get("/api/v1/authentication/list-user-type", app.ListUserTypes)

	mux.Post("/", app.Subscription)

	mux.Post("/api/v1/send-email", app.TestEmail)

	//Inventory routes---------------------------------------------------//
	mux.Get("/api/v1/inventory/getusers", app.GetUsers)

	// Add the Prometheus metrics endpoint to the router-----------------//
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}
