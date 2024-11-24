// handlers_test.go
package main

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/obynonwane/rental-service-proto/inventory"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener

func init() {
	lis = bufconn.Listen(bufSize)
	server := grpc.NewServer()
	inventory.RegisterInventoryServiceServer(server, &mockInventoryServer{})
	go func() {
		if err := server.Serve(lis); err != nil {
			panic(err)
		}
	}()
}

func bufDialer(ctx context.Context, address string) (net.Conn, error) {
	return lis.Dial() // Return the connection directly
}

// Mock server implementation for the Bufconn test
type mockInventoryServer struct {
	inventory.UnimplementedInventoryServiceServer
}

func (m *mockInventoryServer) RateUser(ctx context.Context, req *inventory.UserRatingRequest) (*inventory.UserRatingResponse, error) {
	return &inventory.UserRatingResponse{
		Id:             "rating-123",
		UserId:         req.UserId,
		RaterId:        req.RaterId,
		Rating:         req.Rating,
		Comment:        req.Comment,
		CreatedAtHuman: "2024-11-24 10:00:00",
		UpdatedAtHuman: "2024-11-24 10:00:00",
	}, nil
}

func TestRateUserBufconn(t *testing.T) {
	// Set up gRPC connection with Bufconn
	grpcConn, err := grpc.DialContext(context.Background(), "", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	assert.NoError(t, err)
	defer grpcConn.Close()

	// Instantiate the gRPC client
	client := inventory.NewInventoryServiceClient(grpcConn)

	// Simulate a frontend request's parameters
	userId := "123"
	raterId := "456"
	comment := "Great"
	rating := int32(5)

	// Make the gRPC call
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.RateUser(ctx, &inventory.UserRatingRequest{
		UserId:  userId,
		RaterId: raterId,
		Rating:  rating,
		Comment: comment,
	})

	// Validate the response
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "rating-123", resp.Id)
	assert.Equal(t, userId, resp.UserId)
	assert.Equal(t, raterId, resp.RaterId)
	assert.Equal(t, comment, resp.Comment)
	assert.Equal(t, "2024-11-24 10:00:00", resp.CreatedAtHuman)
}

func TestRateUserMockgen(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock gRPC client
	mockClient := inventory.NewMockInventoryServiceClient(ctrl)

	// Simulate a frontend request's parameters
	userId := "123"
	raterId := "456"
	comment := "Great"
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
			Id:             "rating-123",
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

	resp, err := mockClient.RateUser(ctx, &inventory.UserRatingRequest{
		UserId:  userId,
		RaterId: raterId,
		Rating:  rating,
		Comment: comment,
	})

	// Validate the response
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "rating-123", resp.Id)
	assert.Equal(t, userId, resp.UserId)
	assert.Equal(t, raterId, resp.RaterId)
	assert.Equal(t, comment, resp.Comment)
	assert.Equal(t, "2024-11-24 10:00:00", resp.CreatedAtHuman)
}
