# Реляционные базы: Postgres, MySQL

**Роль:** источник истины. Сюда пишется то, что нельзя потерять.
**Отвечает на вопрос:** где надёжно хранить состояние и менять его атомарно.

## Когда брать

- Есть связи между сущностями и инварианты, которые надо держать (уникальность, внешние ключи, `CHECK`).
- Нужны транзакции: «списать и зачислить» должно произойти целиком или никак.
- Запросы заранее неизвестны: SQL позволяет задать новый вопрос к старым данным без переделки схемы.
- Объём — от нуля до терабайт на одном узле. Это покрывает подавляющее большинство продуктов.

## Когда не брать

- Нужна запись сотен тысяч событий в секунду с линейным горизонтальным ростом → wide-column ([07](07-wide-column.md)).
- Основной запрос — агрегаты по миллиардам строк → колоночная база ([06](06-analytics.md)).
- Основной запрос — релевантный полнотекстовый поиск с опечатками → поисковый движок ([05](05-search.md)).
- Схема действительно не определена и у каждого документа своя форма → документная база ([02](02-documents.md)). Но сначала проверь, не хватит ли `JSONB`.

## Postgres vs MySQL

Оба — зрелые OLTP-базы. Для большинства задач подходят оба. Различия, которые встречаются на практике:

| | Postgres | MySQL (InnoDB) |
|---|---|---|
| Уровень изоляции по умолчанию | READ COMMITTED | REPEATABLE READ |
| MVCC | старые версии строк в самой таблице → нужен VACUUM | undo log → нет VACUUM, но есть purge и длинные транзакции раздувают undo |
| Типы | богатые: массивы, `JSONB` с индексами, диапазоны, `uuid`, enum, свои типы | беднее; `JSON` есть, индексы по нему — через generated columns |
| DDL в транзакции | да | нет (DDL делает неявный commit) |
| Расширения | PostGIS, pg_trgm, TimescaleDB, pgvector… | плагины есть, но экосистема уже |
| `RETURNING` | да | нет (только в MariaDB) |
| Репликация | физическая (WAL) и логическая | binlog (statement/row) — исторически проще в настройке |
| Массовость | чаще выбирают для новых проектов | огромная установленная база, особенно веб |

В этом репо оба адаптера реализуют один и тот же порт — сравни `internal/infrastructure/postgres` и `internal/infrastructure/mysql` построчно. Разница будет в мелочах диалекта: `$1` против `?`, `ANY($1)` против раскрытого `IN (?, ?, …)`, коды ошибок уникальности (`23505` против `1062`), `RowsAffected` (найденные строки против изменённых).

## Что важно понимать

### Транзакции и изоляция

- Четыре уровня из стандарта и какие аномалии они допускают: dirty read, non-repeatable read, phantom read, lost update, write skew.
- В Postgres REPEATABLE READ на самом деле snapshot isolation, а SERIALIZABLE — SSI (serializable snapshot isolation): транзакция может упасть с ошибкой сериализации, и это нормально — её надо повторить.
- В этом репо конкурентные апдейты защищены **optimistic locking** (колонка `version`, `UPDATE … WHERE version = $n`), а не уровнем изоляции. Подумай, почему этого хватает на READ COMMITTED.

### Индексы

- B-tree по умолчанию. Составной индекс работает по левому префиксу: `(customer_id, created_at)` помогает `WHERE customer_id = ? ORDER BY created_at`, но не `WHERE created_at > ?`.
- Partial index (`WHERE published_at IS NULL`) — индекс только по нужной части таблицы. В outbox неопубликованных строк единицы, опубликованных — миллионы.
- `EXPLAIN ANALYZE` — единственный честный ответ на вопрос «использует ли запрос индекс».

### Пулы соединений

Соединение с Postgres — это процесс на сервере. Тысяча горутин — не тысяча соединений: между ними пул (`pgxpool`, `database/sql`). Размер пула ограничивает параллелизм в базу, и это хорошо. В проде перед Postgres часто стоит PgBouncer.

### Миграции

Схема — это код. Она версионируется (goose, migrate, atlas), едет вместе с бинарником (`embed`) и применяется на старте или отдельным шагом деплоя. Подумай, что опаснее: добавить колонку или добавить `NOT NULL` колонку в таблицу на 100 млн строк.

## 🔎 Изучи сам

- ORM — что это вообще; Active Record vs Data Mapper; почему в Go их любят меньше, чем в других языках
- проблема N+1
- query builders и кодогенерация: `sqlc`, `squirrel`, `goqu`, `sqlx` — чем отличаются от ORM (`gorm`, `ent`, `bun`)
- MVCC, VACUUM, bloat, `autovacuum`
- WAL, `fsync`, `synchronous_commit`
- `SELECT … FOR UPDATE`, `SKIP LOCKED`, `NOWAIT`, advisory locks
- deadlock и как его диагностировать
- connection pool: `max_connections`, PgBouncer, transaction vs session pooling
- prepared statements и почему они ломаются с PgBouncer в transaction-режиме
- online-миграции: `CREATE INDEX CONCURRENTLY`, expand/contract, gh-ost, pt-online-schema-change
- репликация: streaming, logical, binlog; read replicas и replication lag

Первоисточники:

- [PostgreSQL docs: Concurrency Control](https://www.postgresql.org/docs/current/mvcc.html) — главы 13.1–13.4
- [PostgreSQL docs: Indexes](https://www.postgresql.org/docs/current/indexes.html)
- [MySQL docs: InnoDB Locking and Transaction Model](https://dev.mysql.com/doc/refman/8.4/en/innodb-locking-transaction-model.html)
- [pgx README](https://github.com/jackc/pgx) — почему pgx, а не `lib/pq`
- [Use The Index, Luke](https://use-the-index-luke.com/) — про индексы лучше, чем где-либо ещё
