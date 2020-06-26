.PHONY: test
test:
	go test ./...

.PHONY: build
build:
	./scripts/build.sh

.PHONY: install
install:
	./scripts/install.sh

.PHONY: clean
clean:
	-rm ion-go
