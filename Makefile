CURRENT_DIR = $(dir $(abspath $(firstword $(MAKEFILE_LIST))))

all: generate test build

CONTROLLER_TOOLS_VERSION ?= v0.12.0
CONTROLLER_GEN ?= bin/controller-gen-${CONTROLLER_TOOLS_VERSION}
${CONTROLLER_GEN}:
	./hack/get-go-tool.sh "sigs.k8s.io/controller-tools/cmd/controller-gen" $(CONTROLLER_TOOLS_VERSION)

ANGRYJET_VERSION ?= v0.0.0-20230925130601-628280f8bf79
ANGRYJET ?= bin/angryjet-${ANGRYJET_VERSION}
${ANGRYJET}:
	./hack/get-go-tool.sh "github.com/crossplane/crossplane-tools/cmd/angryjet" $(ANGRYJET_VERSION)

# yolo, I know what the docs say, but goreleaser is doing essentially nothing unusual in golangci-lint's release pipeline and I always use newest Go version so we should be fine :shrug:
GOLANGCI_LINT_VERSION ?= v1.55.1
GOLANGCI_LINT ?= bin/golangci-lint-${GOLANGCI_LINT_VERSION}
${GOLANGCI_LINT}:
	./hack/get-go-tool.sh "github.com/golangci/golangci-lint/cmd/golangci-lint" $(GOLANGCI_LINT_VERSION)

CRD_REF_DOCS_VERSION ?= v0.0.9
CRD_REF_DOCS ?= bin/crd-ref-docs-${CRD_REF_DOCS_VERSION}
${CRD_REF_DOCS}:
	./hack/get-go-tool.sh "github.com/elastic/crd-ref-docs" $(CRD_REF_DOCS_VERSION)

.PHONY: clean
clean:
	rm -rf ./package/crds
	rm -rf ./apis/object/v1alpha1/zz_generated.managed.go
	rm -rf ./apis/object/v1alpha1/zz_generated.managedlist.go

.PHONY: generate
generate: ${ANGRYJET} ${CRD_REF_DOCS} generate-crds
	$(ANGRYJET) generate-methodsets --header-file=./hack/boilerplate.go.txt ./...
	$(CRD_REF_DOCS) --source-path=${CURRENT_DIR}/apis --config=crd-ref-docs-config.yaml --renderer=markdown --output-path=./docs/crd-docs.md

.PHONY: generate-crds
generate-crds: ${CONTROLLER_GEN}
	$(CONTROLLER_GEN) object:headerFile=./hack/boilerplate.go.txt paths=./... crd:crdVersions=v1 output:artifacts:config=./package/crds

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
