.PHONY: build

build:
	go build -buildmode=plugin -o adapter-slack.so adapter.go
