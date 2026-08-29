.PHONY: run test lint swagger build sam-build sam-deploy

run:
	go run ./cmd/server

test:
	go test ./...

lint:
	go vet ./...

swagger:
	go run github.com/swaggo/swag/cmd/swag@latest init -g cmd/server/main.go -o docs

build:
	docker build -t instance-controller .

sam-build:
	sam build

sam-deploy:
	sam deploy --guided

