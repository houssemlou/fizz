.PHONY: run run-worker build build-worker test lint tidy up down mock gen \
        curl-fizzbuzz curl-fizzbuzz-custom curl-stats curl-health load-test \
        helm-local helm-uninstall

HOST ?= http://localhost:8081

run:
	go run ./cmd/server

run-worker:
	go run ./cmd/stats-worker

build:
	CGO_ENABLED=1 go build -o bin/server ./cmd/server

build-worker:
	CGO_ENABLED=1 go build -o bin/stats-worker ./cmd/stats-worker

test:
	go test ./...

lint:
	go vet ./...

tidy:
	go mod tidy

# Regenerate OpenAPI stubs (requires: go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest).
gen:
	oapi-codegen -config oapi-codegen.yaml api/v1/openapi.yaml

# Regenerate mocks (requires: go install github.com/vektra/mockery/v2@latest).
mock:
	mockery

# curl examples — override host with: make curl-fizzbuzz HOST=http://localhost:9090
curl-fizzbuzz:
	curl -s "$(HOST)/v1/fizzbuzz?int1=3&int2=5&limit=15&str1=fizz&str2=buzz" | python3 -m json.tool

curl-fizzbuzz-custom:
	curl -s "$(HOST)/v1/fizzbuzz?int1=2&int2=7&limit=20&str1=foo&str2=bar" | python3 -m json.tool

curl-stats:
	curl -s "$(HOST)/v1/stats" | python3 -m json.tool

curl-health:
	curl -s "$(HOST)/v1/health" | python3 -m json.tool

# Deploy to local minikube using docker-compose as the dependency layer.
# Prerequisites: minikube start && eval $(minikube docker-env)
helm-local:
	docker build --target server -t fizzbuzz-server:local .
	docker build --target worker -t fizzbuzz-worker:local .
	minikube image load fizzbuzz-server:local
	minikube image load fizzbuzz-worker:local
	helm upgrade --install fizzbuzz ./helm/fizzbuzz \
		-f helm/fizzbuzz/values.local.yaml

helm-uninstall:
	helm uninstall fizzbuzz

# Load test — requires k6 (https://k6.io/docs/get-started/installation/).
# Override host or key: make load-test BASE_URL=http://staging API_KEY=secret
load-test:
	k6 run -e BASE_URL=$(HOST) -e API_KEY=$(API_KEY) k6/smoke.js

# Docker Compose helpers.
up:
	docker compose up

down:
	docker compose down -v
