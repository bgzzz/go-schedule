FROM golang:1.10.3 AS builder

# prepare buildin environment
WORKDIR /go/src/github.com/bgzzz/go-schedule/
RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
COPY . /go/src/github.com/bgzzz/go-schedule/
RUN dep ensure

# building all three parts of the solution
RUN cd management-node && CGO_ENABLED=0 GOOS=linux go build .
RUN cd worker-node && CGO_ENABLED=0 GOOS=linux go build .
RUN cd schedctl && CGO_ENABLED=0 GOOS=linux go build .

# setup management-node image
FROM scratch AS management-node
COPY --from=builder /go/src/github.com/bgzzz/go-schedule/management-node/management-node /.
ENTRYPOINT ["/management-node"]

# setup worker-node 
# choose ubuntu as base image 
# to run some tasks inside 
FROM ubuntu AS worker-node
COPY --from=builder /go/src/github.com/bgzzz/go-schedule/worker-node/worker-node /usr/bin/.
ENTRYPOINT ["/worker-node"]

# setup schedctl image
FROM ubuntu AS schedctl
COPY --from=builder /go/src/github.com/bgzzz/go-schedule/schedctl/schedctl /usr/bin/.
ENTRYPOINT ["/bin/bash"]
