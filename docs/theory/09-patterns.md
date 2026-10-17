# Паттерны-«клей»

Технологии из карты сами по себе — набор коробок. Систему из них делают несколько паттернов. Каждый из них в этом репо реализован в коде; ссылки ведут туда.

## Transactional outbox

**Проблема.** Нужно сохранить заказ И отправить событие. Это две разные системы, и общей транзакции у них нет.

- Сначала commit, потом publish: процесс упал между ними — событие потеряно навсегда.
- Сначала publish, потом commit: commit не удался — событие о заказе, которого нет.
- «Распределённая транзакция» (2PC/XA) — почти никто из брокеров её не поддерживает, а те, кто поддерживает, медленные и хрупкие.

**Решение.** Событие пишется в таблицу `outbox` **в той же транзакции**, что и заказ. Отдельный процесс (relay) читает outbox и публикует в брокер, потом помечает событие отправленным.

**Следствия:**

- Доставка **at-least-once**: relay опубликовал, но упал до пометки — опубликует ещё раз.
- Нужен порядок чтения outbox (`seq`) и защита от двух relay, читающих одно и то же (`FOR UPDATE SKIP LOCKED`).
- Outbox растёт — нужна очистка опубликованного.

В коде: `usecase.PlaceOrder`, `port.Outbox`, `internal/infrastructure/outbox/relay.go`, таблица `outbox` в миграциях.

**Альтернативы:** CDC (Debezium читает WAL/binlog и публикует сам), listen-to-yourself (сначала событие, потом сервис сам себя обновляет из брокера), event sourcing (событие — и есть источник истины).

## Idempotent consumer

**Проблема.** At-least-once означает дубликаты. Если обработчик «начислить бонус» выполнится дважды — бонуса будет два.

**Решения:**

- **Дедупликация по ID события:** таблица обработанных ID в той же транзакции, что и эффект.
- **Естественная идемпотентность:** `UPSERT`, `SET status = 'paid'` вместо `INSERT` и инкремента.
- **Версии:** хранить версию агрегата и игнорировать события с версией ≤ сохранённой. Это решает и дубликаты, и неправильный порядок. Так сделан проектор в OpenSearch (external versioning).
- **Дедупликация на стороне хранилища:** `ReplacingMergeTree` в ClickHouse.

## Optimistic locking

**Проблема.** Два запроса одновременно прочитали заказ v1, оба поменяли, оба сохранили. Второй затёр первого (lost update).

**Решение.** `UPDATE … SET …, version = version + 1 WHERE id = $1 AND version = $2`. Если строка не обновилась — кто-то успел раньше: `ErrConflict`. Клиент перечитывает и решает, повторять ли.

**Альтернатива:** pessimistic locking (`SELECT … FOR UPDATE`) — держит блокировку на время транзакции. Проще рассуждать, хуже под конкуренцией, и опасно, если в транзакции есть сетевые вызовы.

В коде: `port.OrderRepository.Update`, контракт-тест «concurrent updates of one version: exactly one wins».

## Cache-aside + инвалидация

См. [03-cache](03-cache.md). В коде: `internal/infrastructure/cache` (появится в задаче 05).

## Read model (CQRS-lite)

**Проблема.** Одна модель данных не может быть одновременно удобной для транзакционной записи, полнотекстового поиска и агрегатов по истории.

**Решение.** Источник истины — нормализованная транзакционная база. Для каждого вида чтения строится отдельная **производная модель** (search index, таблица фактов в ClickHouse), которую наполняют проекторы из потока событий. Каждую такую модель можно удалить и пересобрать.

**Цена:** eventual consistency. Заказ оплачен, а поиск ещё секунду показывает «created». Интерфейс и пользователи должны быть к этому готовы.

## Decorator над портом

Кэш, трейсинг, метрики, ретраи — это `OrderRepository`, который оборачивает другой `OrderRepository`. Use case о них не знает, всё включается в `main.go`. Каждый декоратор проходит тот же контракт-тест, что и обёрнутый адаптер.

## 🔎 Изучи сам

- dual write problem
- transactional outbox, transaction log tailing, polling publisher
- CDC, Debezium, logical decoding (Postgres), binlog (MySQL), change streams (Mongo)
- inbox pattern
- saga: orchestration vs choreography; compensating transactions
- 2PC, XA и почему их избегают
- CQRS, event sourcing — и почему это не одно и то же
- idempotency key (на уровне HTTP API: Stripe)
- retry с exponential backoff и jitter, circuit breaker, bulkhead, timeout budget

Первоисточники:

- Chris Richardson, [microservices.io: Transactional outbox](https://microservices.io/patterns/data/transactional-outbox.html), [Idempotent consumer](https://microservices.io/patterns/communication-style/idempotent-consumer.html), [Saga](https://microservices.io/patterns/data/saga.html)
- Martin Fowler, [CQRS](https://martinfowler.com/bliki/CQRS.html)
- Stripe, [Designing robust and predictable APIs with idempotency](https://stripe.com/blog/idempotency)
- AWS Builders' Library, [Timeouts, retries and backoff with jitter](https://aws.amazon.com/builders-library/timeouts-retries-and-backoff-with-jitter/)
