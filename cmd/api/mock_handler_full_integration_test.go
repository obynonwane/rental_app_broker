package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/obynonwane/rental-service-proto/inventory"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestRateUserIntegration(t *testing.T) {
	// Connect to the gRPC server (inventory service)
	conn, err := grpc.Dial("inventory-service:50001", grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NoError(t, err)
	defer conn.Close()

	// Create a gRPC client
	client := inventory.NewInventoryServiceClient(conn)

	// Prepare test data
	testUserID := "test-user-id"
	testRaterID := "test-rater-id"
	testComment := "Excellent service!"
	testRating := int32(5)

	// Create a gRPC context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Make the gRPC call
	resp, err := client.RateUser(ctx, &inventory.UserRatingRequest{
		UserId:  testUserID,
		RaterId: testRaterID,
		Rating:  testRating,
		Comment: testComment,
	})

	// Assertions
	assert.NoError(t, err, "Expected no error from RateUser call")
	assert.NotNil(t, resp, "Expected a response from RateUser call")
	assert.Equal(t, testUserID, resp.UserId, "UserId should match the input")
	assert.Equal(t, testRaterID, resp.RaterId, "RaterId should match the input")
	assert.Equal(t, testComment, resp.Comment, "Comment should match the input")
	assert.Equal(t, testRating, resp.Rating, "Rating should match the input")
	assert.NotEmpty(t, resp.Id, "Response ID should not be empty")
	assert.NotEmpty(t, resp.CreatedAtHuman, "Response should have a CreatedAt timestamp")

	// Optional: Verify data in the test database (if applicable)
	// This requires a test database connection and queries.
}

func TestSignupIntegration(t *testing.T) {
	// Request payload
	requestPayload := &SignupPayload{
		FirstName: "obinna",
		LastName:  "johnson",
		Email:     "obinna@gmail.com",
		Phone:     "+2348167365693",
		Password:  "password",
	}

	// Serialize payload to JSON
	jsonData, err := json.MarshalIndent(requestPayload, "", "\t")
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	// Construct URL
	// authURL := os.Getenv("AUTH_URL")
	authURL := `http://authentication-service/api/v1/authentication/signup`
	// if authURL == "" {
	// 	t.Fatalf("AUTH_URL is not set or empty")
	// }
	// authServiceUrl := fmt.Sprintf("%s%s", authURL, "signup")

	// log.Printf("Request URL: %s", authServiceUrl)

	// Create HTTP request
	request, err := http.NewRequest("POST", authURL, bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatalf("Failed to create HTTP request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")

	// Send HTTP request
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer response.Body.Close()

	// Parse response
	var jsonFromService jsonResponse
	err = json.NewDecoder(response.Body).Decode(&jsonFromService)
	if err != nil {
		t.Fatalf("Failed to decode JSON response: %v", err)
	}

	log.Printf("Response from auth service: %+v", jsonFromService)

	if response.StatusCode != http.StatusAccepted {
		t.Fatalf("Unexpected response status: %d, message: %s", response.StatusCode, jsonFromService.Message)
	}
}
