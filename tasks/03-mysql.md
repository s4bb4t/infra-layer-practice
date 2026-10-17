# 03 — MySQL 🟡

## Цель

Реализовать тот же порт на другой реляционной базе и увидеть, что отличается: диалект, уровень изоляции, семантика драйвера. Доказать взаимозаменяемость тем же контрактом.

## Теория

- [01-sql](../docs/theory/01-sql.md): таблица Postgres vs MySQL

## Что дано

- `internal/infrastructure/mysql` — миграция, `Create`, `Get`, `ListByCustomer`, outbox
- контракт-тесты, контейнер

**Не дано (`todo.Step("03")`):** `TxManager.Do`, `Orders.Update`.

## 🤔 Подумай до кода

1. Какой уровень изоляции у InnoDB по умолчанию? Чем он отличается от READ COMMITTED в Postgres? Нужно ли его менять для этого кода?
2. `RowsAffected` в MySQL по умолчанию возвращает число **изменённых**, а не **найденных** строк. Почему в нашем `Update` это не ломает логику? Когда сломало бы? Что такое `clientFoundRows`?
3. У `database/sql` нет параметров-массивов. Как выглядит `MarkPublished` и почему?
4. `DATETIME(6)` vs `TIMESTAMP` в MySQL — в чём разница и почему в DSN стоит `loc=UTC`?
5. В Postgres DDL транзакционный, в MySQL — нет. Чем это опасно для миграций?

## Что сделать

1. Реализуй `TxManager.Do` по образцу `postgres.TxManager.Do`.
2. Реализуй `Orders.Update` с optimistic locking.
3. Прогони контракт.

## ✅ Definition of Done

```sh
make check
go test -tags=integration -v ./internal/infrastructure/mysql/ | grep SKIP   # пусто
```

## ADR

`0004-...`: уровень изоляции для MySQL-адаптера — оставляем по умолчанию или меняем, и почему.

## 🎤 Вопросы на защите

- Покажи три места, где MySQL-адаптер отличается от Postgres-адаптера. Какое из отличий могло бы стать багом, не будь контракт-теста?
- Что такое gap lock и next-key lock? Где они могут появиться в этом коде?
- Postgres и MySQL проходят один контракт. Значит ли это, что они ведут себя одинаково? Чего контракт не проверяет?
