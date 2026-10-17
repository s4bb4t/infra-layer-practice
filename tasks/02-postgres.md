# 02 — Postgres: репозиторий, транзакции, outbox 🟢 + расширение

## Цель

Увидеть, как порт из шага 01 реализуется на настоящей базе, и как одни и те же контракт-тесты гоняются против контейнера. Сделать первое маленькое расширение руками.

## Теория

- [01-sql](../docs/theory/01-sql.md) — целиком
- [09-patterns](../docs/theory/09-patterns.md): outbox
- [10-testing](../docs/theory/10-testing.md): testcontainers

## Что дано

- `internal/infrastructure/postgres` — `TxManager`, `Orders`, `Outbox`, миграции через goose + `embed`
- `internal/testinfra/postgres.go` — контейнер
- `postgres_test.go` — контракт-тесты

**Не дано:** `Orders.ListByCustomer` и индекс под него.

## 🤔 Подумай до кода

1. Почему `pgx`, а не `database/sql` + `lib/pq`? Почему не ORM (`gorm`, `ent`)? Что такое ORM вообще и что ты теряешь и получаешь с ним?
2. `items` хранится как `JSONB`, а не отдельной таблицей `order_items`. Когда это хорошее решение, а когда плохое?
3. `Update` использует `WHERE version = $2`, а не `SELECT … FOR UPDATE`. Почему это работает на READ COMMITTED? Что именно происходит, когда два `UPDATE` одной строки идут одновременно?
4. В `Outbox.Fetch` есть `FOR UPDATE SKIP LOCKED`. От какой проблемы это защищает? Что будет без `SKIP LOCKED`, но с `FOR UPDATE`?
5. Порядок outbox задаётся `seq` (identity). Может ли relay увидеть событие с `seq = 11` раньше, чем `seq = 10`, если они из разных транзакций? Чем это грозит?
6. Почему в `TxManager.Do` rollback делается с `context.WithoutCancel(ctx)`?
7. Почему миграции встроены в бинарник, а не лежат рядом?

## Что сделать

1. Прогони `go test -tags=integration -v ./internal/infrastructure/postgres/`. Два теста про `ListByCustomer` — SKIP.
2. Реализуй `Orders.ListByCustomer`.
3. Добавь миграцию `migrations/00002_*.sql` с индексом под этот запрос.
4. Проверь план запроса: насыпь в локальную базу (`make infra-up`) 100k заказов, сравни `EXPLAIN ANALYZE` до и после индекса. Вставь оба плана в ADR.

## ✅ Definition of Done

```sh
make check
go test -tags=integration -v ./internal/infrastructure/postgres/ | grep SKIP   # пусто
```

## ADR

`0003-...`: индекс под `ListByCustomer` — какой именно, почему такой порядок колонок, что показал `EXPLAIN ANALYZE`, сколько стоит индекс на запись.

## 🎤 Вопросы на защите

- Расскажи по шагам, что происходит в Postgres при `UPDATE orders SET … WHERE id = $1 AND version = $2` у двух транзакций одновременно.
- Чем `(customer_id, created_at DESC)` лучше `(created_at, customer_id)` для этого запроса?
- Что будет, если два инстанса сервиса одновременно стартуют и оба применяют миграции?
- Что такое MVCC и почему из-за него в Postgres нужен VACUUM?
