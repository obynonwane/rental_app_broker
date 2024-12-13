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
	mux.Post("/api/v1/authentication/participant-create-staff", app.ParticipantCreateStaff)
	mux.Get("/api/v1/authentication/countries", app.GetCountries)
	mux.Get("/api/v1/authentication/states", app.GetStates)
	mux.Get("/api/v1/authentication/lgas", app.GetLgas)
	mux.Get("/api/v1/authentication/country/state/{id}", app.GetCountryState)
	mux.Get("/api/v1/authentication/state/lgas/{id}", app.GetStateLga)
	mux.Post("/api/v1/authentication/kyc-renter", app.KycRenter)
	mux.Post("/api/v1/authentication/kyc-business", app.KycBusiness)
	mux.Get("/api/v1/authentication/retrieve-identification-types", app.RetriveIdentificationTypes)
	mux.Get("/api/v1/authentication/list-user-type", app.ListUserTypes)

	mux.Get("/api/v1/authentication/test-rpc", app.testRPC)

	mux.Post("/", app.Subscription)

	mux.Post("/api/v1/send-email", app.TestEmail)

	//Inventory routes---------------------------------------------------//
	mux.Get("/api/v1/inventory/getusers", app.GetUsers)
	mux.Post("/api/v1/inventory/create-inventory", app.CreateInventory)
	mux.Get("/api/v1/inventory/getusers-grpc", app.GetUsersViaGrpc)
	mux.Get("/api/v1/inventory/all-categories", app.AllCategories)
	mux.Get("/api/v1/inventory/all-subcategories", app.AllSubcategories)
	mux.Get("/api/v1/inventory/category/subcategory/{id}", app.GetCategorySubcategories)
	mux.Get("/api/v1/inventory/category/{id}", app.GetCategoryByID)
	mux.Post("/api/v1/inventory/rating", app.RateInventory)
	mux.Post("/api/v1/inventory/rating-user", app.RateUser)
	mux.Get("/api/v1/inventory/inventory-detail/{id}", app.GetInventoryDetail)
	mux.Get("/api/v1/inventory/user-rating/{id}", app.GetUserRatings)
	mux.Get("/api/v1/inventory/inventory-rating/{id}", app.GetInventoryRatings)
	mux.Post("/api/v1/inventory/inventory-rating", app.ReplyInventoryRating)
	mux.Post("/api/v1/inventory/inventory-rating", app.ReplyUserRating)

	// Add the Prometheus metrics endpoint to the router-----------------//
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}
