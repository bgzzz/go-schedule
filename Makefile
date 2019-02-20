# common variables
PREFIX =
NAME = go-schedule
VERSION = latest

.PHONY: proto dep

# build all locally
build: \
	dep \
	proto \
	build-management-node \
	build-worker-node \
	build-schedctl

# generate proto 
proto:
	protoc -I proto/ --go_out=plugins=grpc:proto/ proto/scheduler.proto	

## build solution parts
# TBD: refactor it in variable target fashion
build-management-node:
	make -C management-node/

build-worker-node:
	make -C worker-node/

build-schedctl:
	make -C schedctl/

# locally run unit tests
run-unit-tests: \
	dep \
	proto \
	run-unit-tests-only

# locally run integration tests
run-int-tests: \
	test-int-management-node

# make it a different target for the purpose of 
# running in docker if needed 
run-unit-tests-only: \
	test-unit-management-node \
	test-unit-worker-node \
	test-unit-schedctl

## run unit tests for solution parts 
test-unit-management-node:
	make test-unit -C management-node/ 

test-unit-worker-node:
	make test-unit -C worker-node/

test-unit-schedctl: 
	make test-unit -C schedctl/

## run integrational tests for solution parts
test-int-management-node:
	make test-int -C management-node/

# cleaning locally saved binary 
clean: \
	clean-management-node \
	clean-worker-node \
	clean-schedctl		

clean-management-node: 
	make clean -C management-node/ 

clean-worker-node: 
	make clean -C worker-node/

clean-schedctl: 
	make clean -C schedctl/

# update dependencies 
dep: 
	dep ensure

# build docker image
docker-build-all: \
	dep 
	docker build -t $(PREFIX)management-node:$(VERSION) . --target=management-node
	docker build -t $(PREFIX)worker-node:$(VERSION) . --target=worker-node
	docker build -t $(PREFIX)schedctl:$(VERSION) . --target=schedctl
         
# build image with dependancies and go utils 
docker-build-tests:
	docker build -t $(PREFIX)$(NAME)-tests . --target builder

# run container with unit tests
docker-run-unit: \
	docker-build-tests
	docker run --rm $(PREFIX)$(NAME)-tests make run-unit-tests-only
