VERSION ?= dev

.PHONY: build run clean test fmt

build:
	go build -ldflags="-X 'dolphinzZ/cmd.Version=$(VERSION)'" -o dolphinzZ .

run: build
	./dolphinzZ

clean:
	rm -f dolphinzZ
	rm -f /tmp/dolphinzZ/*.jsonl

test:
	go test ./...

fmt:
	go fmt ./...
