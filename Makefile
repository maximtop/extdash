start:
	CompileDaemon -command="./extdash"

build:
	make clean && go build

clean:
	rm extdash