# Огромный поток записи: Cassandra / ScyllaDB

**Роль:** распределённое хранилище для очень высокой нагрузки на запись с линейным горизонтальным масштабированием.
**Отвечает на вопрос:** как писать сотни тысяч строк в секунду, переживать падение узлов и дата-центров и расти простым добавлением машин.

В этом курсе — **только теория** (задача 15). Не потому что неважно, а потому что ценность Cassandra проявляется на кластере из многих узлов под реальной нагрузкой. Один контейнер в тесте не покажет ничего, кроме синтаксиса CQL.

## Когда брать

- Запись огромная и непрерывная: события, телеметрия, сообщения чата, лента активности.
- Запросы известны заранее и укладываются в «дай по ключу партиции, отсортировано по ключу кластеризации».
- Нужна доступность при падении узлов и целых дата-центров (multi-DC репликация — сильная сторона).
- Готовы к отдельной команде или managed-сервису (Astra, ScyllaDB Cloud, AWS Keyspaces).

## Когда не брать

- Запросы заранее неизвестны или ad-hoc. Нет JOIN, нет произвольных `WHERE`; новый вид запроса — это новая таблица.
- Нужны транзакции между строками. Есть lightweight transactions (Paxos), но они медленные и для частного случая.
- Объём укладывается в один Postgres. Это почти всегда так дольше, чем кажется.

## Cassandra vs ScyllaDB vs альтернативы

| | Cassandra | ScyllaDB | DynamoDB | Postgres + шардирование (Citus) |
|---|---|---|---|---|
| Реализация | Java | C++, shard-per-core (Seastar) | managed AWS | расширение Postgres |
| Протокол | CQL | CQL (совместим) + DynamoDB API | свой | SQL |
| Модель | wide-column, ключ партиции + clustering | то же | key-value / документы | реляционная |
| Сильная сторона | зрелость, экосистема | задержки и пропускная способность на узел | ноль эксплуатации | SQL и транзакции сохраняются |

## Что важно понимать

- **Моделируют от запросов, а не от сущностей.** Одна и та же информация часто денормализуется в несколько таблиц — по одной на каждый запрос.
- **Ключ партиции определяет всё:** на каком узле лежат данные, что можно спросить эффективно, где появятся горячие точки. Партиция не должна неограниченно расти.
- **Tunable consistency:** на каждый запрос задаётся, сколько реплик должно ответить (`ONE`, `QUORUM`, `ALL`). Если `R + W > N`, чтение увидит запись.
- **LSM-tree:** запись — это дописывание в commit log и memtable, поэтому она быстрая. Удаление — это tombstone, и много tombstones убивают чтение.

## Вопросы для ADR (задача 15)

Задача 15 — не код, а решение. Тебе нужно ответить, взял бы ты Cassandra/Scylla для части этой системы, и обосновать ответ. Подсказки, о чём подумать:

- Какая часть данных заказа вообще могла бы туда поехать? История статусов? Лента событий клиента?
- Какие запросы к ней будут? Можно ли их выразить через ключ партиции и clustering?
- Какой объём записи оправдал бы появление новой технологии в команде?
- Что дешевле: Cassandra или партиционирование таблицы в Postgres плюс read replicas?

## 🔎 Изучи сам

- partition key, clustering key, primary key в CQL
- consistent hashing, token ring, vnodes
- replication factor, `NetworkTopologyStrategy`
- consistency levels; `QUORUM`, `LOCAL_QUORUM`; `R + W > N`
- hinted handoff, read repair, anti-entropy repair
- LSM-tree, memtable, SSTable, compaction strategies (STCS, LCS, TWCS)
- tombstones, `gc_grace_seconds`
- lightweight transactions (LWT)
- Amazon Dynamo paper, Google Bigtable paper
- ScyllaDB: shard-per-core architecture, Seastar

Первоисточники:

- [Cassandra docs: Data modeling](https://cassandra.apache.org/doc/latest/cassandra/developing/data-modeling/index.html)
- [Dynamo: Amazon's Highly Available Key-value Store](https://www.allthingsdistributed.com/files/amazon-dynamo-sosp2007.pdf)
- [Bigtable: A Distributed Storage System for Structured Data](https://research.google/pubs/bigtable-a-distributed-storage-system-for-structured-data/)
- [ScyllaDB University](https://university.scylladb.com/) — бесплатные курсы по моделированию
- Discord, [How Discord Stores Trillions of Messages](https://discord.com/blog/how-discord-stores-trillions-of-messages) — миграция с Cassandra на Scylla
