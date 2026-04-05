build:
	CGO_ENABLED=0 go build -o portfolio ./cmd/portfolio/

run: build
	./portfolio

test:
	go test ./...

clean:
	rm -f portfolio

.PHONY: build run test clean
