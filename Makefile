start:
	PORT=3000 CompileDaemon -command="./extdash"

build:
	make clean && go build

clean:
	rm extdash

lint:
	 golangci-lint ./...

format:
	go fmt ./...