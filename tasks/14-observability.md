# 14 — Config, slog, метрики, OTel/Jaeger, DI в main.go 🔴

> **Уровень 🔴 — сам.** Дан текущий `main.go` (всё в памяти) и стенд наблюдаемости в compose.\
> Конфиг, сборку, логи, метрики и трейсы делаешь ты. Что значат уровни — в [README задач](README.md#лестница).

## Цель

**Собрать сервис из любых адаптеров по конфигу.**\
Сейчас `main.go` жёстко собирает memory. Нужно, чтобы хранилище, брокер, кэш и проекции выбирались переменными окружения при запуске ([08-cross-cutting, раздел 3](../docs/theory/08-cross-cutting.md#3-конфиг)).

**Сделать его наблюдаемым.**\
Наблюдаемость (observability) — по логам, метрикам и трейсам можно понять, что сервис делает и где сломался, не залезая в код ([08-cross-cutting](../docs/theory/08-cross-cutting.md)).

**Увидеть один запрос насквозь в Jaeger.**\
Jaeger — интерфейс для трейсов. Трейс — история одного запроса: какие операции он прошёл и сколько длилась каждая ([08-cross-cutting, раздел 7](../docs/theory/08-cross-cutting.md#trace-и-span)).

---

## Теория

| Что читать | Зачем |
|---|---|
| [08-cross-cutting](../docs/theory/08-cross-cutting.md) — целиком | composition root, DI, конфиг, запуск и остановка, slog, Prometheus, OpenTelemetry |
| [09-patterns: decorator](../docs/theory/09-patterns.md#decorator-над-портом) | инструментирование через декораторы над портами |

---

## Что дано

| Где | Что внутри |
|---|---|
| `cmd/orders/main.go` | текущая сборка: всё в памяти |
| `deploy/compose.yaml` | профили `db`, `broker`, `search`, `analytics`, `obs` |

**Профиль `obs` поднимает стенд наблюдаемости:**

| Сервис | Адрес | Зачем |
|---|---|---|
| Jaeger UI | `http://localhost:16686` | смотреть трейсы |
| Jaeger OTLP HTTP | `http://localhost:4318/v1/traces` | сюда сервис отправляет трейсы |
| Prometheus | `http://localhost:9090` | смотреть метрики |

**OTLP — протокол OpenTelemetry для передачи трейсов, метрик и логов.**\
Jaeger принимает его напрямую, отдельный «экспортёр в Jaeger» не нужен ([08-cross-cutting, раздел 7](../docs/theory/08-cross-cutting.md#opentelemetry-и-jaeger)).

---

## 🤔 Подумай до кода

> Отвечай письменно — ответы пойдут в ADR, в раздел «Ответы на вопросы задачи».\
> Ссылка 📖 после вопроса — где в теории объяснена механика. Готового ответа там нет.

**Конфиг**

1. Откуда сервис берёт конфиг? Env, файл, флаги, всё вместе? Что такое 12-factor и согласен ли ты с ним? — 📖 [08-cross-cutting: 12-factor](../docs/theory/08-cross-cutting.md#12-factor-конфиг-в-окружении)
2. Как валидировать конфиг и когда падать? — 📖 [08-cross-cutting: fail fast](../docs/theory/08-cross-cutting.md#fail-fast--падать-сразу)
3. Нужна ли библиотека (`caarlos0/env`, `envconfig`, `viper`, `koanf`) или хватит `os.Getenv`? Где граница? — 📖 [08-cross-cutting: библиотеки](../docs/theory/08-cross-cutting.md#библиотеки)
4. Как не утечь секретами (DSN с паролем) в логи? — 📖 [08-cross-cutting: секреты](../docs/theory/08-cross-cutting.md#секреты), [как спрятать чувствительные данные](../docs/theory/08-cross-cutting.md#как-спрятать-чувствительные-данные)
5. Комбинация «memory store + kafka broker» имеет смысл? Какие комбинации запрещать? — 📖 [00-map: матрица по умолчанию](../docs/theory/00-map.md#4-матрица-по-умолчанию)

**DI и жизненный цикл**

6. Сборка руками vs `wire` vs `fx`: что выбираешь для этого сервиса? — 📖 [08-cross-cutting: DI](../docs/theory/08-cross-cutting.md#2-di-руками-или-фреймворком)
7. В каком порядке стартовать и останавливать компоненты? Что будет, если закрыть пул базы раньше, чем остановится relay? — 📖 [08-cross-cutting: graceful shutdown](../docs/theory/08-cross-cutting.md#graceful-shutdown)
8. Проекторы в том же процессе, что и API, или в отдельном `cmd/projector`? — 📖 [08-cross-cutting: composition root](../docs/theory/08-cross-cutting.md#1-composition-root-maingo)

**Наблюдаемость**

9. Какие метрики нужны, чтобы ночью понять, что сервис болен? (RED, лаг outbox, ошибки потребителей) — 📖 [08-cross-cutting: что мерить](../docs/theory/08-cross-cutting.md#что-мерить-red-use-golden-signals)
10. Почему метка метрики — `r.Pattern`, а не `r.URL.Path`? — 📖 [08-cross-cutting: метки и кардинальность](../docs/theory/08-cross-cutting.md#метки-и-кардинальность)
11. Как `trace_id` попадает в каждую строку лога? — 📖 [08-cross-cutting: как связать логи, метрики и трейсы](../docs/theory/08-cross-cutting.md#8-как-связать-логи-метрики-и-трейсы)
12. Трейс HTTP → use case → БД → кэш → outbox — легко. А как связать его с публикацией из relay и обработкой у потребителя? (контекст рядом с событием, span links) — 📖 [08-cross-cutting: context propagation](../docs/theory/08-cross-cutting.md#context-propagation), [span links](../docs/theory/08-cross-cutting.md#span-links)

---

## Что сделать

#### 1. Конфиг

**Выбор хранилища, брокера, кэша и проекций — через переменные окружения.**\
Например, `ORDERS_STORE=postgres`, `ORDERS_BROKER=kafka`. Имена переменных — твоё решение; в комментарии в compose уже упомянут префикс `ORDERS_` (`ORDERS_OTLP_ENDPOINT`).

**Валидация на старте.**\
Неверный или неполный конфиг — сервис не запускается и пишет, что именно не так.

#### 2. `main.go`

**Сборка любой комбинации адаптеров.**\
`main.go` — composition root: единственное место, которое знает, какой адаптер реализует какой порт ([08-cross-cutting, раздел 1](../docs/theory/08-cross-cutting.md#1-composition-root-maingo)).

**Relay и проекторы — в фоне.**\
Они работают в горутинах рядом с HTTP-сервером.

**Graceful shutdown — корректная остановка по сигналу.**\
Перестать принимать новые запросы, доделать текущие, остановить фоновые процессы, закрыть соединения ([08-cross-cutting, раздел 4](../docs/theory/08-cross-cutting.md#graceful-shutdown)).

#### 3. Логи

**`slog` в формате JSON, в каждой записи — `trace_id` и `span_id`.**\
Тогда по строке лога можно найти трейс запроса, а по трейсу — все его логи.

#### 4. Трейсы

**OpenTelemetry SDK, экспорт в Jaeger по OTLP.**\
SDK — библиотека, которая создаёт span'ы и отправляет их. Span — одна операция внутри трейса: HTTP-запрос, транзакция, запрос в базу.

**Где нужны span'ы:**

- HTTP;
- транзакции;
- репозитории;
- outbox;
- кэш;
- публикация и потребление событий.

**Подсказка: декораторы над портами.**\
Тот же приём, что кэш в шаге 05: обёртка реализует порт, открывает span, вызывает настоящий адаптер и закрывает span ([09-patterns](../docs/theory/09-patterns.md#decorator-над-портом)).

#### 5. Метрики

**Ручка `/metrics` в формате Prometheus.**\
Prometheus сам приходит и забирает метрики по этому адресу ([08-cross-cutting, раздел 6](../docs/theory/08-cross-cutting.md#pull-модель)).

**Какие метрики:**

- RED для HTTP — число запросов, число ошибок, длительность;
- счётчик событий, которые relay переложил в брокер;
- счётчик обработанных и упавших событий у потребителей.

#### 6. HTTP-ручки поиска и отчёта

**Если не сделал в шагах 10 и 11 — сделай сейчас.**\
`GET /orders/search?q=...` и ручка отчёта по продажам.

#### 7. Посмотри глазами

**Подними стенд:**

```sh
make infra-up PROFILES="db broker obs"
```

**Запусти сервис с Postgres + Kafka + Redis, сделай заказ и оплату.**\
Потом найди трейс этого заказа в Jaeger и метрики в Prometheus.

---

### Stretch

**Протащи trace context через outbox и брокер.**\
Так, чтобы обработка события у потребителя была связана с исходным HTTP-запросом в одном трейсе или через span link.

**Stretch — задание сверх обязательного.**\
Его можно не делать, но 🤔 вопрос 12 — как раз про это, и ответить на него стоит в любом случае.

---

## ✅ Definition of Done

```sh
make check
make e2e
# + скриншот трейса из Jaeger в PR: HTTP → tx → orders.create → outbox.add
```

**`make e2e` — потому что ты менял сборку приложения.**\
Матрица из шага 12 должна остаться зелёной: декораторы и конфиг не должны сломать ни одной комбинации.

---

## ADR

**`0015-...`: конфигурация сервиса.**

- источник конфига;
- библиотека или без неё;
- валидация;
- секреты.

**`0016-...` (по желанию): что и как инструментировано, что сознательно не инструментировано.**\
«Сознательно не инструментировано» — тоже решение. У каждого span'а и каждой метрики есть цена.

---

## 🎤 Вопросы на защите

> Отвечаешь устно, без подготовленного текста. Опора — твой ADR и ответы на 🤔.

- Покажи трейс одного заказа и расскажи, где ушло время.
- Метрика `http_requests_total` выросла в 100 раз по числу серий. Что случилось?
- Что произойдёт при `SIGTERM` по шагам?
- Почему DI в Go обычно руками, а в Java/C# — контейнером?

---

## Словарик

| Термин | Коротко | Где подробнее |
|---|---|---|
| 12-factor | правила для облачных сервисов | [08-cross-cutting](../docs/theory/08-cross-cutting.md#12-factor-конфиг-в-окружении) |
| composition root | место, где адаптеры подставляются в порты | [08-cross-cutting](../docs/theory/08-cross-cutting.md#1-composition-root-maingo) |
| DI | зависимости передаются снаружи | [08-cross-cutting](../docs/theory/08-cross-cutting.md#2-di-руками-или-фреймворком) |
| graceful shutdown | корректная остановка | [08-cross-cutting](../docs/theory/08-cross-cutting.md#graceful-shutdown) |
| Jaeger | интерфейс и хранилище трейсов | [08-cross-cutting](../docs/theory/08-cross-cutting.md#opentelemetry-и-jaeger) |
| метка (label) | пара, делящая метрику на ряды | [08-cross-cutting](../docs/theory/08-cross-cutting.md#метки-и-кардинальность) |
| OTLP | протокол передачи телеметрии | [08-cross-cutting](../docs/theory/08-cross-cutting.md#opentelemetry-и-jaeger) |
| RED | rate, errors, duration | [08-cross-cutting](../docs/theory/08-cross-cutting.md#что-мерить-red-use-golden-signals) |
| `SIGTERM` | сигнал «пожалуйста, завершись» | [08-cross-cutting](../docs/theory/08-cross-cutting.md#сигналы) |
| span / trace | одна операция / история запроса | [08-cross-cutting](../docs/theory/08-cross-cutting.md#trace-и-span) |
| span link | связь span'ов без вложенности | [08-cross-cutting](../docs/theory/08-cross-cutting.md#span-links) |
| stretch | задание сверх обязательного | [выше](#stretch) |
