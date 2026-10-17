# 01 — Ядро: домен, порты, use case, memory, контракт-тест 🟢

## Цель

Понять, как устроен сервис, до того как трогать инфраструктуру: где проходят границы слоёв, что такое порт, что такое контракт-тест и почему memory-реализация — не «игрушка», а эталон поведения.

## Теория

- [00-map](../docs/theory/00-map.md) — целиком
- [09-patterns](../docs/theory/09-patterns.md): outbox, optimistic locking
- [10-testing](../docs/theory/10-testing.md): контракт-тесты
- [docs/adr/README.md](../docs/adr/README.md) — что такое ADR

## Что дано (всё)

- `internal/app/domain` — `Order`, статусы и переходы, события
- `internal/app/port/port.go` — все порты
- `internal/app/usecase` — `PlaceOrder`, `PayOrder`, `CancelOrder`, `GetOrder` и юнит-тесты на все ветки
- `internal/infrastructure/memory` — store (репозиторий + outbox + транзакции) и broker
- `internal/infrastructure/contracttest` — контракты на репозиторий, транзакции+outbox, брокер
- `internal/presentation/http`, `cmd/orders/main.go` — HTTP поверх memory

## 🤔 Подумай до кода

1. Выполни `go list -deps -f '{{if not .Standard}}{{.ImportPath}}{{end}}' ./internal/app/...`. Каких пакетов там нет и почему это важно?
2. `PlaceOrder` не публикует событие в брокер. Почему? Что было бы, если бы он сохранял заказ, а потом вызывал `Publish`?
3. Найди `o.Version++` в `usecase.transition`. Зачем он нужен? Какое знание о репозитории здесь дублируется? Куда его можно перенести (подсказки: `Update` возвращает `Order`; `Update` принимает `*Order`; версию двигает сам домен) и что изменится в контракте?
4. Транзакция передаётся через `context.Context`. Какие плюсы у этого подхода и какие минусы? Чем он лучше и хуже явного параметра `tx`?
5. Как memory store добивается атомарности `TxManager.Do`? Какой у него уровень изоляции на самом деле?
6. Прочитай `contracttest/repository.go`. Какой тест упадёт, если репозиторий будет делать `UPDATE … WHERE id = ?` без проверки версии?
7. Почему деньги в `int64` копейках, а не во `float64`?

## Что сделать

1. Склонируй репо, запусти `make hooks` (ставит pre-push), запусти `make check` — всё должно быть зелёным.
2. Запусти сервис `make run` и пройди руками: `POST /orders`, `GET /orders/{id}`, `POST /orders/{id}/pay`, повторный `pay` (должен быть `409`).
3. **Сломай** memory store так, чтобы упал ровно один контракт-тест (например, убери проверку версии). Убедись, что падает именно ожидаемый тест. Верни как было.
4. Добавь в `domain` и `usecase` одну мелочь по выбору: правило «нельзя отменить заказ старше N дней» или «максимум 50 позиций в заказе». С тестами.

## ✅ Definition of Done

```sh
make check
```

## ADR

Прочитай пример [0001-ports-in-app-layer.md](../docs/adr/0001-ports-in-app-layer.md). Напиши `0002-...` про **одно** из:

- передача транзакции через `context.Context` vs явный параметр;
- где должно жить знание о версии (вопрос 3).

## 🎤 Вопросы на защите

- Нарисуй на доске пакеты и стрелки импортов. Где порты? Почему там?
- Что такое контракт-тест и чем он отличается от mock-тестов use case'а?
- Почему memory store — fake, а не mock?
- Что проверяет тест «concurrent updates of one version: exactly one wins» и почему его нельзя убрать?
