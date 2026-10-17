# 07 — Redis Streams через Watermill 🟡

## Цель

Увидеть «почти бесплатный» адаптер: благодаря порту и Watermill второй брокер — это десяток строк. И понять, что «почти бесплатно» не значит «одинаково».

## Теория

- [04-brokers](../docs/theory/04-brokers.md): Redis Streams, сравнение моделей

## Что дано

- `internal/infrastructure/redisstreams` — `New`, `Close`
- контракт-тест, контейнер

**Не дано:** `Broker.Subscribe`.

## 🤔 Подумай до кода

1. Как отображаются наши понятия на Redis Streams: topic → ?, group → ?, один `Subscribe` → ?
2. `OldestId`: `"0"` или `"$"`? Что получит **новая** группа в каждом случае? Какой вариант совпадает с поведением RabbitMQ-очереди из шага 06? Какой — с Kafka `AtStart`?
3. Что такое PEL (Pending Entries List)? Что происходит с сообщением, которое обработчик не подтвердил, а потребитель умер?
4. Stream в Redis растёт бесконечно. Как его ограничивают и чем это грозит медленной группе?
5. Redis уже стоит как кэш. Это довод брать Redis Streams как брокер — или наоборот?

## Что сделать

Реализуй `Subscribe` по образцу `rabbit.Broker.Subscribe`.

## ✅ Definition of Done

```sh
make check
go test -tags=integration -v ./internal/infrastructure/redisstreams/ | grep SKIP   # пусто
```

## ADR

`0008-...`: значение `OldestId` для новых групп и последствия для подключения нового потребителя к живой системе.

## 🎤 Вопросы на защите

- Чем Redis Streams как брокер отличается от RabbitMQ при перезапуске Redis без AOF?
- Чем отличается `XREADGROUP` от `XREAD`?
- Ты поменял одну строку (`OldestId`), и контракт всё ещё зелёный. Какое поведение системы поменялось?
