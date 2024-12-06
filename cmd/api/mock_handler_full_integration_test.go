//go:build integration
// +build integration

package main

import (
	"bytes"
	"context"
	"time"

	// "context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/obynonwane/rental-service-proto/inventory"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/rand"
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

	// Prepare test data (testUserID, testRaterID needs to be supplied manually from users table )

	testUserID := "90fd9f7d-e055-4351-aa14-803441103831"
	testRaterID := "90fd9f7d-e055-4351-aa14-803441103831"
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
	requestPayload := generateSignupPayload()

	// Serialize payload to JSON
	jsonData, err := json.MarshalIndent(requestPayload, "", "\t")
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	// Construct URL
	authServiceUrl := fmt.Sprintf("%s%s", os.Getenv("AUTH_URL"), "signup")

	// Create HTTP request
	request, err := http.NewRequest("POST", authServiceUrl, bytes.NewBuffer(jsonData))
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

	if response.StatusCode != http.StatusAccepted {
		t.Fatalf("Unexpected response status: %d, message: %s", response.StatusCode, jsonFromService.Message)
	}

	assert.Equal(t, http.StatusAccepted, jsonFromService.StatusCode, "returned statuscode not equal to 202")
}

// Utility function to generate random SignupPayload
func generateSignupPayload() *SignupPayload {
	rand.Seed(uint64(time.Now().UnixNano())) // Convert to uint64

	// Generate a random email using uuid
	randomUUID := uuid.New().String()
	email := fmt.Sprintf("%s@gmail.com", randomUUID[:8])

	// Generate a random phone number (example format)
	phone := fmt.Sprintf("+234816%d", rand.Intn(100000000))

	// Generate random first and last names (example)
	firstName := fmt.Sprintf("User%d", rand.Intn(1000))
	lastName := fmt.Sprintf("Name%d", rand.Intn(1000))

	// Use a fixed password or randomize it
	password := "password" // You can generate a random password here if needed

	// Return a new SignupPayload
	return &SignupPayload{
		FirstName: firstName,
		LastName:  lastName,
		Email:     email,
		Phone:     phone,
		Password:  password,
	}
}
