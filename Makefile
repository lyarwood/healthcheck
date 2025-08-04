.PHONY: build clean lint lint-install

build:
	go build -o healthcheck .

clean:
	rm -f healthcheck

lint-install:
	@which golangci-lint > /dev/null || { \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	}

lint: lint-install
	golangci-lint run

lint-fix: lint-install
	golangci-lint run --fix
