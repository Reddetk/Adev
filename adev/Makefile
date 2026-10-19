# Makefile for adev
.DEFAULT_GOAL := help

APP_NAME     := adev
DOCKER_IMAGE := adev:latest
MIGRATE_PATH := ./migrations
GOBIN        := $(shell go env GOPATH)/bin
EXE          := $(if $(filter windows,$(shell go env GOOS)),.exe,)
MIGRATE      := $(GOBIN)/migrate$(EXE)
BIN          := bin/$(APP_NAME)$(EXE)

# Version comes from git: a tag if one exists, otherwise the short commit.
# Tag a release with: make tag VERSION=v0.1.0
GIT_VERSION := $(shell git describe --tags --always --dirty)
VERSION     ?= $(if $(strip $(GIT_VERSION)),$(strip $(GIT_VERSION)),dev)
BUILD_TIME  ?=
LDFLAGS     := -X main.Version=$(VERSION)
ifneq ($(strip $(BUILD_TIME)),)
LDFLAGS     += -X main.BuildTime=$(BUILD_TIME)
endif

# .env feeds the app, docker and the migration command, so they cannot disagree.
-include .env
export

DB_HOST      ?= localhost
DB_PORT      ?= 5432
DB_USER      ?= postgres
DB_PASSWORD  ?= password
DB_NAME      ?= adev
DATABASE_URL ?= postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable

# Windows has no rm -rf and no VAR=value command prefix.
ifeq ($(OS),Windows_NT)
RM      := rmdir /S /Q
PRODBIN := powershell -NoProfile -Command "$$env:CGO_ENABLED=0; $$env:GOOS='linux'; go build -trimpath -ldflags '-s -w $(LDFLAGS)' -o bin/$(APP_NAME) ./cmd/server"
else
RM      := rm -rf
PRODBIN := CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w $(LDFLAGS)' -o bin/$(APP_NAME) ./cmd/server
endif
.PHONY: help deps tools build run test test-race test-coverage test-db ci fmt vet lint clean \
        migrate-install migrate-up migrate-down migrate-force db-reset \
        dev-db dev-db-stop \
        docker-build docker-run compose-up compose-down \
        build-prod sec-scan api-docs mocks tag act-image ci-act ci-act-dry

help:
	@echo adev targets:
	@echo   make deps             download and tidy modules
	@echo   make tools            install migrate, golangci-lint, gosec, swag
	@echo   make build            build bin/$(APP_NAME), version $(VERSION)
	@echo   make run              run the server with the .env settings
	@echo   make ci               vet + unit tests, the unit CI job in one command
	@echo   make test             run all tests
	@echo   make test-race        run all tests with the race detector, needs cgo
	@echo   make test-db          postgres store tests, needs ADEV_TEST_DATABASE_URL
	@echo   make test-coverage    write coverage.out and coverage.html
	@echo   make fmt              gofmt the tree
	@echo   make vet              go vet the tree
	@echo   make lint             golangci-lint, run make tools first
	@echo   make mocks            regenerate testify mocks with goca
	@echo   make act-image      build the local runner image for act, about 700 MB
	@echo   make ci-act ACT_JOB=unit       run a GitHub Actions job locally with act
	@echo   make ci-act-dry ACT_JOB=unit   show the act plan, starts no container
	@echo   make tag VERSION=v0.1.0   tag the current commit
	@echo   make clean            remove build artifacts and caches
	@echo   make dev-db           start the database and wait until it is healthy
	@echo   make dev-db-stop      stop the database
	@echo   make migrate-up       apply migrations to DATABASE_URL
	@echo   make migrate-down     roll one migration back
	@echo   make migrate-force    force a schema version, make migrate-force VERSION=3
	@echo   make db-reset         drop every table
	@echo   make docker-build     build the docker image
	@echo   make compose-up       start app and database with docker compose
	@echo   make build-prod       build a static linux binary
	@echo   make sec-scan         gosec, run make tools first
	@echo   make api-docs         regenerate docs/ with swag, run make tools first

deps:
	go mod download
	go mod tidy

tools:
	go install -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	go install github.com/swaggo/swag/cmd/swag@latest

build:
	go build -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/server

ci: vet test

mocks:
	goca mocks

tag:
	@$(if $(filter v%,$(VERSION)),,$(error pass a version tag, for example: make tag VERSION=v0.1.0))
	git tag -a $(VERSION) -m "release $(VERSION)"
	@echo tagged $(VERSION)


run:
	go run ./cmd/server

test:
	go test ./...

test-race:
	go test -race ./...

# The postgres store tests truncate the forms table, so they need a dedicated database.
test-db:
	@$(if $(strip $(ADEV_TEST_DATABASE_URL)),,$(error set ADEV_TEST_DATABASE_URL to a dedicated test database))
	go test ./internal/repository/... -run Postgres -v

test-coverage:
	go test -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html

fmt:
	gofmt -l -w .

vet:
	go vet ./...

lint:
	golangci-lint run

clean:
	-go clean -cache -testcache
	-$(RM) bin
	-$(RM) coverage.out
	-$(RM) coverage.html

migrate-install:
	go install -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@latest

migrate-up:
	@$(MIGRATE) -path $(MIGRATE_PATH) -database "$(DATABASE_URL)" up

migrate-down:
	@$(MIGRATE) -path $(MIGRATE_PATH) -database "$(DATABASE_URL)" down 1

migrate-force:
	@$(MIGRATE) -path $(MIGRATE_PATH) -database "$(DATABASE_URL)" force $(VERSION)

db-reset:
	@$(MIGRATE) -path $(MIGRATE_PATH) -database "$(DATABASE_URL)" drop -f

# docker compose waits for the healthcheck of the database service, so the next
# migrate-up cannot race the server start. Works on Windows and Unix.
dev-db:
	docker compose up -d --wait database

dev-db-stop:
	docker compose stop database

# Run a GitHub Actions job locally with act.
# The unit job needs no database. The postgres job starts a second container.
ACT_JOB       ?= unit
ACT_IMAGE     ?= adev-act:latest
ACT_JOB_FLAGS ?=

act-image:
	docker build -t $(ACT_IMAGE) ci/act

# act reads .github/workflows and .actrc from the repository root, so these
# targets change directory first. Run them from adev/ as usual.
ci-act:
	cd .. && act -j $(ACT_JOB) $(ACT_JOB_FLAGS)

ci-act-dry:
	cd .. && act -j $(ACT_JOB) -n

docker-build:
	docker build -t $(DOCKER_IMAGE) .

docker-run:
	docker run --rm -p 8080:8080 --env-file .env $(DOCKER_IMAGE)

compose-up:
	docker compose up -d

compose-down:
	docker compose down

build-prod:
	$(PRODBIN)

sec-scan:
	gosec ./...

api-docs:
	swag init -g cmd/server/main.go -o docs
