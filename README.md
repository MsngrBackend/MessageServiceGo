# MessageServiceGo

Go-версия сервиса сообщений для мессенджера. Сервис предоставляет REST API для чатов, участников и сообщений, а также WebSocket для realtime-событий.

Фактический порт сервиса:

```text
http://localhost:8080
```

Swagger/OpenAPI в текущей версии не подключен.

## Стек

- Go 1.25
- PostgreSQL 16
- NATS 2.10
- `net/http`
- `golang.org/x/net/websocket`
- `pgx/v5`

## Переменные окружения

Сервис читает:

| Переменная | Значение по умолчанию | Назначение |
|---|---|---|
| `DATABASE_URL` | `postgres://postgres:ultramegasecret@localhost:5432/messaging_db` | Подключение к PostgreSQL |
| `NATS_URL` | `nats://localhost:4222` | Подключение к NATS |
| `ENCRYPTION_SERVICE_ADDR` | `:8081` | HTTP-адрес EncryptionService |

Пример `.env` для запуска через Docker Compose:

```env
POSTGRES_USER=postgres
POSTGRES_PASSWORD=ultramegasecret
POSTGRES_DB=messaging_db
DATABASE_URL=postgres://postgres:ultramegasecret@postgres:5432/messaging_db
NATS_URL=nats://nats:4222
ENCRYPTION_SERVICE_ADDR=:8081
```

## Запуск через Docker Compose

В `docker-compose.yaml` используется Docker-сеть `msngr-network`:

```yaml
networks:
  msngr-network:
```

Запуск:

```powershell
docker compose up --build -d
docker compose ps
```

Ожидаемые контейнеры:

- `message-service`
- `encryption-service`
- `messaging_postgres`
- `msngr_nats`

Логи сервиса:

```powershell
docker logs message-service
```

При успешном запуске:

```text
server listening on :8080
```

Остановка:

```powershell
docker compose down
```

Остановка с удалением данных PostgreSQL:

```powershell
docker compose down -v
```

## База данных

В репозитории нет миграций, поэтому таблицы нужно создать вручную.

Подключение к PostgreSQL в контейнере:

```powershell
docker exec -it messaging_postgres psql -U postgres -d messaging_db
```

SQL-схема:

```sql
CREATE TABLE chats (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL
);

CREATE TABLE chat_members (
  id BIGSERIAL PRIMARY KEY,
  chat_id BIGINT NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
  user_id TEXT NOT NULL,
  UNIQUE(chat_id, user_id)
);

CREATE TABLE messages (
  id BIGSERIAL PRIMARY KEY,
  chat_id BIGINT NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
  content TEXT NOT NULL,
  sender_id TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Проверка таблиц:

```sql
\dt
\d chats
\d chat_members
\d messages
```

## REST API

Авторизация вынесена в отдельный AuthService/API Gateway. Сам `MessageServiceGo` не валидирует JWT-токены: ожидается, что внешний gateway уже проверил пользователя и передал его идентификатор в сервис.

Для большинства REST-запросов используется заголовок:

```text
X-User-Id: <user_id>
```

### Чаты

#### Создать чат

```http
POST /chats/
X-User-Id: user1
Content-Type: application/json
```

```json
{"name":"test chat"}
```

Ответ:

```json
{"id":1,"name":"test chat"}
```

Создатель автоматически добавляется в `chat_members`.

#### Получить чаты пользователя

```http
GET /chats/
X-User-Id: user1
```

Ответ:

```json
[
  {"id":1,"name":"test chat"}
]
```

#### Получить чат по ID

```http
GET /chats/{id}/
X-User-Id: user1
```

Пользователь должен быть участником чата.

#### Удалить чат

```http
DELETE /chats/{id}/
X-User-Id: user1
```

Пользователь должен быть участником чата. При успехе возвращается `204 No Content`.

### Участники чата

#### Получить список участников

```http
GET /chats/{id}/members/
```

Ответ:

```json
[
  {"id":1,"chat_id":1,"user_id":"user1"}
]
```

#### Добавить участника

```http
POST /chats/{id}/members/
Content-Type: application/json
```

```json
{"user_id":"user2"}
```

#### Удалить участника

```http
DELETE /chats/{id}/members/{user_id}/
X-User-Id: user1
```

`X-User-Id` должен принадлежать пользователю, который состоит в этом чате.

### Сообщения

#### Создать сообщение

```http
POST /messages/
Content-Type: application/json
```

```json
{"chat_id":1,"content":"hello","sender_id":"user1"}
```

Ответ:

```json
{
  "id": 1,
  "chat_id": 1,
  "content": "hello",
  "sender_id": "user1",
  "created_at": "2026-05-16T18:18:56.367754Z"
}
```

#### Получить историю сообщений

```http
GET /messages/chats/{chat_id}/messages/?limit=100&offset=0
```

Сообщения возвращаются от новых к старым.

#### Обновить сообщение

```http
PATCH /messages/{id}/
Content-Type: application/json
```

```json
{"content":"updated hello"}
```

#### Удалить сообщение

```http
DELETE /messages/{id}/
```

При успехе возвращается:

```json
{"detail":"message deleted"}
```

## Примеры запросов в PowerShell

Создать чат:

```powershell
$body = '{"name":"test chat"}'
Invoke-RestMethod -Uri "http://localhost:8080/chats/" -Method Post -ContentType "application/json" -Headers @{ "X-User-Id" = "user1" } -Body $body
```

Получить чаты пользователя:

```powershell
Invoke-RestMethod -Uri "http://localhost:8080/chats/" -Method Get -Headers @{ "X-User-Id" = "user1" }
```

Создать сообщение:

```powershell
$body = '{"chat_id":1,"content":"hello","sender_id":"user1"}'
Invoke-RestMethod -Uri "http://localhost:8080/messages/" -Method Post -ContentType "application/json" -Body $body
```

Получить историю:

```powershell
Invoke-RestMethod -Uri "http://localhost:8080/messages/chats/1/messages/?limit=10&offset=0" -Method Get
```

Посмотреть HTTP-код ошибки:

```powershell
try {
  $body = '{"chat_id":9999,"content":"ghost","sender_id":"user1"}'
  Invoke-RestMethod -Uri "http://localhost:8080/messages/" -Method Post -ContentType "application/json" -Body $body
} catch {
  [pscustomobject]@{
    StatusCode = $_.Exception.Response.StatusCode.value__
    Body = $_.ErrorDetails.Message
  }
}
```

## WebSocket

Endpoint:

```text
ws://localhost:8080/ws/chat?chat_id=1&user_id=user1
```

Для Postman нужно указать header:

```text
Origin: http://localhost:8080
```

Без корректного `Origin` сервер может вернуть `403`.

### Отправить сообщение

```json
{"type":"message","message":"hello from websocket"}
```

Сообщение рассылается активным WebSocket-клиентам этого же `chat_id`.

### Индикатор печати

```json
{"type":"typing"}
```

### Входящие события

Сообщение:

```json
{"text":"hello from websocket","sender_id":"user1"}
```

Typing:

```json
{"sender_id":"user1"}
```

## NATS

Если `NATS_URL` задан и NATS доступен, REST endpoint `POST /messages/` публикует событие:

```text
msngr.chat.<chat_id>.event
```

Payload:

```json
{
  "type": "message",
  "text": "hello",
  "sender_id": "user1"
}
```

Также в коде есть метод публикации статуса пользователя:

```text
msngr.user.status
```

Payload:

```json
{
  "user_id": "user1",
  "online": true
}
```

## EncryptionService

EncryptionService запускается отдельным HTTP-сервисом:

```text
http://localhost:8081
```

Код сервиса вынесен в соседний проект:

```text
E:\My Programs\Msngr\EncryptionService
```

Сервис сделан легким: он хранит публичные X25519-ключи пользователей и формирует зашифрованные конверты сообщений в формате `nacl-box-x25519-xsalsa20-poly1305-sealed-v1`.

Важное ограничение E2E: приватные ключи не должны попадать в EncryptionService или MessageService. Для строгого end-to-end шифрования клиент генерирует пару ключей локально, регистрирует только публичный ключ, шифрует сообщение для участников чата и отправляет в `MessageServiceGo` уже зашифрованный `content`.

### Healthcheck

```http
GET /health
```

### Зарегистрировать публичный ключ

`public_key` — base64 от 32 байт X25519 public key.

```http
POST /keys/
Content-Type: application/json
```

```json
{"user_id":"user1","public_key":"base64-encoded-32-byte-key"}
```

### Получить публичный ключ

```http
GET /keys/{user_id}/
```

### Получить ключи нескольких пользователей

```http
POST /keys/lookup/
Content-Type: application/json
```

```json
{"user_ids":["user1","user2"]}
```

### Зашифровать сообщение для получателей

Endpoint может использовать публичные ключи из хранилища или ключи, переданные прямо в `recipients`.

```http
POST /messages/encrypt/
Content-Type: application/json
```

```json
{
  "content": "hello",
  "recipients": [
    {"user_id": "user1"},
    {"user_id": "user2", "public_key": "base64-encoded-32-byte-key"}
  ]
}
```

Ответ:

```json
{
  "version": "nacl-box-x25519-xsalsa20-poly1305-sealed-v1",
  "envelopes": [
    {
      "user_id": "user1",
      "algorithm": "nacl-box-x25519-xsalsa20-poly1305-sealed-v1",
      "ciphertext": "base64-ciphertext"
    }
  ]
}
```

Для хранения в `messages.content` можно передавать JSON с `version` и `envelopes` как строку. MessageService при этом не расшифровывает payload и остается простым транспортом/хранилищем сообщений.
