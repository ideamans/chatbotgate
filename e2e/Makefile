ROOT_DIR := ..
DEV_COMPOSE := docker/docker-compose.e2e.yaml
PLAYWRIGHT_COMPOSE := docker/docker-compose.e2e.playwright.yaml
TMP_DIR := tmp
RESULTS_DIR := test-results
DEFAULT_HEADLESS := $(if $(PLAYWRIGHT_HEADLESS),$(PLAYWRIGHT_HEADLESS),true)
HEADLESS ?= $(DEFAULT_HEADLESS)

DOCKER_COMPOSE ?= docker compose
NPM ?= npm

.PHONY: prepare-e2e dev dev-down test playwright test-down clean-e2e

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

# Run Playwright-based end-to-end tests with full lifecycle
# Usage: make test
# HEADLESS=false make test (to run with browser UI)

test: prepare-e2e
	@echo "Starting services..."
	@$(DOCKER_COMPOSE) -f $(DEV_COMPOSE) up -d --build
	@echo "Running Playwright tests (HEADLESS=$(HEADLESS))..."
	@PLAYWRIGHT_HEADLESS=$(HEADLESS) $(NPM) test; \
		EXIT_CODE=$$?; \
		echo "Shutting down services..."; \
		$(DOCKER_COMPOSE) -f $(DEV_COMPOSE) down; \
		exit $$EXIT_CODE

# Run only Playwright tests (services must be already running)
# Usage: make playwright
# HEADLESS=false make playwright (to run with browser UI)

playwright:
	@echo "Running Playwright tests (HEADLESS=$(HEADLESS))..."
	@PLAYWRIGHT_HEADLESS=$(HEADLESS) $(NPM) test

# Tear down test stack and remove volumes created by make test

test-down:
	$(DOCKER_COMPOSE) -f $(DEV_COMPOSE) -f $(PLAYWRIGHT_COMPOSE) down -v

# Remove generated OTP/test-result artifacts

clean-e2e:
	@# Remove all files in tmp directory
	[ ! -d $(TMP_DIR) ] || find $(TMP_DIR) -type f -delete
	@# Remove test results but keep .gitkeep
	[ ! -d $(RESULTS_DIR) ] || find $(RESULTS_DIR) -mindepth 1 ! -name '.gitkeep' -delete
