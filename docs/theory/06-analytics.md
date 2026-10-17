# Аналитика: ClickHouse

**Роль:** аналитическое хранилище (OLAP).
**Отвечает на вопрос:** как посчитать выручку по дням за три года по миллиардам строк за секунды и не положить при этом продакшен-базу.

## Когда брать

- Запросы — агрегаты (`sum`, `count`, `quantile`, `uniq`) по большим диапазонам, а не точечные выборки.
- Данные в основном дописываются и почти не меняются: события, логи, метрики, клики.
- Отчёты по OLTP-базе стали тяжёлыми и мешают пользовательским запросам.

## Когда не брать

- Нужны частые `UPDATE` и `DELETE` отдельных строк. Они есть, но это тяжёлые фоновые мутации, а не обычная операция.
- Нужны транзакции и строгая консистентность.
- Точечные запросы по ключу с низкой задержкой — это работа OLTP-базы или кэша.
- Объёмы маленькие: Postgres с правильными индексами посчитает отчёт по миллиону строк сам.

## ClickHouse vs альтернативы

| | ClickHouse | Postgres (+ TimescaleDB) | BigQuery / Snowflake / Redshift | DuckDB |
|---|---|---|---|---|
| Хранение | колоночное | строчное (Timescale — гибрид) | колоночное | колоночное |
| Где живёт | свой кластер или cloud | у тебя уже есть | managed cloud | встраивается в процесс |
| Сильная сторона | скорость на сырых данных, real-time вставка | одна база на всё | ноль эксплуатации, платишь за запрос | локальная аналитика, файлы parquet |
| Обновления строк | дорого | нормально | нормально | нормально |

## Что важно понимать

### Колоночное хранение

`SELECT sum(total) FROM orders` читает с диска только колонку `total`, а не все строки целиком. Колонка из одинаковых по типу значений отлично сжимается (в 5–20 раз), а векторные инструкции CPU обрабатывают её пачками. Отсюда скорость.

### MergeTree

Основной движок таблиц. Вставка создаёт неизменяемый кусок (part), отсортированный по `ORDER BY`. В фоне куски сливаются. Следствия:

- **Мелкие вставки по одной строке — антипаттерн:** тысячи кусков, «Too many parts». Вставляй батчами или включи `async_insert`.
- **`ORDER BY` — это одновременно первичный индекс (разреженный) и способ хранения.** Его выбирают под основные фильтры запросов.
- **Семейство движков решает задачи при слиянии:** `ReplacingMergeTree` схлопывает дубликаты по ключу, `SummingMergeTree` суммирует, `AggregatingMergeTree` хранит промежуточные состояния агрегатов.

### Дубликаты от at-least-once

Брокер может доставить событие дважды. `ReplacingMergeTree` с ключом, включающим `event_id`, рано или поздно уберёт дубль. Но «рано или поздно» — после слияния, поэтому запрос пишется с `FINAL` (дорого на больших таблицах) или с `GROUP BY` по ключу.

### Лог фактов, а не таблица состояний

В этом репо аналитика хранит не «текущий статус заказа», а события: размещён, оплачен, отменён. Текущее состояние — это запрос к фактам. Так не нужно обновлять строки, и история не теряется.

## 🔎 Изучи сам

- OLTP vs OLAP, row store vs column store
- star schema, snowflake schema, факты и измерения
- MergeTree: parts, merges, sparse primary index, granules, `index_granularity`
- `ReplacingMergeTree`, `SummingMergeTree`, `AggregatingMergeTree`, `CollapsingMergeTree`
- materialized views в ClickHouse (это триггер на вставку, а не то, что в Postgres)
- `FINAL` и почему он дорогой
- `async_insert`, Buffer-таблицы, батчинг вставок
- `LowCardinality`, кодеки сжатия
- Kafka table engine — ClickHouse сам читает из Kafka
- ETL vs ELT, CDC (Debezium)
- DuckDB, Parquet, Apache Arrow

Первоисточники:

- [ClickHouse docs: MergeTree](https://clickhouse.com/docs/engines/table-engines/mergetree-family/mergetree)
- [ClickHouse docs: Best practices for inserts](https://clickhouse.com/docs/optimize/bulk-inserts)
- [ClickHouse: Why is ClickHouse so fast?](https://clickhouse.com/docs/concepts/why-clickhouse-is-so-fast)
- Kleppmann, DDIA, глава 3: «Column-Oriented Storage»
