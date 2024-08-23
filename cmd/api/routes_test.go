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
	chiRoutes := testRoutes.(chi.Router) // cast testRoutes to type chi.Router

	//list of expected routes in the application
	routes := []string{
		"/api/v1/authentication/signup",
		"/api/v1/authentication/signup",
		"/api/v1/authentication/login",
		"/api/v1/authentication/get-me",
		"/api/v1/authentication/verify-token",
		"/api/v1/authentication/log-out",
		"/api/v1/authentication/log-out",
		"/api/v1/authentication/verify-email",
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
