start:
	CompileDaemon -exclude-dir=".git" -exclude-dir="tmp"

build: clean
	go build cli/webext

clean:
	rm -f webext

lint:
	 golangci-lint run ./...

format:
	gofumpt -w .

test:
	go test ./...

zip:
	cd tmp/extension && zip -r ../extension.zip ./