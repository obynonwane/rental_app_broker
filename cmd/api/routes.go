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
	mux.Post("/api/v1/authentication/subdomain-exist", app.SubdomainExist)
	mux.Get("/api/v1/authentication/retrieve-identification-types", app.RetriveIdentificationTypes)
	mux.Get("/api/v1/authentication/retrieve-industries", app.RetriveIndustries)
	mux.Get("/api/v1/authentication/list-user-type", app.ListUserTypes)
	mux.Post("/api/v1/authentication/reset-password-email", app.SendResetPasswordEmail)
	mux.Post("/api/v1/authentication/change-password", app.ChangePassword)
	mux.Post("/api/v1/authentication/request-verification-email", app.RequestVerificationEmail)

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
	mux.Get("/api/v1/inventory/category", app.GetCategoryByID)
	mux.Post("/api/v1/inventory/rating", app.RateInventory)
	mux.Post("/api/v1/inventory/rating-user", app.RateUser)
	mux.Get("/api/v1/inventory/inventory-detail", app.GetInventoryDetail)
	mux.Get("/api/v1/inventory/user-detail", app.GetUserDetail)
	mux.Get("/api/v1/inventory/user-rating/{id}", app.GetUserRatings)
	mux.Get("/api/v1/inventory/inventory-rating/{id}", app.GetInventoryRatings)
	mux.Post("/api/v1/inventory/reply-inventory-rating", app.ReplyInventoryRating)
	mux.Post("/api/v1/inventory/reply-user-rating", app.ReplyUserRating)

	mux.Get("/api/v1/inventory/user-rating-helpful/{id}", app.UserRatingHepful)
	mux.Get("/api/v1/inventory/report-user-rating/{id}", app.ReportUserRating)

	mux.Get("/api/v1/inventory/inventory-rating-helpful/{id}", app.InventoryRatingHepful)
	mux.Get("/api/v1/inventory/report-inventory-rating/{id}", app.ReportInventoryRating)

	mux.Post("/api/v1/inventory/search", app.SearchInventory)
	mux.Post("/api/v1/inventory/premium-partners", app.PremiumPartner)
	mux.Post("/api/v1/inventory/save-inventory", app.SaveInventory)
	mux.Post("/api/v1/inventory/delete-saved-inventory", app.DeleteSaveInventory)
	mux.Post("/api/v1/inventory/inventory-availability", app.MarkInventoryAvailability)
	mux.Get("/api/v1/inventory/delete-inventory/{id}", app.DeleteInventory)
	mux.Get("/api/v1/inventory/user-saved-inventory", app.GetUserSavedInventory)
	mux.Get("/api/v1/inventory/premium-extras", app.GetPremiumUsersExtras)

	mux.Get("/api/v1/inventory/inventory-rating-replies", app.GetInventoryRatingReplies)
	mux.Get("/api/v1/inventory/user-rating-replies", app.GetUserRatingReplies)

	mux.Get("/api/v1/business/details", app.GetBusinessDetail)

	//Booking routes---------------------------------------------------//
	mux.Post("/api/v1/booking/create-booking", app.CreateBooking)
	mux.Get("/api/v1/booking/my-booking", app.MyBookings)
	mux.Get("/api/v1/booking/booking-requests", app.GetBookingRequest)
	mux.Get("/api/v1/booking/pending-booking-count", app.GetPendingBookingCount)

	mux.Get("/api/v1/purchase/pending-purchase-count", app.GetPendingPurchaseCount)

	mux.Post("/api/v1/purchase/create-order", app.CreatePrurchaseOrder)
	mux.Get("/api/v1/purchase/my-purchase", app.MyPurchase)
	mux.Get("/api/v1/purchase/purchase-requests", app.GetPurchaseRequest)

	mux.Get("/api/v1/inventory/my-inventories", app.MyInventories)

	mux.Get("/api/v1/subscription/my-subscription-history", app.MySubscriptionHistory)

	//Chat  routes---------------------------------------------------//
	mux.Get("/api/v1/chat/ws", app.ChatHandler)
	mux.Get("/api/v1/chat/chat-history", app.GetChatHistory)
	mux.Get("/api/v1/chat/chat-list", app.GetChatList)
	mux.Get("/api/v1/chat/unread-chat", app.GetUnreadChat)
	mux.Get("/api/v1/chat/mark-chat-as-read", app.MarkChatAsRead)
	mux.Post("/api/v1/chat/delete-chat", app.DeleteChat)

	// Profile routes-----------------------------------------------//
	mux.Post("/api/v1/authentication/profile-image", app.UploadProfileImage)
	mux.Post("/api/v1/authentication/shop-banner", app.UploadBanner)

	//Elastic search routes---------------------------------------------------//
	// mux.Get("/api/v1/elastic-search/getusers", app.EGetUsers)
	// mux.Get("/api/v1/elastic-search/inventory", app.SearchInventory)
	// mux.Post("/api/v1/elastic-search/index", app.IndexInventory)

	// Add the Prometheus metrics endpoint to the router-----------------//
	mux.Handle("/metrics", promhttp.Handler())

	// payment gateway
	mux.Post("/api/v1/subscription/paystack-transaction-initialization", app.PaystackTransactionInitialization)
	mux.Post("/api/v1/subscription/verify-paystack-transaction", app.VerifyPaystackTransaction)
	mux.Get("/api/v1/subscription/cancel-subscription", app.CancelSubscription)
	mux.Get("/api/v1/subscription/activate-subscription", app.ActivateSubscription)
	mux.Get("/api/v1/subscription/subscription-history", app.GetSubscriptionHistory)
	mux.Get("/api/v1/subscription/plans", app.GetPlans)

	return mux
}
