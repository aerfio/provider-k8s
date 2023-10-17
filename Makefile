#CURRENT_DIR = $(dir $(abspath $(firstword $(MAKEFILE_LIST))))
#GOBIN=$(CURRENT_DIR)/bin

all: generate test build

CONTROLLER_TOOLS_VERSION ?= v0.13.0
CONTROLLER_GEN ?= bin/controller-gen-${CONTROLLER_TOOLS_VERSION}
${CONTROLLER_GEN}:
	./hack/get-go-tool.sh "sigs.k8s.io/controller-tools/cmd/controller-gen" $(CONTROLLER_TOOLS_VERSION)

ANGRYJET_VERSION ?= v0.0.0-20230714144037-2684f4bc7638
ANGRYJET ?= bin/angryjet-${ANGRYJET_VERSION}
${ANGRYJET}:
	./hack/get-go-tool.sh "github.com/crossplane/crossplane-tools/cmd/angryjet" $(ANGRYJET_VERSION)

# yolo, I know what the docs say, but goreleaser is doing essentially nothing unusual and I always use newest Go version :shrug:
GOLANGCI_LINT_VERSION ?= v1.54.2
GOLANGCI_LINT ?= bin/golangci-lint-${GOLANGCI_LINT_VERSION}
${GOLANGCI_LINT}:
	./hack/get-go-tool.sh "github.com/golangci/golangci-lint/cmd/golangci-lint" $(GOLANGCI_LINT_VERSION)

.PHONY: generate
generate: ${ANGRYJET} ${CONTROLLER_GEN}
	rm -rf ./package/crds
	$(CONTROLLER_GEN) object:headerFile=./hack/boilerplate.go.txt paths=./... crd:crdVersions=v1 output:artifacts:config=./package/crds
	${ANGRYJET} generate-methodsets --header-file=./hack/boilerplate.go.txt ./...

.PHONY: test
test:
	go test ./... -race -v

.PHONY: build
build:
	go build -o ./bin/provider-lambda ./cmd/provider

lint: ${GOLANGCI_LINT}
	$(GOLANGCI_LINT) run ./...
