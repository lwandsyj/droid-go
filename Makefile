all: build

build:
	mkdir -p build
	go build -o build/droid main.go

run:
	./build/droid

format:
	go fmt

test:
	go test -v

clean:
	rm -rf build
