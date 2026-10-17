# Сквозные вопросы: конфиг, логи, метрики, трейсы, DI

**Роль:** то, что не относится ни к одному адаптеру, но без чего систему нельзя запустить, понять и починить.
**Отвечает на вопрос:** как собрать всё это вместе, увидеть, что происходит, и найти, где сломалось.

## Composition root: `main.go`

Единственное место, которое знает, какой адаптер реализует какой порт. Здесь:

1. читается и валидируется конфиг;
2. создаются клиенты (пулы, соединения);
3. адаптеры заворачиваются в декораторы (трейсинг, кэш, метрики);
4. из адаптеров собираются use case'ы;
5. стартуют фоновые процессы (outbox relay, консьюмеры);
6. стартует HTTP;
7. по сигналу всё останавливается в обратном порядке (graceful shutdown).

### DI: руками или фреймворком

В Go чаще всего собирают руками: это просто вызовы конструкторов, и компилятор проверяет, что всё сошлось. Когда графа зависимостей становится много, берут:

- **кодогенерацию** (`google/wire`) — граф проверяется на этапе компиляции, в рантайме магии нет;
- **рантайм-контейнеры** (`uber-go/fx`, `uber-go/dig`) — гибко, есть lifecycle-хуки, но ошибки проявляются только при запуске.

Вопрос для задачи 14: где проходит граница, после которой ручная сборка становится хуже фреймворка?

## Конфиг

- **12-factor:** конфиг живёт в окружении (env vars), а не в коде и не в файле внутри образа.
- **Валидация на старте (fail fast):** опечатка в `ORDERS_STORE` должна уронить процесс сразу, а не через час на первом запросе.
- **Секреты** — отдельная тема: не в git, не в логах, в идеале — из secret manager (Vault, AWS Secrets Manager, Kubernetes Secrets).
- **Библиотеки:** `os.Getenv` руками, `caarlos0/env`, `kelseyhightower/envconfig`, `spf13/viper`, `knadh/koanf`. Выбор — в ADR задачи 14.

## Логи: `log/slog`

- **Структурированные** (JSON), а не `fmt.Printf`: логи читают машины (Loki, Elasticsearch), а люди фильтруют по полям.
- **Уровни:** Debug — для разработки, Info — значимые события жизненного цикла, Warn — деградация, которую система пережила, Error — запрос или операция не удалась.
- **`trace_id` в каждой строке** связывает лог с трейсом. Для этого используют `*Context`-методы (`InfoContext`) и handler, который достаёт span из контекста.
- **Не логируй и не возвращай одновременно.** Ошибка логируется один раз — там, где её обработали (обычно на верхнем уровне).

## Метрики: Prometheus

- **Pull-модель:** сервис отдаёт `/metrics`, Prometheus приходит и забирает.
- **Типы:** counter (только растёт), gauge (текущее значение), histogram (распределение, отсюда перцентили).
- **RED** для сервисов: Rate, Errors, Duration. **USE** для ресурсов: Utilization, Saturation, Errors.
- **Кардинальность** — главный враг. Метка `path="/orders/0191f..."` с id внутри создаёт новую временную серию на каждый заказ, и Prometheus умирает. Метка должна быть шаблоном маршрута (`r.Pattern`), а не путём.

## Трейсы: OpenTelemetry + Jaeger

- **Trace** — дерево **spans** одного запроса через все компоненты. Span — операция с началом, длительностью, атрибутами и статусом.
- **OpenTelemetry** — вендор-нейтральный стандарт и SDK. **Jaeger** — один из бэкендов, который принимает OTLP, хранит и показывает трейсы.
- **Context propagation:** внутри процесса span живёт в `context.Context`, между процессами — в заголовках (`traceparent` по W3C Trace Context). Через брокер контекст тоже можно протащить — в заголовках сообщения. Через outbox — только если сохранить его рядом с событием.
- **Сэмплирование:** в проде пишут не все трейсы (head sampling 1–10% или tail sampling «все медленные и все с ошибками»).

Цель задачи 14 — увидеть в Jaeger один трейс, в котором есть HTTP-запрос, use case, транзакция, запрос к базе, обращение к кэшу. Публикация в брокер случается позже, в relay, и в отдельном трейсе. Подумай, как их связать.

## 🔎 Изучи сам

- The Twelve-Factor App
- composition root, dependency injection, service locator (и почему последний считают антипаттерном)
- `google/wire`, `uber-go/fx`
- graceful shutdown: `signal.NotifyContext`, `http.Server.Shutdown`, порядок остановки, `errgroup`
- structured logging, `slog.Handler`, log levels, log sampling
- RED и USE методы, Google SRE «four golden signals»
- Prometheus: counter, gauge, histogram, summary; labels, cardinality; PromQL `rate()`, `histogram_quantile()`
- OpenTelemetry: traces, metrics, logs; SDK vs API; OTLP; exporters; resource attributes; semantic conventions
- W3C Trace Context, `traceparent`, baggage
- head vs tail sampling
- span links (как связать producer и consumer, когда это не parent-child)
- health checks: liveness vs readiness

Первоисточники:

- [The Twelve-Factor App](https://12factor.net/)
- [Go blog: Structured Logging with slog](https://go.dev/blog/slog)
- [Prometheus docs: Instrumentation best practices](https://prometheus.io/docs/practices/instrumentation/)
- [OpenTelemetry Go docs](https://opentelemetry.io/docs/languages/go/)
- Google SRE Book, [Monitoring Distributed Systems](https://sre.google/sre-book/monitoring-distributed-systems/)
