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

// Mock server implementation for the Bufconn test
type mockInventoryServer struct {
	inventory.UnimplementedInventoryServiceServer
}

// initialise a memory size of bufcoon in-memory listener
const bufSize = 1024 * 1024

// create a bufcoon listener of type *bucoon.Listener
var lis *bufconn.Listener

func init() {
	// initializes the inm-memory listener
	lis = bufconn.Listen(bufSize)

	// create a new instance of gRPC server
	server := grpc.NewServer()

	// Registers a mock server implementing
	inventory.RegisterInventoryServiceServer(server, &mockInventoryServer{})
	go func() {
		if err := server.Serve(lis); err != nil {
			panic(err)
		}
	}()
}

// dials the bufconn listener instead of a real network connection
func bufDialer(ctx context.Context, address string) (net.Conn, error) {
	return lis.Dial() // Return the connection directly
}

// Uses Integration testing - Bufcoon based testing (gRPC  mocking the server side)
func (m *mockInventoryServer) RateUser(ctx context.Context, req *inventory.UserRatingRequest) (*inventory.UserRatingResponse, error) {
	return &inventory.UserRatingResponse{
		Id:             "15abc220-967b-44cb-9e95-183b63571e88",
		UserId:         req.UserId,
		RaterId:        req.RaterId,
		Rating:         req.Rating,
		Comment:        req.Comment,
		CreatedAtHuman: "2024-11-24 10:00:00",
		UpdatedAtHuman: "2024-11-24 10:00:00",
	}, nil
}

// Uses Integration testing - Bufcoon based testing (gRPC mocking the client)
func TestRateUserBufconn(t *testing.T) {
	// Set up gRPC connection with Bufconn
	grpcConn, err := grpc.DialContext(context.Background(), "", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	assert.NoError(t, err)
	defer grpcConn.Close()

	// Instantiate the gRPC client
	client := inventory.NewInventoryServiceClient(grpcConn)

	// Simulate a frontend request's parameters
	userId := "6a7b83f0-30cb-4854-a32e-3576bf491858"
	raterId := "7a937e9d-1dc2-4e6d-ba38-d1648b05730c"
	comment := "greate product"
	rating := int32(5)

	// Make the gRPC call - handles the request timeout
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
	assert.Equal(t, "15abc220-967b-44cb-9e95-183b63571e88", resp.Id)
	assert.Equal(t, userId, resp.UserId)
	assert.Equal(t, raterId, resp.RaterId)
	assert.Equal(t, comment, resp.Comment)
	assert.Equal(t, "2024-11-24 10:00:00", resp.CreatedAtHuman)
}

// uses unit testing (mock based testing) - using goMock
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
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "6a7b83f0-30cb-4854-a32e-3576bf491858", resp.Id)
	assert.Equal(t, userId, resp.UserId)
	assert.Equal(t, raterId, resp.RaterId)
	assert.Equal(t, comment, resp.Comment)
	assert.Equal(t, "2024-11-24 10:00:00", resp.CreatedAtHuman)
}
