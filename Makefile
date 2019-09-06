.PHONY: help run-tests test-shell up down start init clean-logs ps githooks dev-container docker-clean
.DEFAULT_GOAL := help
DOCKER_BUILDKIT=1
export DOCKER_BUILDKIT

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

up:  ## bring up docker compose in background
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
