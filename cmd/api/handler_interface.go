package main

import "net/http"

type Handler interface {
	Signup(w http.ResponseWriter, r *http.Request)
	Login(w http.ResponseWriter, r *http.Request)	
	SignupAdmin(w http.ResponseWriter, r *http.Request)	
	GetMe(w http.ResponseWriter, r *http.Request)	
	VerifyToken(w http.ResponseWriter, r *http.Request)	
	Logout(w http.ResponseWriter, r *http.Request)	
	VerifyEmail(w http.ResponseWriter, r *http.Request)	
	ParticipantCreateStaff(w http.ResponseWriter, r *http.Request)	
	GetCountries(w http.ResponseWriter, r *http.Request)	
	GetStates(w http.ResponseWriter, r *http.Request)	
	GetLgas(w http.ResponseWriter, r *http.Request)	
	GetCountryState(w http.ResponseWriter, r *http.Request)	
	GetStateLga(w http.ResponseWriter, r *http.Request)	
	KycRenter(w http.ResponseWriter, r *http.Request)	
	KycBusiness(w http.ResponseWriter, r *http.Request)	
	RetriveIdentificationTypes(w http.ResponseWriter, r *http.Request)	
	ListUserTypes(w http.ResponseWriter, r *http.Request)	
	testRPC(w http.ResponseWriter, r *http.Request)	
	Subscription(w http.ResponseWriter, r *http.Request)	
	TestEmail(w http.ResponseWriter, r *http.Request)	
	GetUsers(w http.ResponseWriter, r *http.Request)	
	CreateInventory(w http.ResponseWriter, r *http.Request)	
	GetUsersViaGrpc(w http.ResponseWriter, r *http.Request)	
	AllCategories(w http.ResponseWriter, r *http.Request)	
	AllSubcategories(w http.ResponseWriter, r *http.Request)	
	GetCategorySubcategories(w http.ResponseWriter, r *http.Request)	
	GetCategoryByID(w http.ResponseWriter, r *http.Request)	
	RateInventory(w http.ResponseWriter, r *http.Request)	
	RateUser(w http.ResponseWriter, r *http.Request)	

}
