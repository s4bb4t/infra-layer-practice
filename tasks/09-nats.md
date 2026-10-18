# 09 — NATS JetStream 🔴

> **Уровень 🔴 — сам.** Дан только контейнер с JetStream.\
> Пакет, тест и подключение к контракту — твои. Что значат уровни — в [README задач](README.md#лестница).

## Цель

**Написать брокерный адаптер полностью самостоятельно.**\
Пакет `internal/infrastructure/nats`, файл теста, подключение к `contracttest.RunBroker` — всё с нуля, по образцу уже готовых адаптеров.

**NATS — лёгкий брокер сообщений, JetStream — его встроенный слой хранения.**\
Сам по себе NATS доставляет сообщения только тем, кто подключён прямо сейчас. JetStream добавляет хранение, подтверждения и повторную доставку ([04-brokers, раздел 11](../docs/theory/04-brokers.md#11-nats-jetstream)).

---

## Теория

| Что читать | Зачем |
|---|---|
| [04-brokers: NATS JetStream](../docs/theory/04-brokers.md#11-nats-jetstream) | core и JetStream, subjects, stream и consumer, подтверждения, дедупликация |
| [NATS docs: JetStream](https://docs.nats.io/nats-concepts/jetstream) | официальная документация, англ. |
| [JetStream consumers](https://docs.nats.io/nats-concepts/jetstream/consumers) | все параметры consumer, англ. |

---

## Что дано

| Где | Что внутри |
|---|---|
| `testinfra.StartNATS` | контейнер NATS с включённым JetStream |
| больше ничего | |

**Клиент — `github.com/nats-io/nats.go`, пакет `jetstream`.**\
Это новый API JetStream. Старый API (`nc.JetStream()` в корне пакета `nats`) авторы называют legacy — для нового кода берут `jetstream`.

---

## 🤔 Подумай до кода

> Отвечай письменно — ответы пойдут в ADR, в раздел «Ответы на вопросы задачи».\
> Ссылка 📖 после вопроса — где в теории объяснена механика. Готового ответа там нет.

1. Core NATS vs JetStream: почему core NATS не подходит под наш контракт? — 📖 [04-brokers: core NATS и JetStream](../docs/theory/04-brokers.md#core-nats-и-jetstream)
2. Как отобразить наши понятия: topic → stream + subject? group → durable consumer? Что такое push и pull consumer и какой ты выберешь? — 📖 [04-brokers: stream и consumer](../docs/theory/04-brokers.md#stream-и-consumer), [push и pull](../docs/theory/04-brokers.md#push-и-pull-consumers)
3. `DeliverPolicy` для новой группы: `All`, `New`, `Last`? Как это соотносится с выбором в шагах 06–08? — 📖 [04-brokers: откуда начинает новая группа](../docs/theory/04-brokers.md#откуда-начинает-новая-группа)
4. Ack, Nak, NakWithDelay, Term, InProgress — когда что? — 📖 [04-brokers: подтверждения в JetStream](../docs/theory/04-brokers.md#подтверждения-в-jetstream)
5. `AckWait`, `MaxDeliver` — какие значения разумны и что будет при `MaxDeliver` исчерпанном? — 📖 [04-brokers: подтверждения в JetStream](../docs/theory/04-brokers.md#подтверждения-в-jetstream)
6. В JetStream есть дедупликация публикаций по `Nats-Msg-Id` в окне `Duplicates`. Как это помогает outbox relay? Почему это не отменяет идемпотентность потребителей? — 📖 [04-brokers: дедупликация публикаций](../docs/theory/04-brokers.md#дедупликация-публикаций)
7. Имена stream и consumer ограничены по символам. Как ты это обработаешь? — 📖 [04-brokers: ещё в NATS](../docs/theory/04-brokers.md#ещё-в-nats)

---

## Что сделать

#### 1. Пакет `internal/infrastructure/nats`

**Четыре функции:**

- `New(ctx, url, topic)` — подключиться, создать или обновить stream;
- `Publish` — опубликовать события;
- `Subscribe` — создать durable consumer для группы и запустить потребление;
- `Close` — остановить подписки и закрыть соединение.

**Проверка соответствия порту — на этапе компиляции.**\
Как в других адаптерах: `var _ port.EventPublisher = (*Broker)(nil)` и то же для `port.EventSubscriber`. Если метод не совпадает по сигнатуре, код не соберётся.

**Durable consumer — consumer с именем, который хранит свою позицию на сервере.**\
Перезапустился сервис — consumer продолжит с того места, где остановился ([04-brokers, раздел 11](../docs/theory/04-brokers.md#stream-и-consumer)).

#### 2. Тест `nats_test.go`

**Три части, всё по образцу `rabbit_test.go`:**

1. `//go:build integration` первой строкой;
2. `TestMain` через `testinfra.Main` и `testinfra.StartNATS`;
3. `contracttest.RunBroker` с фабрикой, которая создаёт брокер на новый топик для каждого подтеста.

**Новый топик на подтест — чтобы события одного подтеста не попадали в другой.**\
Как устроена фабрика и зачем `t.Cleanup` — в [10-testing, раздел 3](../docs/theory/10-testing.md#3-инструменты-go-для-тестов).

---

## ✅ Definition of Done

```sh
make check
go test -tags=integration -v ./internal/infrastructure/nats/   # PASS, без SKIP
```

---

## ADR

**`0010-...`: конфигурация stream и consumer и почему такая.**

- storage — где хранится stream: в памяти или на диске;
- retention — когда сообщения удаляются;
- ack policy — как подтверждаются сообщения;
- deliver policy — откуда начинает новая группа;
- дедупликация — окно `Duplicates` и что кладёшь в `Nats-Msg-Id`.

---

## 🎤 Вопросы на защите

> Отвечаешь устно, без подготовленного текста. Опора — твой ADR и ответы на 🤔.

- Сравни четыре адаптера брокеров: где больше всего кода, где меньше всего контроля, где больше всего гарантий «из коробки»?
- Если бы нужно было выбрать один брокер для этого сервиса в проде, какой и почему?

---

## Словарик

| Термин | Коротко | Где подробнее |
|---|---|---|
| ack policy / deliver policy | как подтверждать / откуда начинать | [04-brokers](../docs/theory/04-brokers.md#подтверждения-в-jetstream) |
| core NATS | доставка без хранения | [04-brokers](../docs/theory/04-brokers.md#core-nats-и-jetstream) |
| durable consumer | consumer с именем и сохранённой позицией | [04-brokers](../docs/theory/04-brokers.md#stream-и-consumer) |
| JetStream | слой хранения в NATS | [04-brokers](../docs/theory/04-brokers.md#11-nats-jetstream) |
| `Nats-Msg-Id` | заголовок для дедупликации публикаций | [04-brokers](../docs/theory/04-brokers.md#дедупликация-публикаций) |
| push / pull consumer | сервер шлёт сам / клиент запрашивает | [04-brokers](../docs/theory/04-brokers.md#push-и-pull-consumers) |
| retention | правило удаления сообщений из stream | [04-brokers](../docs/theory/04-brokers.md#stream-и-consumer) |
| stream | хранилище сообщений по набору subjects | [04-brokers](../docs/theory/04-brokers.md#stream-и-consumer) |
| subject | адрес сообщения, вроде `orders.placed` | [04-brokers](../docs/theory/04-brokers.md#subjects-и-wildcards) |
