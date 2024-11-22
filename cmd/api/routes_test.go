package main

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
)

func Test_routes_exist(t *testing.T) {

	// create instance of Config struct
	testApp := Config{}

	// calls the routes method on testApp to obtain the application routes
	// chi router have method walks on its type, that lets you walk through the routes to make sure they exist
	testRoutes := testApp.routes()
	chiRoutes := testRoutes.(chi.Router) // cast testRoutes to type chi.Router nice

	//list of expected routes in the application - slice of strings
	routes := []string{
		"/",
		"/api/v1/authentication/signup",
		"/api/v1/authentication/admin/signup",
		"/api/v1/authentication/login",
		"/api/v1/authentication/get-me",
		"/api/v1/authentication/verify-token",
		"/api/v1/authentication/log-out",
		"/api/v1/authentication/verify-email",
		"/api/v1/authentication/participant-create-staff",
		"/api/v1/authentication/countries",
		"/api/v1/authentication/states",
		"/api/v1/authentication/lgas",
		"/api/v1/authentication/country/state/{id}",
		"/api/v1/authentication/state/lgas/{id}",
		"/api/v1/authentication/kyc-renter",
		"/api/v1/authentication/kyc-business",
		"/api/v1/authentication/retrieve-identification-types",
		"/api/v1/authentication/list-user-type",
		"/api/v1/authentication/test-rpc",
		"/api/v1/send-email",
		"/api/v1/inventory/getusers",
		"/api/v1/inventory/create-inventory",
		"/api/v1/inventory/getusers-grpc",
		"/api/v1/inventory/all-categories",
		"/api/v1/inventory/all-subcategories",
		"/api/v1/inventory/category/subcategory/{id}",
		"/api/v1/inventory/category/{id}",
		"/api/v1/inventory/rating",
		"/api/v1/inventory/rating-user",
		"/metrics",
	}

	// loops through above list calling routesExist to verify their existance
	for _, route := range routes {
		routesExist(t, chiRoutes, route)
	}
}

// routesExist: This function checks if a particular route exists within the chi.Router
func routesExist(t *testing.T, routes chi.Router, route string) {
	found := false

	// chi.Walk: Walks through the routes defined in the router, applying the provided function to each route.
	_ = chi.Walk(routes, func(method string, foundRoute string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		if route == foundRoute {
			found = true
		}
		return nil
	})

	if !found {
		t.Errorf("did not find %s in registered routes", route)
	}
}
