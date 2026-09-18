.PHONY: up down test-unit test-integration test

up:
	docker compose up --build

down:
	docker compose down --remove-orphans

test-unit:
	docker compose -f docker-compose.test.yml run --build --rm api-test-unit

test-integration:
	@status=0; \
	docker compose -f docker-compose.test.yml up --build --abort-on-container-exit --exit-code-from api-test-integration api-test-integration || status=$$?; \
	docker compose -f docker-compose.test.yml down --volumes --remove-orphans; \
	exit $$status

test: test-unit test-integration