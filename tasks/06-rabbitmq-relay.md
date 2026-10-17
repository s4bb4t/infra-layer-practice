# 06 — RabbitMQ через Watermill + outbox relay 🟢

## Цель

Замкнуть outbox: события из таблицы реально уходят в брокер. Увидеть брокерный контракт и первую реализацию — через библиотеку-абстракцию.

## Теория

- [04-brokers](../docs/theory/04-brokers.md) — целиком
- [09-patterns](../docs/theory/09-patterns.md): outbox, idempotent consumer

## Что дано (всё)

- `contracttest/broker.go` — контракт брокера
- `memory/broker.go` — эталонный in-process брокер
- `internal/infrastructure/watermillx` — мост «Watermill ↔ наши порты»
- `internal/infrastructure/rabbit` — адаптер
- `internal/infrastructure/outbox/relay.go` — relay с юнит-тестами

## 🤔 Подумай до кода

1. Опиши топологию, которую создаёт `rabbit.Broker`: какие exchanges, какие очереди, как они связаны. Как из неё получаются fan-out между группами и competing consumers внутри группы?
2. Контракт говорит: «событие, опубликованное после возврата `Subscribe`, дойдёт». Почему в RabbitMQ событие, опубликованное **до** первого `Subscribe` группы, теряется?
3. Что делает `msg.Nack()` в Watermill AMQP? Куда девается сообщение? Что будет с порядком?
4. Что такое poison message? Как `watermillx.Consume` с ним поступает и почему именно так?
5. Relay публикует и помечает внутри одной транзакции базы. Что это даёт? Чем плохо держать транзакцию открытой во время сетевого вызова в брокер?
6. Relay упал после `Publish`, но до `COMMIT`. Что увидят потребители? Чья это проблема?
7. Что спрятал от тебя Watermill? Найди хотя бы две возможности RabbitMQ, которые через `watermillx` недоступны.

## Что сделать

1. Прочитай всё перечисленное в «Что дано». Прогони `go test -tags=integration -v ./internal/infrastructure/rabbit/`.
2. Подними локально `make infra-up PROFILES="db broker"`, открой RabbitMQ management UI (`http://localhost:15672`, guest/guest) и посмотри на exchanges/queues после прогона контракт-теста.
3. **Расширение:** очистка опубликованных событий старше N часов. Объяви **новый** порт `port.OutboxCleaner` с одним методом, реализуй его в memory и Postgres, запускай очистку из relay по таймеру, если cleaner передан. С тестом.
   - Почему отдельный порт, а не новый метод в `OutboxReader`? (Попробуй добавить метод в `OutboxReader` и посмотри, что перестанет компилироваться.)
   - Почему нельзя удалять событие сразу после `MarkPublished`?

## ✅ Definition of Done

```sh
make check
```

## ADR

`0007-...`: брокер подключён через Watermill, а не напрямую через `amqp091-go`, — что выиграли, что потеряли.

## 🎤 Вопросы на защите

- Нарисуй путь события от `PlaceOrder` до обработчика потребителя. Где оно может потеряться? Где задублироваться?
- At-least-once, at-most-once, exactly-once: что даёт эта система и почему не больше?
- Почему в контракте брокера дубликаты разрешены, а потерь нет?
