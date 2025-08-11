SHELL := /bin/sh

.PHONY: run test docker

run:
	go run ./cmd/router --config=config/models.yaml --addr=:8080

test:
	go test ./...

docker:
	docker build -t llm-router-go .