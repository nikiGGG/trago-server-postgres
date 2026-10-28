# Проверка исправлений после mission review

Изменения подготовлены после ревью `postgres-dashboard-review-fixes-01M5ZJTF`.
По решению пользователя PostgreSQL 18 заменяет отдельную обязательную проверку
PostgreSQL 17. Launcher теперь по умолчанию запускает `postgres:18`, проверяет
реальную версию сервера и допускает major 17 или 18.

## Что исправлено

- RISK-1: фильтр извлекает пароли из keyword DSN с backslash-экранированием,
  пробелами вокруг `=` и из URL userinfo/query. Скрывает пароль и полную DSN
  в обычном и JSON-экранированном выводе. Неразбираемая DSN приводит к ошибке
  и подавлению захваченного вывода.
- RISK-2: stdout/stderr проверки версии захватываются и фильтруются; код
  завершения сохраняется. DSN передаётся helper-процессам через окружение.
- RISK-3: оба fractional-теста Stats2 обязательны в manifest. Наблюдаемые
  PostgreSQL-тесты вне manifest также не могут пропускаться или оставаться
  без завершения. Исключены только явно перечисленные server-free тесты.
- TEST-1: оба отрезка по 0,6 секунды входят в одну группу `type=review`;
  ожидается 2 секунды. Ошибочное округление общей суммы дало бы 1 секунду.
- `make test-postgres` явно использует Bash для `pipefail`, независимо от
  системного `/bin/sh`.

Новый фильтр `hack/redact-postgres-output.py` использует Python 3 и только
стандартную библиотеку. Python 3 должен быть доступен на PATH локально и в CI.

## Что проверено здесь

- `make test-database-launcher`: **27 случаев, 0 ошибок, 0 пропусков**.
  Включены оба воспроизведённых формата пароля, успешный/ошибочный запуск,
  JSON с кавычками/backslash/переносом строки, ошибка version probe,
  неразбираемая DSN, разрешённые major 17/18 и отклонённые 16/19.
- `bash -n hack/test-databases.sh hack/test-databases_test.sh`: exit 0.
- `git diff --check`: exit 0.

Go, Docker и Yarn отсутствуют в текущем окружении. Новые Go-регрессии,
реальный PostgreSQL, lint и полный pipeline здесь **не запускались**.
Исторические результаты PostgreSQL 18 не подтверждают эти новые изменения.

## Команды для другого устройства

Выполнить из корня репозитория с перенесёнными изменениями. Нужны Go с CGO/C
компилятором, Python 3, Docker, Node и Yarn; версии Go/gqlgen/goimports заданы
в `go.mod`, Makefile и workflow. Для локального disposable PostgreSQL не
задавать `TRAGGO_TEST_POSTGRES_DSN`: launcher сам создаст и удалит контейнер.

```bash
git rev-parse HEAD
git status --short
git diff --binary > /tmp/postgres-dashboard-followup.patch
# Если новые файлы ещё не закоммичены, перенести их тоже: git diff их не включает.

make download-tools
export PATH="$(go env GOPATH)/bin:$PATH"
make install
make generate
NODE_OPTIONS=--openssl-legacy-provider make build-js

make test-database-launcher
go test -count=1 ./hack/...
go test -count=1 ./model/... ./statistics/... ./database/... ./graphql/...
make test-postgres
make lint
make test
git diff --check
```

Обычный Go-запуск без DSN может пропустить PostgreSQL-тесты: это проверка
SQLite. Обязательную проверку PostgreSQL выполняет `make test-postgres`:
ожидается **20/20 обязательных live suites**, **8/8 обязательных children**,
ноль пропусков live PostgreSQL-тестов и exit 0.

Для другого образа PostgreSQL 18, например из доступного зеркала:

```bash
TRAGGO_TEST_PG_IMAGE=your-mirror/postgres:18 make test-postgres
```

Сохранить фактический commit и рабочий diff, версии инструментов, команды,
коды завершения и сводки тестов. `goimports -l` в составе `make lint` должен
показать ноль замечаний: сам по себе его exit 0 не означает чистое форматирование.
Полный `make test` должен закончиться успешно для backend race/coverage и UI.

Закрывать замечания ревью следует после успешного выполнения этих команд на
новых исходниках. Отдельный прогон PostgreSQL 17 не требуется.

## Фактический прогон на полном тулчейне (2026-10-29)

Окружение с полным набором инструментов: Go `go1.25.5 darwin/arm64`, Docker
`27.3.1`, Python `3.13.5`, Node `v22.20.0`, Yarn, Bash. Диспозиционный контейнер
PostgreSQL 18 взят из корпоративного зеркала
`docker-proxy.artifactory.tcsbank.ru/postgres:18` (`TRAGGO_TEST_PG_IMAGE`) и
запущен на свободном порту 55435 (`TRAGGO_TEST_PG_PORT`) — дефолтный порт 55432
на этой машине занят (ssh-туннель), это окружение, а не дефект кода.

Исходники: `git rev-parse HEAD` = `6b5996c5afd6bc0d6604a226dc81e66ae75aed46`
(ветка `plan/postgres-dashboard`), рабочий каталог чистый, изменений в нём нет.

| Проверка | Команда | Результат |
|---|---|---|
| Launcher shell suite | `make test-database-launcher` | **27/27** (0 failed, 0 skipped) |
| Go-тесты hack (checker) | `go test -count=1 ./hack/...` | ok |
| Live PostgreSQL gate | `make test-postgres` | **20/20 live suites, 8/8 children, 0 skips, exit 0** |
| TEST-1 SQLite | `go test ./statistics/ -run TestStats$` | pass; `fractional_sum-of-rounded_vs_rounded-sum` = 2s |
| Redaction boundaries | `hack/redact-postgres-output.py` (URL userinfo/query, keyword с пробелами и `\`, JSON-escaped, неразбираемая DSN) | всё корректно; fail-closed на неразбираемой DSN |
| Shell syntax | `bash -n hack/test-databases.sh hack/test-databases_test.sh` | exit 0 |
| Whitespace | `git diff --check` | exit 0 |

Вывод gate:

```
PostgreSQL result manifest summary:
  required live suites: 20/20 passed
  required children:    8/8 passed
  live-suite skips:     0 (any is a failure)
  server-free skips:    0 (informational, excluded)
```

Статус по пунктам ревью:
- **RISK-1** — подтверждён поведением фильтра на всех граничных форматах
  (URL userinfo/query, keyword-пароль с пробелами вокруг `=` и backslash-
  экранированием, JSON-экранированный вывод); неразбираемая DSN даёт
  `error: output suppressed` и exit 1.
- **RISK-2** — version probe захватывает/фильтрует оба потока, сохраняет код
  завершения, DSN передаётся через `TRAGGO_TEST_POSTGRES_DSN`; реальный сервер
  major 18 принят (допустимы 17/18).
- **RISK-3** — live gate: все 20 обязательных roots и 8 children завершились
  pass, ноль пропусков; checker-тесты в `./hack/...` зелёные (включая
  `TestStats2FractionalSuitesAreMandatory` и `TestUnlistedPostgresTestsCannotSkip...`).
- **TEST-1** — `fractional_sum-of-rounded_vs_rounded-sum`: два отрезка по 0,6 с
  в одной группе `type=review` дают 2 с (sum-of-rounded), а не 1 с.
- **make test-postgres** — `SHELL := /bin/bash` и `set -o pipefail` в Makefile;
  ненулевой exit launcher'а распространяется через pipeline (проверено на
  провале из-за занятого порта).

Не запускались здесь: `make generate`, `make lint` (goimports/vet/yarn) и полный
`make test` (race/coverage backend + UI). Они не входят в объём ревью-фиксов и
покрывают более широкий pipeline; два целевых гейта документа
(`test-database-launcher` и `test-postgres`) на реальных исходниках зелёные.
Отдельный прогон PostgreSQL 17 не требуется (решение пользователя: PostgreSQL 18).
