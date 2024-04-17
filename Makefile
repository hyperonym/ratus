NAME := ratus
VERSION := 0.8.0

DOCKER_HUB_OWNER ?= hyperonym
DOCKER_HUB_IMAGE := $(DOCKER_HUB_OWNER)/$(NAME):$(VERSION)

GITHUB_PACKAGES_OWNER ?= hyperonym
GITHUB_PACKAGES_IMAGE := ghcr.io/$(GITHUB_PACKAGES_OWNER)/$(NAME):$(VERSION)

TARGET_BINARY_PLATFORMS := aix/ppc64,android/arm64,darwin/amd64,darwin/arm64,freebsd/386,freebsd/amd64,freebsd/arm64,linux/386,linux/amd64,linux/arm64,linux/mips64le,linux/ppc64le,linux/riscv64,linux/s390x,windows/386,windows/amd64,windows/arm64
TARGET_CONTAINER_PLATFORMS := linux/386,linux/amd64,linux/arm64,linux/mips64le,linux/ppc64le,linux/s390x

.PHONY: build
build:
	@CGO_ENABLED=0 go build -a -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o bin/ ./cmd/*

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

.PHONY: docker
docker:
	@docker build --build-arg "VERSION=$(VERSION)" --tag $(DOCKER_HUB_IMAGE) .

.PHONY: docker-export
docker-export:
	@docker buildx build --build-arg "VERSION=$(VERSION)" --platform $(TARGET_BINARY_PLATFORMS) --target binary --output bin/ .

.PHONY: docker-hub
docker-hub:
	@docker buildx build --build-arg "VERSION=$(VERSION)" --platform $(TARGET_CONTAINER_PLATFORMS) --push --tag $(DOCKER_HUB_IMAGE) .

.PHONY: github-packages
github-packages:
	@docker buildx build --build-arg "VERSION=$(VERSION)" --platform $(TARGET_CONTAINER_PLATFORMS) --push --tag $(GITHUB_PACKAGES_IMAGE) .

.PHONY: github-release
github-release: changelog
	@gh release create v$(VERSION) -F release/changelog.md -t v$(VERSION)

.PHONY: install
install: build
	@install -d /usr/local/bin
	@install -m755 bin/* /usr/local/bin/

.PHONY: release
release: changelog docker-export
	@$(foreach platform,$(shell find bin -maxdepth 1 -mindepth 1 -type d | cut -c 5-),zip -9 -j release/$(NAME)-$(VERSION)-$(subst _,-,$(platform)).zip bin/$(platform)/*;)

.PHONY: run
run:
	@go run ./cmd/*

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
