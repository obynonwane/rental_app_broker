# Routing Package 					= github.com/go-chi/chi/v5 (https://github.com/go-chi/chi)
# Routing Package Middleware 		= github.com/go-chi/chi/v5/middleware
# Routing Package Cors protection 	= github.com/go-chi/cors


test: 
	cd cmd/api && go test -v -cover ./...