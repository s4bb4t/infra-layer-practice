# 05 — Redis: кэш-декоратор 🔴

## Цель

Добавить кэш так, чтобы use case'ы о нём не узнали: декоратор над `OrderRepository`. Написать контракт-тест для нового порта самому.

## Теория

- [03-cache](../docs/theory/03-cache.md) — целиком
- [09-patterns](../docs/theory/09-patterns.md): decorator, cache-aside

## Что дано

- `port.Cache` в `internal/app/port/port.go`
- `testinfra.StartRedis`
- больше ничего

## 🤔 Подумай до кода

1. Чем отличаются cache-aside, read-through, write-through, write-behind? Какой здесь уместен и почему?
2. При `Update` положить новое значение в кэш или удалить ключ? Что будет в каждом варианте, если транзакция потом откатится?
3. Опиши гонку: читатель промахнулся, писатель обновил и инвалидировал, читатель положил в кэш старое значение. Можно ли её полностью убрать? Что её ограничивает?
4. Redis упал. Что должен делать `Get` декоратора: вернуть ошибку или пойти в базу?
5. Кэшировать ли `ListByCustomer`? Что придётся инвалидировать при каждом новом заказе?
6. Как сериализовать `Order` в кэш? JSON, gob, protobuf, msgpack — что важно в этом выборе?
7. Как назвать ключи? Зачем префикс?

## Что сделать

1. `internal/infrastructure/contracttest/cache.go` — `RunCache`: промах, set/get, перезапись, удаление нескольких ключей, истечение TTL, отсутствие алиасинга байтов.
2. `internal/infrastructure/memory/cache.go` — memory-реализация `port.Cache`, проходит `RunCache`.
3. `internal/infrastructure/redis` — `port.Cache` на `go-redis`, проходит `RunCache` против контейнера.
4. `internal/infrastructure/cache` — декоратор `OrderRepository` (cache-aside + инвалидация). Он сам проходит `RunOrderRepository`.
5. Юнит-тесты декоратора: второй `Get` не идёт в базу; `Update` инвалидирует; сломанный кэш не ломает чтения.

## ✅ Definition of Done

```sh
make check
go test -tags=integration -v ./internal/infrastructure/redis/ ./internal/infrastructure/cache/ | grep -E 'SKIP|FAIL'   # пусто
```

## ADR

`0006-...`: стратегия кэширования и инвалидации заказов: что кэшируем, что нет, TTL и почему такой, какую несогласованность принимаем.

## 🎤 Вопросы на защите

- Почему декоратор, а не вызовы кэша внутри use case'а?
- Как декоратор проходит контракт-тест репозитория и что это доказывает?
- Что такое thundering herd и как бы ты его здесь лечил?
- Что изменится, если сервис запущен в 10 инстансах и кэш in-process, а не в Redis?
