# Routing Package 					= github.com/go-chi/chi/v5 (https://github.com/go-chi/chi)
# Routing Package Middleware 		= github.com/go-chi/chi/v5/middleware
# Routing Package Cors protection 	= github.com/go-chi/cors

# Set the binary name
BROKER_BINARY=brokerApp

# build_broker_service: builds the broker binary as a Linux executable
build_broker_service: ## Build the broker service binary
	@echo "Building broker service binary..."
	@cd cmd/api && env GOOS=linux CGO_ENABLED=0 go build -o ../../$(BROKER_BINARY)
	@echo "Done!"
	
test: 
	cd cmd/api && go test -v -cover ./...