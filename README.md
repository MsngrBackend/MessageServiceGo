# MessageService

Микросервис для обмена сообщениями в реальном времени. Предоставляет REST API для управления чатами и сообщениями, а также WebSocket для получения сообщений онлайн.

## REST API

Полная документация не доступна в Swagger: **`http://localhost:8000/docs`**

Для запросов, требующих аутентификации, передавайте заголовок:
```
X-User-ID: <user_id>
```

## WebSocket

Swagger не покрывает WebSocket. Подключение:

```
ws://localhost:8000/{chat_id}/{user_id}?username={username}
```

| Параметр   | Тип    | Описание              |
|------------|--------|-----------------------|
| `chat_id`  | int    | ID чата               |
| `user_id`  | string | ID пользователя       |
| `username` | string | Query-параметр, имя   |

> Пользователь должен быть участником чата, иначе соединение будет отклонено

### Отправка сообщения

```json
{ "type": "message", "text": "Привет!" }
```

### Индикатор печати

```json
{ "type": "typing" }
```

### Входящие события

```json
// Новое сообщение
{ "text": "Привет!", "sender_id": "user123" }

// Кто-то печатает
{ "sender_id": "user123" }
```

## NATS (несколько инстансов MessageService)

Переменная окружения **`NATS_URL`** включает доставку событий чата между репликами сервиса. Без неё поведение как раньше: рассылка только внутри одного процесса.

События:

- **`msngr.chat.<chat_id>.event`** — JSON с полями `chat_id`, `type` (`message` | `typing` | `system`), `sender_id`, для сообщений ещё `text`.
- **`msngr.user.status`** — JSON `user_id`, `status` (`online` | `offline`) при входе/выходе из WebSocket чата.

Сервис **Profile** при заданном `NATS_URL` публикует **`msngr.profile.updated`** (`user_id`, `kind`: `profile` | `avatar`) после успешного PATCH профиля или изменения аватара.

