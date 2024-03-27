
.PHONY: install, test

install:
	go install

test:
	go generate ./...
