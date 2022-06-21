start:
	CompileDaemon -exclude-dir=".git" -exclude-dir="tmp"

build: clean
	go build

clean:
	rm -f extdash

lint:
	 golangci-lint run ./...

format:
	go fmt ./...

test:
	go test ./...

zip:
	cd tmp/extension; zip -r ../extension.zip ./

