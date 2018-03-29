default: test

test:
	go test -v -cover ./...

run-example:
	go run example/example.go
