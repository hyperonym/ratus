NAME := ratus
CMD := ratus
VERSION := 0.6.0

DOCKER_HUB_OWNER ?= hyperonym
DOCKER_HUB_IMAGE := $(DOCKER_HUB_OWNER)/$(NAME):$(VERSION)

GITHUB_PACKAGES_OWNER ?= hyperonym
GITHUB_PACKAGES_IMAGE := ghcr.io/$(GITHUB_PACKAGES_OWNER)/$(NAME):$(VERSION)

TARGET_BINARY_PLATFORMS := aix/ppc64,android/arm64,darwin/amd64,darwin/arm64,freebsd/386,freebsd/amd64,freebsd/arm64,linux/386,linux/amd64,linux/arm64,linux/mips64le,linux/ppc64le,linux/riscv64,linux/s390x,windows/386,windows/amd64,windows/arm64
TARGET_CONTAINER_PLATFORMS := linux/386,linux/amd64,linux/arm64,linux/mips64le,linux/ppc64le,linux/s390x

LDFLAGS := -ldflags "-s -w -X main.version=v$(VERSION)"

BUILD_INPUT := ./cmd/$(CMD)
ifeq ($(OS),Windows_NT)
	BUILD_OUTPUT := bin/$(CMD).exe
else
	BUILD_OUTPUT := bin/$(CMD)
endif

comma := ,

.PHONY: build
build:
	@CGO_ENABLED=0 go build -buildvcs=false -trimpath $(LDFLAGS) -o $(BUILD_OUTPUT) $(BUILD_INPUT)

.PHONY: build-%
build-%:
	@CGO_ENABLED=0 GOOS=$(firstword $(subst -, ,$*)) GOARCH=$(lastword $(subst -, ,$*)) go build -a -trimpath $(LDFLAGS) -o bin/$(subst -,/,$*)/$(CMD)$(if $(findstring windows,$*),.exe,) $(BUILD_INPUT)

.PHONY: changelog
changelog:
	@mkdir -p release
	@git log $(shell git describe --tags --abbrev=0 2> /dev/null)..HEAD --pretty='tformat:* [%h] %s' > release/changelog.md
	@cat release/changelog.md

.PHONY: clean
clean:
	@go clean
	@rm -f coverage.out
	@rm -rf bin/ release/

.PHONY: docker-%
docker-%: build-%
	@docker build --platform=$(subst -,/,$*) --tag $(DOCKER_HUB_IMAGE)-$* .

.PHONY: docker-hub
docker-hub: $(foreach t,$(subst $(comma), ,$(TARGET_CONTAINER_PLATFORMS)),build-$(subst /,-,$(t)))
	@docker buildx build --push --platform=$(TARGET_CONTAINER_PLATFORMS) --tag $(DOCKER_HUB_IMAGE) .

.PHONY: github-packages
github-packages: $(foreach t,$(subst $(comma), ,$(TARGET_CONTAINER_PLATFORMS)),build-$(subst /,-,$(t)))
	@docker buildx build --push --platform=$(TARGET_CONTAINER_PLATFORMS) --tag $(GITHUB_PACKAGES_IMAGE) .

.PHONY: github-release
github-release: changelog
	@gh release create v$(VERSION) -F release/changelog.md -t v$(VERSION)

.PHONY: install
install: build
	@install -d /usr/local/bin
	@install -m755 $(BUILD_OUTPUT) /usr/local/bin/

.PHONY: release
release: $(foreach t,$(subst $(comma), ,$(TARGET_BINARY_PLATFORMS)),release-$(subst /,-,$(t)))

.PHONY: release-%
release-%: changelog build-%
	@zip -9 -j release/$(CMD)-$(VERSION)-$*.zip bin/$(subst -,/,$*)/*

.PHONY: run
run:
	@go run $(BUILD_INPUT)

.PHONY: test
test:
	@go test -timeout 5m -v ./...

.PHONY: test-coverage
test-coverage:
	@go test -race -covermode=atomic -coverprofile=coverage.out ./...

.PHONY: test-engine-%
test-engine-%:
	@go test -timeout 5m -v ./internal/engine/$*

.PHONY: test-short
test-short:
	@go test -short -v ./...

.PHONY: spec
spec:
	@swag init --dir internal/controller --generalInfo controller.go -o docs --parseDependency --parseInternal --outputTypes json,yaml
	@curl -X POST "https://converter.swagger.io/api/convert" -H "accept: application/yaml" -H "Content-Type: application/json" -d "@docs/swagger.json" -o docs/openapi.yaml
	@curl -X POST "https://converter.swagger.io/api/convert" -H "accept: application/json" -H "Content-Type: application/json" -d "@docs/swagger.json" | python3 -m json.tool > docs/openapi.json

.PHONY: spec-serve
spec-serve:
	@python3 -m http.server --directory docs/ 8080
