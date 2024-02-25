CURRENT_DIR = $(dir $(abspath $(firstword $(MAKEFILE_LIST))))

all: generate test build

# yolo, I know what the docs say, but goreleaser is doing essentially nothing unusual in golangci-lint's release pipeline and I always use newest Go version so we should be fine :shrug:
GOLANGCI_LINT_VERSION = v1.56.2
GOLANGCI_LINT ?= bin/golangci-lint-${GOLANGCI_LINT_VERSION}
${GOLANGCI_LINT}:
	./hack/get-go-tool.sh "github.com/golangci/golangci-lint/cmd/golangci-lint" $(GOLANGCI_LINT_VERSION)

.PHONY: clean
clean:
	rm -rf ./package/crds
	rm -rf ./apis/object/v1alpha1/zz_generated.managed.go
	rm -rf ./apis/object/v1alpha1/zz_generated.managedlist.go

.PHONY: generate
generate: ${CRD_REF_DOCS}
	go generate -tags generate ./apis/...

.PHONY: test
test:
	go test ./... -race

.PHONY: build
build:
	go build -o ./bin/provider-k8s ./cmd/provider

.PHONY: lint
lint: ${GOLANGCI_LINT}
	$(GOLANGCI_LINT) run ./...

.PHONY: lint-fix
lint-fix: ${GOLANGCI_LINT}
	$(GOLANGCI_LINT) run ./... --fix
