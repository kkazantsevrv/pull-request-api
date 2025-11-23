# Pull Request API

Сервис для автоматического назначения ревьюверов и управления Pull Requests.

## Требования

- Docker & Docker Compose
- Go 1.22+ (для локальной разработки)

## Быстрый старт (Docker)

Проект настроен на запуск одной командой. База данных поднимется автоматически, миграции накатятся при старте приложения.

1. **Клонируйте репозиторий:**
   ```bash
   git clone https://github.com/kkazantsevrv/pull-request-api
   cd pull-request-api

2. **Запустите проект**
    ```bash
    make docker-up
    # Или напрямую через docker-compose:
    docker-compose up --build
3. **Провека**
    Сервер доступен по адресу: http://localhost:8080 (убедитесь, что порт 5432 свободен, либо измените проброс портов в docker-compose.yml)

## Команды Makefile

1. make docker-up — Сборка и запуск всего окружения (App + DB).
1. make docker-down — Остановка и удаление контейнеров.
1. make test — Запуск всех тестов (интеграционные).
1. make run — Локальный запуск (требует локально запущенной PostgreSQL).

## API ENDPOINTS

1. **Создать команду с участниками (создаёт/обновляет пользователей)**
    ```bash
    curl -X POST http://localhost:8080/team/add \
    -H "Content-Type: application/json" \
    -d '{
        "team_name": "Backend",
        "members": [
            {"user_id": "1", "username": "Khabib", "is_active": true},
            {"user_id": "2", "username": "Conor", "is_active": true},
            {"user_id": "3", "username": "Islam", "is_active": true}
        ]
    }'

2. **Cоздать PR и автоматически назначить до 2 ревьюверов из команды автора (Авторы не назначаются сами себе)**
    ```bash
    curl -X POST http://localhost:8080/pullRequest/create \
    -H "Content-Type: application/json" \
    -d '{
        "pull_request_id": "PR-101",
        "author_id": "1",
        "pull_request_name": "Feature Login"
    }'

3. **Получить PR'ы, где пользователь назначен ревьювером**
    ```bash
    curl -X GET "http://localhost:8080/users/getReview?user_id=2"

4. **Посмотреть статистику по PR**
    ```bash
    curl -X GET http://localhost:8080/users/getAssignmentStats

5. **Назначить активным**
    ```bash
    curl -X POST http://localhost:8080/users/setIsActive \
    -H "Content-Type: application/json" \
    -d '{
        "is_active": true,
        "user_id": "2"
    }'

6. **Пометить PR как Merged**
    ```bash
    curl -X POST http://localhost:8080/pullRequest/merge \
    -H "Content-Type: application/json" \
    -d '{
        "pull_request_id": "PR-101"
    }'

7. **Переназначить конкретного ревьювера на другого из его команды**
    ```bash
    curl -X POST http://localhost:8080/pullRequest/reassign \
    -H "Content-Type: application/json" \
    -d '{
        "pull_request_id": "PR-101",
        "old_user_id": "2"
    }'

8. **Получить команду с участниками**
    ```bash
    curl -X GET "http://localhost:8080/team/get?team_name=Backend"

# Результаты нагрузочного тестирования

Дата: 22 ноября 2025  
Инструмент: [k6](https://k6.io/)  
Эндпоинт: `POST /pullRequest/create`  
Количество виртуальных пользователей (VUs): 10  
Количество итераций: 100  

---

## Основные метрики

| Метрика                        | Значение                         |
|--------------------------------|---------------------------------|
| Прошедшие проверки (status 200) | 100% (100/100)                  |
| Данные полученные               | 31 kB                            |
| Данные отправленные             | 25 kB                            |
| Среднее время запроса (http_req_duration) | 29.15 ms                   |
| Минимальное время запроса       | 3.72 ms                          |
| Максимальное время запроса      | 46.8 ms                          |
| 90-й процентиль времени запроса | 34.91 ms                         |
| 95-й процентиль времени запроса | 35.7 ms                          |
| Процент неудачных запросов      | 0%                               |
| Итераций в секунду              | 322.5                             |

---

## Интерпретация

- Все запросы успешно выполнены, ошибок не зафиксировано.  
- Среднее время обработки запроса — ~29 ms.  
- Максимальное время запроса — 46.8 ms.  
- Система выдерживает одновременную нагрузку 10 виртуальных пользователей без ошибок.
