# infra-layer-practice

Практика по инфраструктурному слою. **Один сервис заказов и много адаптеров.**

Интерфейсы (порты) объявлены во внутренних слоях. Каждая инфраструктурная технология реализует какой-то из них, и все реализации одного порта проходят **один и тот же** набор контракт-тестов. Контейнеры поднимаются из тестов через testcontainers-go, а не руками. Вопросы «что это, какие аналоги, чем отличаются, когда брать» ты разбираешь в ADR после каждого шага и защищаешь устно.

```
internal/app/domain      ← бизнес-правила, импортирует только stdlib
internal/app/port        ← что приложение требует от мира (интерфейсы)
internal/app/usecase     ← оркестрация через порты
internal/infrastructure  ← реализации портов: memory, postgres, mysql, mongo, redis, rabbit, kafka, nats, ...
internal/presentation    ← HTTP
cmd/orders               ← composition root: какой адаптер реализует какой порт
```

Главный тезис раздела виден прямо в импортах: `infrastructure` импортирует `app`, а `app` не импортирует ничего из `infrastructure`.

## С чего начать

1. Прочитай [карту инфраструктуры](docs/theory/00-map.md).
2. Прочитай [что такое ADR](docs/adr/README.md).
3. Открой [список задач](tasks/README.md) и начни с [01](tasks/01-core.md).

## Требования

- Go 1.27+
- Docker (Docker Desktop, OrbStack, colima) — **запущенный**: интеграционные тесты поднимают контейнеры
- `make`
- `golangci-lint` ставить не нужно: Makefile запускает закреплённую версию через `go run`

Первый прогон интеграционных тестов скачивает образы (несколько ГБ суммарно) — это долго один раз.

## ⚠️ Перед каждым push — обязательно

```sh
make check
```

`make check` — это:

| Шаг | Команда | Что ловит |
|---|---|---|
| форматирование | `gofmt -l .` должен быть пуст | неотформатированный код |
| vet | `go vet ./...` (и со всеми тегами) | подозрительные конструкции |
| линтер | `golangci-lint run` (конфиг `.golangci.yml`) | errcheck, staticcheck, errorlint, sqlclosecheck, ... |
| unit | `go test -race ./...` | домен, use case'ы, memory |
| integration | `go test -race -tags=integration ./internal/infrastructure/...` | контракт-тесты адаптеров на настоящих контейнерах |

**Поставь хук один раз после клонирования:**

```sh
make hooks
```

После этого `git push` сам запускает `make check` и не пускает push, пока он не зелёный. CI на PR гоняет то же самое: если не прошло локально, не пройдёт и там.

Длинные наборы (`make e2e`, `make chaos`) не входят в `check` и в pre-push. Их запускаешь сам на шагах 12–14 и перед PR этих шагов. В CI они запускаются вручную (Actions → ci → Run workflow).

## Как сдавать шаг

```sh
git switch main && git pull
git switch -c step-NN
# ... работа по tasks/NN-*.md ...
make check
git push -u origin step-NN
# открыть PR в main этого репозитория — шаблон описания подставится сам
```

- В PR: код шага + ADR в `docs/adr/`.
- Шаблон описания PR: [.github/pull_request_template.md](.github/pull_request_template.md), пример заполненного: [docs/pr-example.md](docs/pr-example.md).
- Ментор ревьюит PR вместе с ADR, потом — **устная защита** по вопросам из задачи.
- После апрува — merge в `main`, следующий шаг от свежего `main`.

## Как понять, что шаг сделан

Заглушки в коде возвращают `todo.Step("NN")`. Контракт-тесты показывают такие методы как **SKIP**, поэтому `make check` зелёный даже на недописанном `main`. Шаг сделан, когда в тестах его пакета не осталось SKIP:

```sh
go test -tags=integration -v ./internal/infrastructure/mysql/ | grep SKIP   # пусто = готово
```

## Команды

```sh
make help         # все цели
make check        # всё обязательное перед push
make test         # только unit
make integration  # контракт-тесты адаптеров (Docker)
make e2e          # матрица store × broker (долго; E2E_STORES=… E2E_BROKERS=… сужают)
make chaos        # тесты отказов (долго)
make run          # запустить сервис на :8080
make infra-up PROFILES="db broker obs"   # локальная инфраструктура для ручного запуска
make infra-down
```

## Карта репозитория

| Путь | Что там |
|---|---|
| `docs/theory/` | теория по ролям: когда брать, когда не брать, чем отличаются; блоки 🔎 «Изучи сам» |
| `docs/adr/` | ADR: введение, шаблон, пример и твои решения |
| `tasks/` | 15 задач — по одной на шаг |
| `internal/infrastructure/contracttest/` | контракт-тесты портов |
| `internal/testinfra/` | старт контейнеров для тестов |
| `internal/todo/` | маркер «здесь пишешь ты» |
| `test/e2e/` | матрица конфигураций (шаг 12) |
| `test/chaos/` | тесты отказов (шаг 13) |
| `deploy/compose.yaml` | инфраструктура для ручного запуска (шаг 14) |
