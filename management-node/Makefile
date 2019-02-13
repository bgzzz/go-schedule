# common varaibles
PREFIX = 
NAME = management-node
VERSION = latest

# building binary locally
build:
	env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/$(NAME)
# cleaning locally saved binary 
clean:
	rm ./bin/$(NAME)
	 
# run unit tests locally
test-unit:
	go test -tags=unit -v

