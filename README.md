# basketball-stats-service
NBA Basketball Stats Service — Go, Postgres, Docker. Home assignment for Skyhawk Security.

## Validation & Errors

Сервисный слой выполняет нормализацию и доменную валидацию до обращения к БД. Правила:
- Строки trim.
- Enum (position/status) приводятся к единому регистру.
- Пагинация: limit <=0 -> 50, limit>100 -> 100, offset<0 -> 0.
- Все нарушения агрегируются в один ErrInvalidInput c массивом field_errors.

HTTP маппинг ошибок:
| Error | HTTP | payload.error |
|-------|------|---------------|
| ErrInvalidInput | 400 | invalid_input |
| ErrNotFound | 404 | not_found |
| ErrAlreadyExists | 409 | already_exists |
| ErrConflict | 409 | conflict |
| (прочее) | 500 | internal_error |

Пример 400:
```
{
  "error": "invalid_input",
  "message": "one or more fields are invalid",
  "field_errors": [
    {"field": "name", "message": "must not be empty"},
    {"field": "position", "message": "must be one of PG, SG, SF, PF, C"}
  ]
}
```

## Make Targets (основные)
- `make run` — запустить сервис.
- `make test` — все тесты.
- `make test-contract` — контрактные тесты репозитория (при наличии Postgres).

## Структура слоёв
- handler -> service -> repository -> Postgres.
- handler ничего не знает про pgx.
- service ничего не знает про SQL/pgx.
- repository не знает про HTTP / gin.

## Roadmap (кратко)
- [x] Repository contracts + contract tests.
- [x] Service validation дизайн.
- [x] HTTP error mapping.
- [ ] Unit tests services (частично — skeleton добавлен).
- [ ] Game & Stats handlers.
