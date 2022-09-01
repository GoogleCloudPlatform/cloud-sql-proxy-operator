# Kubebuilder creates a makefile to generate stuff. This is the stub of
# a makefile to be replaced with the kubebuilder boilerplate.

.PHONY: build
build:
	mkdir -p bin
	go build -o bin/cloud-sql-proxy-operator ./...

.PHONY: test
test:
	go test ./...