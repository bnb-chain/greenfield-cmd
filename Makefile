SHELL := /bin/bash

.PHONY: all build

build:
	go build -o ./build/gnfd-cmd cmd/*.go
	cp config.toml build
