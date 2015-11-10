.PHONY: default dnsgo deps clean all run

export GOPATH:=$(shell pwd)

all: deps
	go install dnsgo

deps:
	go get -d -v dnsgo

clean:
	go clean -i -r dnsgo/...

run: deps all
	./bin/dnsgo

default: all