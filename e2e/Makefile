ROOT_DIR := ..
DEV_COMPOSE := docker/docker-compose.e2e.yaml
PLAYWRIGHT_COMPOSE := docker/docker-compose.e2e.playwright.yaml
TMP_DIR := tmp
RESULTS_DIR := test-results

DOCKER_COMPOSE ?= docker compose

.PHONY: prepare-e2e dev dev-down test test-down clean-e2e

prepare-e2e:
	@mkdir -p $(TMP_DIR) $(RESULTS_DIR)

# Launch proxy + stub auth for manual verification
# Usage: make dev
# Press Ctrl+C to stop, or run make dev-down in another terminal

dev: prepare-e2e
	$(DOCKER_COMPOSE) -f $(DEV_COMPOSE) up --build

# Stop and remove dev containers

dev-down:
	$(DOCKER_COMPOSE) -f $(DEV_COMPOSE) down

# Run Playwright-based end-to-end tests
# Usage: make test

test: prepare-e2e
	$(DOCKER_COMPOSE) -f $(DEV_COMPOSE) -f $(PLAYWRIGHT_COMPOSE) up --build --abort-on-container-exit --exit-code-from playwright-runner

# Tear down test stack and remove volumes created by make test

test-down:
	$(DOCKER_COMPOSE) -f $(DEV_COMPOSE) -f $(PLAYWRIGHT_COMPOSE) down -v

# Remove generated OTP/test-result artifacts

clean-e2e:
	[ ! -d $(TMP_DIR) ] || find $(TMP_DIR) -type f -delete
	[ ! -d $(RESULTS_DIR) ] || find $(RESULTS_DIR) -mindepth 1 ! -name '.gitkeep' -delete
