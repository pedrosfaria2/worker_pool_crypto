.PHONY: all build run test clean docker docker-build docker-run

# Go parameters
BINARY_NAME=worker_pool_crypto
MAIN_PATH=cmd/worker/main.go

# Docker parameters
DOCKER_IMAGE=worker-pool-crypto
DOCKER_TAG=latest

all: clean build

build:
	@echo "Building..."
	go build -o bin/$(BINARY_NAME) $(MAIN_PATH)

run: build
	@echo "Running..."
	./bin/$(BINARY_NAME)

test:
	@echo "Testing..."
	go test -v ./...

clean:
	@echo "Cleaning..."
	rm -rf bin/
	go clean

docker-build:
	@echo "Building docker image..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

docker-run:
	@echo "Running docker container..."
	docker run -p 8080:8080 $(DOCKER_IMAGE):$(DOCKER_TAG)

up:
	@echo "Starting docker-compose..."
	docker-compose up --build

down:
	@echo "Stopping docker-compose..."
	docker-compose down

logs:
	@echo "Showing logs..."
	docker-compose logs -f