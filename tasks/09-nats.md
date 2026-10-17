# 09 — NATS JetStream 🔴

## Цель

Написать брокерный адаптер полностью самостоятельно: пакет, тест, подключение к контракту.

## Теория

- [04-brokers](../docs/theory/04-brokers.md): NATS JetStream
- [NATS docs: JetStream](https://docs.nats.io/nats-concepts/jetstream), [JetStream consumers](https://docs.nats.io/nats-concepts/jetstream/consumers)

## Что дано

- `testinfra.StartNATS` (JetStream включён)
- больше ничего

## 🤔 Подумай до кода

1. Core NATS vs JetStream: почему core NATS не подходит под наш контракт?
2. Как отобразить наши понятия: topic → stream + subject? group → durable consumer? Что такое push и pull consumer и какой ты выберешь?
3. `DeliverPolicy` для новой группы: `All`, `New`, `Last`? Как это соотносится с выбором в шагах 06–08?
4. Ack, Nak, NakWithDelay, Term, InProgress — когда что?
5. `AckWait`, `MaxDeliver` — какие значения разумны и что будет при `MaxDeliver` исчерпанном?
6. В JetStream есть дедупликация публикаций по `Nats-Msg-Id` в окне `Duplicates`. Как это помогает outbox relay? Почему это не отменяет идемпотентность потребителей?
7. Имена stream и consumer ограничены по символам. Как ты это обработаешь?

## Что сделать

1. `internal/infrastructure/nats`: `New(ctx, url, topic)`, `Publish`, `Subscribe`, `Close`.
2. `nats_test.go` с `//go:build integration`, `TestMain` через `testinfra.Main`, `contracttest.RunBroker`.

## ✅ Definition of Done

```sh
make check
go test -tags=integration -v ./internal/infrastructure/nats/   # PASS, без SKIP
```

## ADR

`0010-...`: конфигурация stream/consumer (storage, retention, ack policy, deliver policy, дедупликация) и почему такая.

## 🎤 Вопросы на защите

- Сравни четыре адаптера брокеров: где больше всего кода, где меньше всего контроля, где больше всего гарантий «из коробки»?
- Если бы нужно было выбрать один брокер для этого сервиса в проде, какой и почему?
