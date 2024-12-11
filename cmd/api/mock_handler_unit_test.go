package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/obynonwane/rental-service-proto/inventory"
	"github.com/stretchr/testify/assert"
)

// GRPC (Inventory service) - uses unit testing (mock based testing) - using goMock
// used generated mock proto file
func TestRateUserMockgen(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock gRPC client
	mockClient := inventory.NewMockInventoryServiceClient(ctrl)

	// Simulate a frontend request's parameters
	userId := "5bce1593-c6a6-4d2d-ab6a-fd2962cffb59"
	raterId := "7a937e9d-1dc2-4e6d-ba38-d1648b05730c"
	comment := "greate product"
	rating := int32(5)

	// Define the expected behavior of the mock client
	mockClient.EXPECT().
		RateUser(gomock.Any(), &inventory.UserRatingRequest{
			UserId:  userId,
			RaterId: raterId,
			Rating:  rating,
			Comment: comment,
		}).
		Return(&inventory.UserRatingResponse{
			Id:             "6a7b83f0-30cb-4854-a32e-3576bf491858",
			UserId:         userId,
			RaterId:        raterId,
			Rating:         rating,
			Comment:        comment,
			CreatedAtHuman: "2024-11-24 10:00:00",
			UpdatedAtHuman: "2024-11-24 10:00:00",
		}, nil)

	// Simulate the client logic
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// make call to the grpc server method
	resp, err := mockClient.RateUser(ctx, &inventory.UserRatingRequest{
		UserId:  userId,
		RaterId: raterId,
		Rating:  rating,
		Comment: comment,
	})

	// Validate the response
	t.Log("Checking that err is nil...")
	assert.NoError(t, err)
	t.Log("Checking that response is nil...")
	assert.NotNil(t, resp)
	t.Log("Checking if the supplied supplied ID is same as the returned ID")
	assert.Equal(t, "6a7b83f0-30cb-4854-a32e-3576bf491858", resp.Id)
	t.Log("Checking if the supplied UserID is same as the returned UserID")
	assert.Equal(t, userId, resp.UserId)
	t.Log("Checking if the supplied RaterId is same as the returned RaterId")
	assert.Equal(t, raterId, resp.RaterId)
	t.Log("Checking if the supplied comment is same as the returned comment")
	assert.Equal(t, comment, resp.Comment)
	t.Log("Checking if the supplied Rating is not greater then 5")
	assert.LessOrEqual(t, resp.Rating, int32(5), "supplied rating is greater than 5")
	t.Log("Checking if the supplied Date is equal to the returned Date")
	assert.Equal(t, "2024-11-24 10:00:00", resp.CreatedAtHuman)
}

// HTTP - Protocol (Authservice)
func TestSignupHandler(t *testing.T) {
	// Create a mock handler
	mockHandler := &MockHandler{}

	// Define the behavior for Signup
	mockHandler.SignupFunc = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "user account created", "status_code": 200, "error": false}`))
	}

	// Create a test request
	req, err := http.NewRequest("POST", "/api/v1/authentication/signup", bytes.NewBufferString(`{"first_name": "John", "last_name": "Doe", "email": "john@example.com", "phone":"+2348162361601", "password": "password"}`))
	if err != nil {
		t.Fatal(err)
	}

	// Create a ResponseRecorder to capture the response
	rr := httptest.NewRecorder()

	// Call the handler directly
	mockHandler.Signup(rr, req)

	// Check the response
	if rr.Code != http.StatusOK {
		t.Fatalf("Expected status OK, got %v", rr.Code)
	}
	expected := `{"message": "user account created", "status_code": 200, "error": false}`
	if rr.Body.String() != expected {
		t.Fatalf("Expected response body %v, got %v", expected, rr.Body.String())
	}

}

func TestLoginHandler(t *testing.T) {
	// create a mock handler
	mockHandler := &MockHandler{}

	// Define the behaviour for Login
	mockHandler.LoginFunc = func(w http.ResponseWriter, r *http.Request) {

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "Signup successful"}`))
	}
}
