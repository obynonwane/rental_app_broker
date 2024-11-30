// mock_handler.go
package main

import (
	"net/http"
)

// MockHandler is a mock implementation of the Handler interface
type MockHandler struct {
	SignupFunc                 func(w http.ResponseWriter, r *http.Request)
	LoginFunc                  func(w http.ResponseWriter, r *http.Request)
	SignupAdmin                func(w http.ResponseWriter, r *http.Request)
	GetMe                      func(w http.ResponseWriter, r *http.Request)
	VerifyToken                func(w http.ResponseWriter, r *http.Request)
	Logout                     func(w http.ResponseWriter, r *http.Request)
	VerifyEmail                func(w http.ResponseWriter, r *http.Request)
	ParticipantCreateStaff     func(w http.ResponseWriter, r *http.Request)
	GetCountries               func(w http.ResponseWriter, r *http.Request)
	GetStates                  func(w http.ResponseWriter, r *http.Request)
	GetLgas                    func(w http.ResponseWriter, r *http.Request)
	GetCountryState            func(w http.ResponseWriter, r *http.Request)
	GetStateLga                func(w http.ResponseWriter, r *http.Request)
	KycRenter                  func(w http.ResponseWriter, r *http.Request)
	KycBusiness                func(w http.ResponseWriter, r *http.Request)
	RetriveIdentificationTypes func(w http.ResponseWriter, r *http.Request)
	ListUserTypes              func(w http.ResponseWriter, r *http.Request)
	testRPC                    func(w http.ResponseWriter, r *http.Request)
	Subscription               func(w http.ResponseWriter, r *http.Request)
	TestEmail                  func(w http.ResponseWriter, r *http.Request)
	GetUsers                   func(w http.ResponseWriter, r *http.Request)
	CreateInventory            func(w http.ResponseWriter, r *http.Request)
	GetUsersViaGrpc            func(w http.ResponseWriter, r *http.Request)
	AllCategories              func(w http.ResponseWriter, r *http.Request)
	AllSubcategories           func(w http.ResponseWriter, r *http.Request)
	GetCategorySubcategories   func(w http.ResponseWriter, r *http.Request)
	GetCategoryByID            func(w http.ResponseWriter, r *http.Request)
	RateInventory              func(w http.ResponseWriter, r *http.Request)
	RateUser                   func(w http.ResponseWriter, r *http.Request)
}

func (m *MockHandler) Signup(w http.ResponseWriter, r *http.Request) {
	if m.SignupFunc != nil {
		m.SignupFunc(w, r)
	}
}

func (m *MockHandler) Login(w http.ResponseWriter, r *http.Request) {
	if m.LoginFunc != nil {
		m.LoginFunc(w, r)
	}
}
