.PHONY: dev build down logs clean

# Development with hot reload
dev:
	docker compose -f docker-compose.yml -f docker-compose.dev.yml up --build

# Production build
build:
	docker compose up --build

# Stop everything
down:
	docker compose down

# Follow logs
logs:
	docker compose logs -f

# Stop and delete all data (volumes)
clean:
	docker compose down -v
