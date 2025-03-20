run:
	@go run ./cmd/main.go

tests:
	go test ./... -coverprofile=coverage.out  -coverpkg=./... -v -race

coverage:
	@go tool cover -html=coverage.out
