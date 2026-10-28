TAGS=netgo osusergo sqlite_omit_load_extension
VERSION=$(shell git describe --tags --abbrev=0 | cut -c 2-)
COMMIT=$(shell git rev-parse --verify HEAD)
DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LD_FLAGS=-s -w -linkmode external -extldflags "-static" -X main.BuildDate=$(DATE) -X main.BuildMode=prod -X main.BuildCommit=$(COMMIT) -X main.BuildVersion=$(VERSION)
BUILD_DIR=./build
PWD=$(shell pwd)

license-dir:
	mkdir -p build/license || true

download-tools:
	go install golang.org/x/tools/cmd/goimports@v0.1.10
	go install github.com/99designs/gqlgen@v0.17.85

generate-go:
	gqlgen

generate-js:
	(cd ui && yarn generate)

generate: generate-go generate-js

lint-go:
	go vet ./...
	goimports -l $(shell find . -type f -name '*.go' -not -path "./vendor/*")

lint-js:
	(cd ui && yarn format:check)
	(cd ui && yarn lint:check)

lint: lint-go lint-js

format-go:
	goimports -w $(shell find . -type f -name '*.go' -not -path "./vendor/*")

format-js:
	(cd ui && yarn format)

format: format-go format-js

test-go:
	go test --race -coverprofile=coverage.txt -covermode=atomic ./...

test-js:
	(cd ui && CI=true yarn test)

test: test-go test-js

# test-database-launcher runs the launcher's own shell regression suite. It
# exercises loopback publishing, credential-safe output, required-mode failure
# (DSN absent / unusable loopback DSN) and cleanup using command doubles, so it
# needs no server. Mirrors the `postgres-launcher-test` step in build.yml.
.PHONY: test-database-launcher
test-database-launcher:
	./hack/test-databases_test.sh

# test-postgres runs the shared cross-database regression/acceptance fixtures
# against BOTH SQLite (always executed) and a disposable PostgreSQL server
# (mandatory, FR-006 / WP04). It mirrors the `postgres-test` job in
# .github/workflows/build.yml. See hack/test-databases.sh for the environment
# variables it reads (TRAGGO_TEST_POSTGRES_DSN / TRAGGO_TEST_PG_IMAGE /
# TRAGGO_TEST_PG_PORT / TRAGGO_TEST_PG_PACKAGES / TRAGGO_TEST_GO_JSON).
#
# Without TRAGGO_TEST_POSTGRES_DSN the script starts a disposable postgres:18
# container itself; with a DSN it reuses the running server. All six mandatory
# packages are exercised (statistics, device, dashboard, test, database,
# model), the child go test runs with -json (TRAGGO_TEST_GO_JSON=1), and the
# JSON stream is validated by hack/check-postgres-results.go against the
# reviewed live-suite manifest. pipefail preserves the launcher's own nonzero
# exit so a failed harness never reports green. TRAGGO_TEST_POSTGRES_REQUIRED is
# always forced to 1 inside the launcher so a green run provably executed real
# PostgreSQL fixtures.
.PHONY: test-postgres
test-postgres: SHELL := /bin/bash
test-postgres:
	set -o pipefail; TRAGGO_TEST_GO_JSON=1 TRAGGO_TEST_PG_PACKAGES="./statistics/... ./device/... ./dashboard/... ./test/... ./database/... ./model/..." ./hack/test-databases.sh | go run ./hack/check-postgres-results.go

install-go:
	go mod download

install-js:
	(cd ui && yarn)

build-js:
	(cd ui && yarn build)

pre-build: build-js

build-bin-local: pre-build
	CGO_ENABLED=1 go build -a -ldflags '${LD_FLAGS}' -tags '${TAGS}' -o ${BUILD_DIR}/traggo-server

.PHONY: release
release:
	docker build -t traggo:build -f docker/Dockerfile.build docker
	docker run \
		--rm \
		-v "$$HOME/.docker/config.json:/root/.docker/config.json" \
		-e CGO_ENABLED=1 \
		-e GITHUB_TOKEN="$$GITHUB_TOKEN" \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v $$PWD:/work \
		-w /work \
		traggo:build \
		release --skip=validate --clean

.PHONY: release-snapshot
release-snapshot:
	docker build -t traggo:build -f docker/Dockerfile.build docker
	docker run \
		--rm \
		-v "$$HOME/.docker/config.json:/root/.docker/config.json" \
		-e CGO_ENABLED=1 \
		-e GITHUB_TOKEN="$$GITHUB_TOKEN" \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v $$PWD:/work \
		-w /work \
		traggo:build \
		release --clean --snapshot

install: install-go install-js

.PHONY: build
