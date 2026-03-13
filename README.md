# Лабораторная работа №3: Авторизация и аутентификация (JWT, OAuth2, Cookies)

## Описание проекта
Расширение RESTful API из Лабораторной работы №2 с реализацией системы управления доступом. Реализованы:
- Полный CRUD для пользователей (регистрация, вход, выход, сброс пароля)
- JWT-токены: Access (15 мин) + Refresh (7 дней) с подписью HS256
- Безопасные куки: передача токенов через `HttpOnly`, `SameSite` cookies
- Хеширование паролей: `bcrypt` с уникальной солью для каждого пользователя
- Refresh токены в БД: хранение хешей токенов для отзыва сессий (logout)
- Middleware: защита эндпоинтов, извлечение и валидация токенов
- Сброс пароля: генерация одноразовых токенов с временем жизни
- Авторизация через Яндекс: реализация потока Authorization Code Grant вручную
- CSRF-защита: проверка параметра `state` при обработке OAuth callback
- Защита ресурсов: все эндпоинты `/categories` и `/products` требуют авторизации
- Миграции базы данных через `golang-migrate`
- Локальный desktop GUI-клиент в папке `app/` для работы с API как с компактным Postman-клиентом

## Инструкция по запуску

### 1. Клонирование репозитория
```bash
git clone https://github.com/BugLivesMatter/lab_2.git   
cd lab_2
```

### 2. Проверка переменных окружения
Файл `.env` уже присутствует в корне проекта. Убедитесь, что он содержит следующие параметры:

```env
# === Database ===
DB_HOST=postgres
DB_PORT=5432
DB_USER=student
DB_PASSWORD=student
DB_NAME=wp_labs

# === JWT Secrets (мин. 32 символа) ===
JWT_ACCESS_SECRET=your_super_secret_access_key_change_in_prod
JWT_REFRESH_SECRET=your_super_secret_refresh_key_change_in_prod
JWT_ACCESS_EXPIRATION=15m
JWT_REFRESH_EXPIRATION=168h

# === OAuth2 Yandex ===
YANDEX_CLIENT_ID=your_yandex_client_id
YANDEX_CLIENT_SECRET=your_yandex_client_secret
YANDEX_CALLBACK_URL=http://localhost:4200/auth/oauth/yandex/callback
```

При необходимости отредактируйте значения под вашу среду.

> ⚠️ **Важно:** Никогда не коммитьте файл `.env` с реальными секретами в репозиторий! Используйте `.env.example` как шаблон.

### 3. Запуск приложения
```bash
docker-compose up --build
```

Приложение будет доступно по адресу: `http://localhost:4200`

### 4. Остановка приложения
```bash
docker-compose down
```

### 5. Полная очистка (удаление БД)
```bash
docker-compose down -v
```

## Локальный desktop-клиент `app/`

В папке `app/` находится отдельное локальное Go-приложение с графическим окном (Windows), без Docker и без отдельного backend-порта для клиента.  
Оно подключается к уже запущенному API по адресу `http://localhost:4200` и даёт:

- готовые пресеты для `GET / POST / PUT / PATCH / DELETE`;
- просмотр JSON-ответов;
- табличный вид результатов;
- отдельные вкладки для live-данных и структуры сущностей (schema).

Примечание: клиент использует нативные Windows-виджеты через `walk`.

Запуск:

```bash
go run ./app
```

Если API слушает другой адрес, можно задать его так:

```bash
set API_BASE_URL=http://localhost:4200
go run ./app
```

## Описание API

### Авторизация и аутентификация

| Метод | Эндпоинт | Описание | Доступ | Статус успеха |
|-------|----------|----------|--------|---------------|
| `POST` | `/auth/register` | Регистрация нового пользователя | Public | `201 Created` |
| `POST` | `/auth/login` | Вход (установка cookies) | Public | `200 OK` |
| `POST` | `/auth/refresh` | Обновление пары токенов | Public (требуется valid Refresh Cookie) | `200 OK` |
| `GET` | `/auth/whoami` | Проверка статуса и данные пользователя | Private | `200 OK` |
| `POST` | `/auth/logout` | Завершение текущей сессии | Private | `200 OK` |
| `POST` | `/auth/logout-all` | Завершение всех сессий пользователя | Private | `200 OK` |
| `GET` | `/auth/oauth/:provider` | Инициация входа через OAuth | Public | `302 Redirect` |
| `GET` | `/auth/oauth/:provider/callback` | Обработка ответа от OAuth провайдера | Public | `200 OK` |
| `POST` | `/auth/forgot-password` | Запрос на сброс пароля | Public | `200 OK` |
| `POST` | `/auth/reset-password` | Установка нового пароля | Public | `200 OK` |

### Категории

| Метод | Эндпоинт | Описание | Доступ | Статус успеха |
|-------|----------|----------|--------|---------------|
| `GET` | `/categories` | Список категорий с пагинацией | Private | `200 OK` |
| `GET` | `/categories/:id` | Получить категорию по ID | Private | `200 OK` |
| `POST` | `/categories` | Создать категорию | Private | `201 Created` |
| `PUT` | `/categories/:id` | Полное обновление категории | Private | `200 OK` |
| `PATCH` | `/categories/:id` | Частичное обновление категории | Private | `200 OK` |
| `DELETE` | `/categories/:id` | Мягкое удаление категории | Private | `204 No Content` |

### Продукты

| Метод | Эндпоинт | Описание | Доступ | Статус успеха |
|-------|----------|----------|--------|---------------|
| `GET` | `/products` | Список продуктов с пагинацией | Private | `200 OK` |
| `GET` | `/products/:id` | Получить продукт по ID | Private | `200 OK` |
| `POST` | `/products` | Создать продукт | Private | `201 Created` |
| `PUT` | `/products/:id` | Полное обновление продукта | Private | `200 OK` |
| `PATCH` | `/products/:id` | Частичное обновление продукта | Private | `200 OK` |
| `DELETE` | `/products/:id` | Мягкое удаление продукта | Private | `204 No Content` |

### Параметры пагинации
Доступны для эндпоинтов списков (`/categories`, `/products`):

| Параметр | Тип | По умолчанию | Описание |
|----------|-----|--------------|----------|
| `page` | integer | `1` | Номер страницы (начинается с 1) |
| `limit` | integer | `10` | Записей на странице (макс. 100) |

### Формат ответа с пагинацией
```json
{
  "data": [ ... ],
  "meta": {
    "total": 100,
    "page": 1,
    "limit": 10,
    "totalPages": 10
  }
}
```

## Тестирование API

### 1. Регистрация пользователя
**Запрос:**
```bash
curl -X POST http://localhost:4200/auth/register \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"test@example.com\",\"password\":\"SecurePass123!\",\"phone\":\"+79991234567\"}"
```

**Ожидаемый ответ (201 Created):**
```json
{
  "message": "пользователь успешно зарегистрирован",
  "userId": "uuid-..."
}
```

---

### 2. Вход (установка куки)
**Запрос:**
```bash
curl -X POST http://localhost:4200/auth/login \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"test@example.com\",\"password\":\"SecurePass123!\"}" \
  -c cookies.txt
```

**Ожидаемый ответ (200 OK):**
```json
{
  "message": "успешный вход",
  "accessExpiresIn": "15m0s",
  "refreshExpiresIn": "168h0m0s"
}
```

---

### 3. Проверка авторизации (WhoAmI)
**Запрос:**
```bash
curl http://localhost:4200/auth/whoami -b cookies.txt
```

**Ожидаемый ответ (200 OK):**
```json
{
  "id": "uuid-...",
  "email": "test@example.com",
  "phone": "+79991234567",
  "createdAt": "2026-03-13T..."
}
```

---

### 4. OAuth через Яндекс (инициация)
**Запрос в браузере:**
```
GET http://localhost:4200/auth/oauth/yandex
```

**Ожидаемо:** Редирект на `https://oauth.yandex.ru/authorize?...`

---

### 5. OAuth callback (после авторизации)
**Ожидаемый ответ (200 OK):**
```json
{
  "email": "milana@test.ru",
  "message": "успешный вход через OAuth",
  "userId": "uuid-..."
}
```

## Миграции
Миграции применяются **автоматически** при запуске приложения через `docker-compose up --build`.

Файлы миграций находятся в папке `internal/migrations/`:
- `000001_create_categories_table.up.sql` / `.down.sql`
- `000002_create_products_table.up.sql` / `.down.sql`
- `000003_create_users_table.up.sql` / `.down.sql`
- `000004_create_refresh_tokens_table.up.sql` / `.down.sql`
- `000005_create_password_reset_tokens_table.up.sql` / `.down.sql`

> Отдельная команда для запуска миграций не требуется — они выполняются в функции `runMigrations()` при старте сервера.

### Критерии приемки
1.  Репозиторий: Код загружен на GitHub/GitLab.
2.  Документация: Файл `README.md` содержит:
    -   Краткое описание проекта.
    -   Инструкция по запуску через `docker-compose up --build`.
    -   Пример файла переменных окружения (`.env.example`).
    -   Описание API (список эндпоинтов и параметров пагинации).
    -   Инструкция по запуску миграций (если требуется отдельная команда).
3.  Безопасность:
    -   Пароли, Access и Refresh токены в БД захешированы с использованием соли.
    -   Токены передаются только через `HttpOnly` cookies.
    -   Не использованы готовые библиотеки аутентификации.
    -   Реализована проверка параметра `state` в OAuth.
4.  Функциональность:
    -   Все указанные эндпоинты работают корректно.
    -   Механизм Refresh Token реализован.
    -   Logout и Logout-all инвалидируют токены в БД.
    -   OAuth вход реализован вручную и работает.
    -   Ресурсы из Лабораторной работы №2 защищены.
5.  Код:
    -   Соблюдена модульная структура.
    -   Использованы DTO для данных.
    -   Присутствует валидация входящих данных.
6.  Инфраструктура: Приложение запускается через `docker-compose up --build`.

### Контрольные вопросы
1.  В чем фундаментальная разница между аутентификацией и авторизацией?
2.  Что такое соль (salt) при хешировании пароля? Зачем она нужна и почему она должна быть уникальной для каждого пользователя?
3.  Из каких частей состоит JWT? Как сервер проверяет подлинность токена?
4.  Почему Refresh Token рекомендуется хранить в базе данных, а Access Token можно проверять Stateless?
5.  В чем преимущества использования `HttpOnly` cookies для хранения токенов по сравнению с `LocalStorage`?
6.  Зачем необходим эндпоинт `/whoami` при использовании `HttpOnly` cookies?
7.  Зачем нужен параметр `state` в OAuth 2.0 и какие риски возникают при его отсутствии?
8.  Опишите шаги потока Authorization Code Grant.
9.  В чем техническая разница между реализацией `/logout` и `/logout-all`?
10. Как реализовать защиту от CSRF атак при использовании OAuth?
11. Какие данные безопасно возвращать в ответе профиля пользователя, а какие категорически нельзя?