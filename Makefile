run:
	@echo 'Running the project ...'
	go build -o bin/main ./cmd/hezzl-goods/main.go
	./bin/main

up: 
	docker compose -f deployment/local/docker-compose.yml up -d

down:
	docker compose -f deployment/local/docker-compose.yml down --remove-orphans

tidy:
	go mod tidy

lint: tidy
	# gofumpt -w .
	gci write . --skip-generated -s standard -s default 	
	golangci-lint run ./...

test: up
	go test -race ./... -v -coverpkg=./... -coverprofile=coverage.txt -covermode atomic
	go tool cover -func=coverage.txt | grep 'total'
	which gocover-cobertura || go install github.com/t-yuki/gocover-cobertura@latest
	gocover-cobertura < coverage.txt > coverage.xml

image:
	docker build -f deployment/local/Dockerfile .