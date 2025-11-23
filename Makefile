.PHONY: build run test clean docker-up docker-down migrate-new

BINARY_NAME=pr-api

build:
	go build -o ${BINARY_NAME} cmd/server/main.go

# Запуск локально (требует поднятой БД локально или через docker-compose up postgres)
run: build
	./${BINARY_NAME}

# Запуск тестов
test:
	go test -v ./...

# Очистка
clean:
	go clean
	rm -f ${BINARY_NAME}

# ==============================================================================
# Docker команды
# ==============================================================================

# Поднять всё в докере (с пересборкой)
docker-up:
	docker compose up --build

# Остановить и удалить контейнеры
docker-down:
	docker compose down

# Просмотр логов
docker-logs:
	docker compose logs -f