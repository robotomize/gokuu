GO_CMD?=go
CI_LINT ?= golangci-lint

test:
	@$(GO_CMD) test -v -cover -covermode=atomic ./...

test-cover:
	@$(GO_CMD) test -race -timeout=10m ./... -coverprofile=coverage.out

lint:
	@$(GO_CMD) list -f '{{.Dir}}' ./...  \
		| xargs golangci-lint run; if [ $$? -eq 1 ]; then \
			echo ""; \
			echo "Lint found suspicious constructs. Please check the reported constructs"; \
			echo "and fix them if necessary before submitting the code for reviewal."; \
		fi

tool:
	@$(GO_CMD) build -o ./bin/gocygen ./tools/gocygen
	@$(GO_CMD) build -o ./bin/gocyupd ./tools/gocyupd