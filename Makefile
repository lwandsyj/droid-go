all: build

build:
	mkdir -p build
	go build -o build/droid

run:
	./build/droid

format:
	go fmt

test:
	go test -v

clean:
	rm -rf build

.PHONY: all build run format test clean