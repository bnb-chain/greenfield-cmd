SHELL=/usr/bin/env bash
# PROJECTNAME=$(shell basename "$(PWD)")
PROJECTNAME=gnfd-cmd
LDFLAGS=-ldflags="-X 'main.Version=$(shell git describe --tags --dirty=-dev)'"

.PHONY: help build install clean

## help: Get more info on make commands.
help: Makefile
	@echo " Choose a command run in "$(PROJECTNAME)":"
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
.PHONY: help

## build: Build binary.
build:
	@echo "--> Building"
	@go build -o build/ ${LDFLAGS} ./cmd/${PROJECTNAME}

## install: Install the binary into the GOBIN directory.
install:
	@echo "--> Installing"
	@go install ${LDFLAGS} ./cmd/${PROJECTNAME}

## clean: Clean up celestia-node-exporter binary.
clean:
	@echo "--> Cleaning up ./build & ./downloads"
	@rm -rf build/*
	@rm -rf downloads/*

