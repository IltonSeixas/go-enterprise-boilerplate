.PHONY: proto test build run

proto:
	protoc \
		--go_out=. --go_opt=module=github.com/IltonSeixas/go-enterprise-boilerplate \
		--go-grpc_out=. --go-grpc_opt=module=github.com/IltonSeixas/go-enterprise-boilerplate \
		-I proto proto/boilerplate/v1/boilerplate.proto

test:
	go test ./...

build:
	go build -o bin/server ./cmd/server

run: build
	./bin/server
