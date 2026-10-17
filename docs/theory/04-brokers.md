# Брокеры: Kafka, NATS JetStream, RabbitMQ, Redis Streams

**Роль:** доставка событий между частями системы.
**Отвечает на вопрос:** как сообщить другим, что что-то произошло, не вызывая их напрямую и не теряя сообщений.

## Когда брать

- У изменения несколько потребителей (поиск, аналитика, уведомления), и их число растёт.
- Нужно развязать сервисы по времени: продюсер не ждёт, пока потребитель обработает.
- Нужно сгладить пики: очередь принимает всплеск, потребители разбирают в своём темпе.
- Нужно переиграть историю событий (только log-based брокеры: Kafka, JetStream, Redis Streams).

## Когда не брать

- Нужен ответ прямо сейчас (синхронный запрос-ответ). Брокер умеет request-reply, но это не его сильная сторона.
- Потребитель один и живёт в том же процессе — хватит канала или вызова функции.
- Нужна точность «ровно один раз» end-to-end. Ни один брокер этого не даёт без идемпотентных потребителей (см. [09-patterns](09-patterns.md)).

## Две модели

**Очередь (RabbitMQ):** сообщение лежит в очереди, пока его не подтвердили, потом удаляется. Брокер умный: маршрутизация через exchanges, ack/nack на каждое сообщение, dead-letter очереди, приоритеты.

**Лог (Kafka, JetStream, Redis Streams):** сообщения дописываются в конец лога и хранятся по времени или размеру, независимо от того, прочитали их или нет. Потребитель хранит **позицию** (offset). Несколько групп читают один лог независимо, любую можно перемотать назад.

## Сравнение

| | Kafka | NATS JetStream | RabbitMQ | Redis Streams |
|---|---|---|---|---|
| Модель | лог, партиции | лог (stream), consumers | очереди + exchanges | лог в Redis |
| Подтверждение | commit offset (позиция) | ack на сообщение | ack/nack на сообщение | `XACK` на сообщение, PEL |
| Порядок | внутри партиции | внутри stream (с оговорками при redelivery) | внутри очереди (нарушается при nack/requeue) | внутри stream |
| Масштабирование потребителей | ≤ числа партиций в группе | любое (pull consumers) | любое | любое в группе |
| Повтор истории | да | да | нет (кроме streams в RabbitMQ 3.9+) | да |
| Дедупликация на брокере | idempotent producer (в рамках сессии) | `Nats-Msg-Id` в окне `Duplicates` | нет | нет |
| Эксплуатация | тяжёлая (KRaft упростил) | лёгкая, один бинарник | средняя | у тебя уже есть Redis |
| Пропускная способность | очень высокая | высокая | средняя | зависит от памяти Redis |
| Типичный выбор | event streaming, большие объёмы, интеграция многих сервисов | лёгкий брокер для микросервисов, edge, IoT | task queues, сложная маршрутизация | «брокер без нового брокера» |

## Что важно понимать

### Гарантии доставки

- **At-most-once:** отправили и забыли. Может потеряться.
- **At-least-once:** повторяем, пока не подтвердят. Может прийти дважды. Это режим всех адаптеров в этом репо.
- **Exactly-once:** в общем случае недостижимо через сеть. Kafka EOS работает только внутри Kafka (read-process-write в транзакции). Снаружи exactly-once = at-least-once + идемпотентный потребитель.

### Consumer groups

Все четыре брокера в этом репо реализуют один контракт (`contracttest.RunBroker`):

- каждая группа получает каждое событие (fan-out между группами);
- внутри группы событие обрабатывает один потребитель (competing consumers);
- если обработчик вернул ошибку, событие придёт снова.

Как каждый брокер это реализует — разное. Это и есть содержание задач 06–09.

### Kafka: нет nack

В Kafka нельзя сказать «это сообщение не получилось, отдай потом, а следующие давай». Можно только сдвинуть позицию. Если одно событие не обрабатывается, ты выбираешь: блокировать партицию и ретраить на месте, отправить событие в retry/DLQ-топик и идти дальше, или пропустить. Каждый вариант — это решение для ADR.

### Порядок

Порядок гарантирован только в пределах партиции, очереди или stream. Даже там redelivery его ломает: событие 1 упало, событие 2 обработалось, событие 1 пришло снова. Поэтому в конверте события есть `Version`, а потребители сравнивают версии, а не полагаются на порядок прихода.

### Watermill

Библиотека, которая прячет брокеры за общими `Publisher`/`Subscriber`. Плюсы: один код на десяток брокеров, готовые middleware (retry, poison queue, throttling). Минусы: наименьший общий знаменатель — специфичные возможности брокера (Kafka-партиции, JetStream-дедупликация) спрятаны или недоступны. В этом репо RabbitMQ и Redis Streams подключены через Watermill, а Kafka и NATS написаны руками. Сравни объём кода и то, что ты контролируешь.

## 🔎 Изучи сам

- at-most-once / at-least-once / exactly-once; idempotent consumer
- Kafka: partition, offset, consumer group, rebalance, ISR, `acks=all`, `min.insync.replicas`, idempotent producer, transactions, compaction, retention, KRaft vs ZooKeeper
- Kafka: Schema Registry, Avro / Protobuf, schema evolution, compatibility modes
- RabbitMQ: exchange types (direct, fanout, topic, headers), bindings, prefetch (QoS), publisher confirms, dead letter exchange, quorum queues, streams
- NATS: core NATS vs JetStream, subjects и wildcards, push vs pull consumers, `AckWait`, `MaxDeliver`, `Nats-Msg-Id`, KV и Object Store
- Redis Streams: `XADD`, `XREADGROUP`, PEL, `XAUTOCLAIM`, `MAXLEN`
- dead letter queue, poison message, retry topics с задержкой
- backpressure
- sarama vs franz-go vs confluent-kafka-go — почему в этом репо franz-go

Первоисточники:

- [Kafka docs: Design](https://kafka.apache.org/documentation/#design)
- [franz-go README](https://github.com/twmb/franz-go)
- [NATS docs: JetStream](https://docs.nats.io/nats-concepts/jetstream)
- [RabbitMQ tutorials](https://www.rabbitmq.com/tutorials) (1–5) и [Reliability Guide](https://www.rabbitmq.com/docs/reliability)
- [Redis Streams introduction](https://redis.io/docs/latest/develop/data-types/streams/)
- [Watermill docs](https://watermill.io/docs/)
- Jay Kreps, [The Log: What every software engineer should know](https://engineering.linkedin.com/distributed-systems/log-what-every-software-engineer-should-know-about-real-time-datas-unifying)
