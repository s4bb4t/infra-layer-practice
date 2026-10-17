# Пример заполненного PR

Так может выглядеть описание PR шага 03. Заголовок PR: `step-03: MySQL adapter`.

---

## Шаг

03 — MySQL. Задача: [tasks/03-mysql.md](../tasks/03-mysql.md)

## Что сделано

- `mysql.TxManager.Do`: транзакция через `database/sql`, вложенные вызовы присоединяются к внешней
- `mysql.Orders.Update`: optimistic locking через `WHERE version = ?`, различаю `ErrNotFound` и `ErrConflict` вторым запросом
- контракт `RunOrderRepository` и `RunTxOutbox` зелёные против MySQL 8.4

## ADR

[0004-mysql-isolation-level.md](adr/0004-mysql-isolation-level.md) — оставляем REPEATABLE READ по умолчанию: optimistic locking не зависит от уровня изоляции, а UPDATE в InnoDB всегда читает последнюю закоммиченную версию строки.

## Definition of Done

- [x] `make check` зелёный локально (fmt, vet, golangci-lint, unit, integration)
- [x] в тестах пакетов этого шага нет SKIP
- [x] пункты DoD из задачи выполнены
- [x] ADR написан и отвечает на все «🤔 Подумай до кода»
- [x] в diff нет ничего лишнего

## Что было сложно / что осталось непонятным

Сначала проверял `RowsAffected() == 0` как конфликт и получил ложные конфликты в тесте, где апдейт ничего не менял. Разобрался с `clientFoundRows`. Не до конца понял, когда в InnoDB случаются gap locks в этом коде и могут ли они дать deadlock при конкурентных `Create`.

---

## Как работать с ветками и PR

```sh
git switch main && git pull
git switch -c step-03
# ... работа ...
make check                      # обязательно, хук pre-push тоже это проверит
git push -u origin step-03
gh pr create --fill             # или через веб-интерфейс GitHub
```

После ревью — правки в ту же ветку. После апрува — merge в `main` (squash), ветка удаляется, следующий шаг начинается от свежего `main`.
