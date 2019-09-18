.PHONY: help test up down start init clean-logs ps githooks dev-container docker-clean rbox1 info go profile html show cover escape get pause

.DEFAULT_GOAL := help
DOCKER_BUILDKIT=1
export DOCKER_BUILDKIT

# https://timmurphy.org/2015/09/27/how-to-get-a-makefile-directory-path/
BASE=$(dir $(realpath $(firstword $(MAKEFILE_LIST))))

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

go: ## go image for testing inside docker
	@docker-compose exec client sh

info:
	@echo BASE: $(BASE)

rbox1: ## exec into rbox1 for testing
	@docker-compose exec -w /rqlite-v4.5.0-linux-amd64 rbox1 bash

up:  down ## bring up docker compose in background
	docker-compose up -d

down: ## Run docker-compose down, stopping all the containers
	@docker-compose down || echo "an issue occurred shutting composer down"

githooks: ## Install githooks, requires Git version >= 2.9
	git config core.hooksPath .githooks

init: githooks ## Install requirements, githooks, and any other developement utilties

dev-container: ## Build docker container for development and pull latest for external images
	#docker-compose pull controller-postgres controller-rabbitmq controller-fakesmtp nginx
	#docker build --target development -t controller-dev:latest -f docker/Dockerfile .

start: dev-containers up ## build and start dev containers

kill: ## kills all docker instances
	docker kill $$(docker ps -q) 2> /dev/null || true

prune: ## removes system volumes
	docker system prune --volumes -f

docker-clean: kill prune  ## Remove all unused containers, networks, images (both dangling and unreferenced), and volumes

#up: start-containers ## Build dev container and start the stack
start-containers: down dev-container up ## Build dev container and start the stack

ps: ## show docker ps
	@docker-compose ps

profile:
	@go test -coverprofile cover.out

html:
	@go tool cover -html cover.out

show:	profile html

cover:
	@go test -cover $(arg1)  $(goflags) ./...

escape:
	@go build -gcflags '-m' db.go lite.go table.go

test:	## test project (use $GO_TEST to modify)
	http_proxy=http://localhost:8888/ go test ./... $(GO_TEST)

get:
	@go get -v -u -t ./...

pause:
	sleep 30
