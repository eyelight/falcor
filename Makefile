clean:
	rm -rf build

build:
	mkdir build

example: clean build
	go build -o build/falcor-example examples/main.go
	
