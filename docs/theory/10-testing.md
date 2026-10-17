# Как тестировать инфраструктуру

## Пирамида этого репо

| Уровень | Что | Где | Тег | Как запустить |
|---|---|---|---|---|
| Unit | домен, use case'ы на memory и fake | `internal/app/...` | — | `make test` |
| Контракт | каждый адаптер против спецификации своего порта | `internal/infrastructure/*/..._test.go` | `integration` | `make integration` |
| E2E-матрица | одни и те же сценарии на всех комбинациях хранилищ и брокеров | `test/e2e` | `e2e` | `make e2e` |
| Chaos | сценарии с задержками, обрывами, падениями | `test/chaos` | `chaos` | `make chaos` |

## Контракт-тесты

Контракт-тест — это поведенческая спецификация порта, записанная кодом **один раз** и прогоняемая против **каждой** реализации.

```go
// internal/infrastructure/postgres/postgres_test.go
contracttest.RunOrderRepository(t, func(t *testing.T) port.OrderRepository {
    return postgres.NewOrders(newPool(t))
})
```

Что это даёт:

- **Доказательство взаимозаменяемости.** Если memory и Postgres проходят один контракт, use case не отличит одно от другого. Это и есть тезис слоя, проверенный руками.
- **Быстрые unit-тесты use case'ов** на memory без контейнеров — и уверенность, что memory ведёт себя как настоящая база.
- **Новый адаптер — это зелёный контракт.** Не нужно придумывать, что проверять.

Чего контракт не проверяет: производительность, поведение под нагрузкой, специфичные возможности технологии. Для них — отдельные тесты рядом с адаптером.

### Как писать контракт

- Тестируй **поведение порта**, а не реализацию: «после Update версия стала +1», а не «выполнился такой-то SQL».
- Каждый пункт из комментария к интерфейсу должен стать тестом. Нет теста — нет гарантии.
- Изолируй тесты данными (уникальные id), а не новым контейнером на тест.
- Ошибки — тоже часть контракта: `ErrNotFound`, `ErrConflict`, `ErrAlreadyExists`.
- Конкурентность — тоже часть контракта, если порт её обещает.

### SKIP вместо FAIL для недописанного

Заглушки в этом репо возвращают `todo.Step("NN")`, и контракт-тест превращает это в SKIP. Поэтому `make check` зелёный на `main`, даже когда шаг не сделан. **Шаг сделан, когда в выводе его пакета нет ни одного SKIP:**

```sh
go test -tags=integration -v ./internal/infrastructure/mysql/ | grep SKIP   # пусто = готово
```

## testcontainers-go

Контейнеры поднимаются из кода теста, а не руками и не через `docker compose up` перед тестами:

- тест сам знает, что ему нужно;
- порты случайные — тесты можно гонять параллельно, и они не конфликтуют с локальным Postgres;
- контейнер убирается после теста (а если процесс упал — его подберёт Ryuk).

В этом репо всё про контейнеры собрано в `internal/testinfra`: `StartPostgres`, `StartKafka` и так далее. Контейнер поднимается **один раз на пакет** в `TestMain`: старт Postgres занимает секунды, тест — миллисекунды.

## E2E-матрица

`test/e2e/harness` собирает приложение целиком (без HTTP) из реальных адаптеров и гоняет одни и те же сценарии на всех комбинациях `store × broker`. Если сценарий проходит на `postgres+kafka` и падает на `mongo+nats` — значит, один из адаптеров нарушает контракт так, как контракт-тест не поймал. Это повод дописать контракт.

Асинхронные эффекты (событие в брокере, документ в индексе) проверяются через `harness.Eventually`, а не одной проверкой: сразу после use case'а их ещё нет, и это нормально.

## Chaos

Toxiproxy стоит между сервисом и Postgres/RabbitMQ и умеет добавлять задержку, обрывать соединения, ограничивать пропускную способность. Вопросы, на которые отвечают chaos-тесты:

- брокер лежит — заказы принимаются? События доходят после восстановления?
- база тормозит — запрос падает по таймауту без полузаписанных данных?
- relay упал между publish и mark — потребители переживут дубликат?

## 🔎 Изучи сам

- test pyramid, testing trophy
- consumer-driven contract testing (Pact) — родственник контракт-тестов, но между сервисами
- test doubles: dummy, stub, spy, mock, fake — в чём разница; почему здесь fake (memory), а не mock
- property-based testing (`testing/quick`, `pgregory.net/rapid`)
- build tags в Go: `//go:build integration`
- `t.Cleanup`, `TestMain`, `t.Parallel`
- testcontainers: Ryuk, wait strategies, reuse, modules
- Toxiproxy: toxics (`latency`, `timeout`, `limit_data`, `reset_peer`, `bandwidth`)
- chaos engineering: Chaos Monkey, принципы ([principlesofchaos.org](https://principlesofchaos.org/))
- Jepsen: как тестируют распределённые системы на линеаризуемость

Первоисточники:

- [testcontainers-go docs](https://golang.testcontainers.org/)
- [Toxiproxy README](https://github.com/Shopify/toxiproxy)
- Martin Fowler, [Contract Test](https://martinfowler.com/bliki/ContractTest.html), [Test Double](https://martinfowler.com/bliki/TestDouble.html)
