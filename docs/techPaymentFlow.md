# Technical Payment Flow - Техническая документация платежной системы

Этот документ описывает техническую архитектуру, математику и реализацию платежной системы.

## 🏗️ Архитектура по слоям (Clean Architecture)

### Слой адаптеров (Adapters)

#### Primary Adapters (входящие интерфейсы)
- **`internal/adapters/primary/http/controllers/telegram/controllerWebhook.go`**
  - Принимает webhook от Telegram с `pre_checkout_query` и `successful_payment`
  - Маршрутизирует в `telegram.Service`

#### Secondary Adapters (исходящие интерфейсы)
- **`internal/adapters/secondary/payment/telegram_stars/provider.go`**
  - Реализует `IPaymentProvider` для Telegram Stars
  - `CreateInvoice()` → вызывает `telegram.Client.SendInvoice()`
  - `ConfirmPreCheckout()` → вызывает `telegram.Client.AnswerPreCheckoutQuery()`
  - Поддерживает **несколько ботов** через `map[string]*telegram.Client`

- **`internal/adapters/secondary/telegram/client.go`**
  - Низкоуровневый клиент Telegram Bot API
  - Методы: `SendInvoice()`, `AnswerPreCheckoutQuery()`

- **`internal/adapters/secondary/storage/pg/`**
  - PostgreSQL репозитории для `payments` и `tg_users`
  - Миграции: `0005_create_payments_table.sql`, `0006_add-bot-id-to-payments.sql`, `0007_add-payment-fields-to-users.sql`

---

### Слой портов (Ports / Interfaces)

#### Service Ports
- **`internal/ports/service/payment.go`**
  ```go
  type IPaymentService interface {
      CreatePayment(ctx, botID, userID, chatID, productID, title, description, amount) (*Payment, error)
      HandlePreCheckoutQuery(ctx, botID, queryID, userID, amount, currency, payload) (bool, error)
      HandleSuccessfulPayment(ctx, botID, userID, chatID, paymentID, chargeID) error
  }
  ```

#### Repository Ports
- **`internal/ports/repository/payment.go`**
  ```go
  type IPaymentRepo interface {
      Create(ctx, payment) error
      GetByID(ctx, id) (*Payment, error)
      GetByProviderID(ctx, providerID) (*Payment, error)
      GetByPayload(ctx, payload) (*Payment, error)
      UpdateStatus(ctx, id, status, succeededAt, failedAt, errorMessage) error
      GetLastSuccessfulPaymentDate(ctx, userID) (*time.Time, error)
      GetBotIDForUser(ctx, userID) (string, error)
  }
  ```

- **`internal/ports/repository/user.go`**
  ```go
  type IUserRepo interface {
      SetPaidStatus(ctx, userID, isPaid) error
      UpdateFreeMsgCount(ctx, userID) error
      GetUsersWithExpiredSubscriptions(ctx) ([]uuid.UUID, error)
      RevokeExpiredSubscriptions(ctx) (int64, error)
  }
  ```

#### Provider Ports
- **`internal/ports/payment/provider.go`**
  ```go
  type IPaymentProvider interface {
      CreateInvoice(ctx, req CreateInvoiceRequest) (*CreateInvoiceResult, error)
      ConfirmPreCheckout(ctx, botID, queryID, ok, errorMessage) error
  }
  ```

---

### Слой сервисов (Services)

- **`internal/services/telegram/`**
  - `module.go`: Маршрутизация обновлений Telegram (включая платежи)
  - `payment_handler.go`: Обработка `pre_checkout_query` и `successful_payment`
  - Интегрирован с `PaymentUseCase` через интерфейс

- **`internal/services/jobs/subscription_expirer.go`**
  - Джоба для автоматического отзыва истёкших подписок
  - Запускается каждый день в **03:00 по МСК**
  - Вызывает `astroUsecase.RevokeExpiredSubscriptions()`

---

### Слой use cases (Business Logic)

- **`internal/usecases/payment/module.go`**
  - Реализует `IPaymentService`
  - Содержит бизнес-логику:
    - `CreatePayment()`: создание платежа в БД, отправка invoice
    - `HandlePreCheckoutQuery()`: валидация платежа перед оплатой
    - `HandleSuccessfulPayment()`: обработка успешной оплаты, выдача продукта
    - `grantProduct()`: установка `is_paid = true`, сброс `free_msg_count = 0`

- **`internal/usecases/astro/`**
  - `text_handler.go`: проверка лимита бесплатных сообщений, инициация платежа
  - `commands.go`: команда `/buy` для ручной инициации платежа
  - `subscription.go`: логика отзыва истёкших подписок, уведомления пользователям

---

## 🔢 Математика и бизнес-логика

### Лимиты и счётчики

#### Бесплатный лимит
```
FREE_MESSAGES_LIMIT = 15 (настраивается через env: ASTRO_FREE_MESSAGES_LIMIT)
```

#### Логика проверки лимита
```go
isPaidUser = user.IsPaid || user.ManualGranted

if !isPaidUser && user.FreeMsgCount >= FREE_MESSAGES_LIMIT {
    // Показать invoice, заблокировать отправку в RAG
}
```

#### Инкремент счётчика
- Выполняется только если: `!isPaidUser` и сообщение отправляется в RAG
- Команды (например, `/my_info`) **не тратят** бесплатные сообщения

---

### TTL подписки

```
SUBSCRIPTION_TTL = 30 дней

expiryDate = lastPayment.succeeded_at + 30 дней

if now > expiryDate {
    // Отозвать подписку: is_paid = false
}
```

#### SQL-запрос для проверки истёкших подписок
```sql
SELECT DISTINCT u.id
FROM tg_users u
INNER JOIN (
    SELECT user_id, MAX(succeeded_at) as last_payment_date
    FROM payments
    WHERE status = 'succeeded' AND succeeded_at IS NOT NULL
    GROUP BY user_id
) p ON u.id = p.user_id
WHERE u.is_paid = true
  AND u.manual_granted = false
  AND (p.last_payment_date AT TIME ZONE 'Europe/Moscow' AT TIME ZONE 'UTC') 
      < NOW() - INTERVAL '30 days'
```

**Важно:** Конвертация часового пояса нужна, т.к. `succeeded_at` хранится как `timestamp without time zone` (подразумевается Moscow), а `NOW()` возвращает UTC.

---

### Статусы платежа

```go
type PaymentStatus string

const (
    PaymentStatusPending   = "pending"   // Создан, invoice отправлен, ожидает оплаты
    PaymentStatusSucceeded = "succeeded" // Успешно оплачен, продукт выдан
    PaymentStatusFailed    = "failed"    // Ошибка при создании invoice или отклонён
)
```

#### State Machine
```
[pending] → [succeeded] (при успешной оплате)
[pending] → [failed]    (при ошибке создания invoice или отклонении в pre-checkout)
```

---

## 🤖 Работа с несколькими ботами

### Архитектура поддержки мультибота

#### Хранение `bot_id` в платежах
- Каждый платёж привязан к конкретному `bot_id` (столбец `bot_id` в таблице `payments`)
- `bot_id` передаётся через все слои: `CreatePayment(botID, ...)`, `HandlePreCheckoutQuery(botID, ...)`

#### Провайдер Telegram Stars
- `Provider` хранит `map[string]*telegram.Client` (botID → Client)
- При создании invoice использует правильный клиент для `botID`:
  ```go
  client, err := p.getClient(req.BotID)
  ```

#### Отзыв подписок
- При отзыве подписки система получает `bot_id` из последнего успешного платежа:
  ```go
  botID, err := s.PaymentRepo.GetBotIDForUser(ctx, userID)
  ```
- Уведомление отправляется в правильный бот

---

### Ограничения текущей реализации

**Проблема:** `is_paid` и `free_msg_count` — это **глобальные** флаги для пользователя, не привязанные к конкретному боту.

**Сценарий:**
- Пользователь оплатил доступ в боте `astro1`
- `is_paid = true`, `free_msg_count = 0`
- Пользователь открывает бота `astro2`
- В боте `astro2` у него тоже `is_paid = true` (хотя он не оплачивал)

**Решение:** Нужно добавить таблицу `user_subscriptions` с полями `user_id`, `bot_id`, `is_paid`, `expires_at`, чтобы каждый бот имел свой учёт подписок.

---

## 📊 База данных

### Таблица `payments`
```sql
CREATE TABLE payments (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES tg_users(id),
    bot_id VARCHAR(255) NOT NULL,  -- Для мультибота
    amount BIGINT NOT NULL,
    currency VARCHAR(10) NOT NULL DEFAULT 'XTR',
    method VARCHAR(50) NOT NULL,
    provider_id VARCHAR(255) NOT NULL,  -- invoice_id от Telegram
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    product_id VARCHAR(100) NOT NULL,
    product_title VARCHAR(255) NOT NULL,
    metadata JSONB DEFAULT '{}',  -- payload хранится здесь
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    succeeded_at TIMESTAMP,
    failed_at TIMESTAMP,
    error_message TEXT
);
```

**Индексы:**
- `idx_payments_user_id` - для поиска платежей пользователя
- `idx_payments_provider_id` - для поиска по `invoice_id`
- `idx_payments_bot_id` - для фильтрации по боту
- `idx_payments_metadata_payload` - для поиска по `payload` (B-tree индекс на `metadata->>'payload'`)

### Таблица `tg_users` (добавленные поля)
```sql
ALTER TABLE tg_users ADD COLUMN is_paid BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE tg_users ADD COLUMN free_msg_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE tg_users ADD COLUMN manual_granted BOOLEAN NOT NULL DEFAULT FALSE;
```

---

## 🔄 Полный технический флоу

### 1. Инициация платежа

```
User → text_handler.go (handleUserQuestion)
    ↓
Check: isPaidUser || FreeMsgCount < LIMIT
    ↓
[If limit reached]
    ↓
PaymentService.CreatePayment(botID, userID, chatID, ...)
    ↓
PaymentRepo.Create(payment) → DB [status='pending']
    ↓
PaymentProvider.CreateInvoice(botID, ...) → Telegram API
    ↓
User receives invoice in Telegram
```

---

### 2. Pre-checkout validation

```
Telegram → webhook (pre_checkout_query)
    ↓
telegram.Service.HandlePreCheckoutQuery(botID, query)
    ↓
PaymentUseCase.HandlePreCheckoutQuery(botID, queryID, userID, amount, currency, payload)
    ↓
PaymentRepo.GetByPayload(payload) → DB
    ↓
Validations:
    - payment exists
    - payment.user_id == query.user_id
    - payment.amount == query.amount
    - payment.currency == query.currency
    - payment.status == 'pending'
    ↓
PaymentProvider.ConfirmPreCheckout(botID, queryID, ok=true/false, errorMessage)
    ↓
Telegram allows/denies payment
```

---

### 3. Successful payment

```
Telegram → webhook (successful_payment)
    ↓
telegram.Service.HandleSuccessfulPayment(botID, message)
    ↓
PaymentUseCase.HandleSuccessfulPayment(botID, userID, chatID, paymentID, chargeID)
    ↓
PaymentRepo.GetByID(paymentID) → DB
    ↓
Validations:
    - payment exists
    - payment.user_id == message.user_id
    - payment.status == 'pending'
    ↓
PaymentRepo.UpdateStatus(paymentID, 'succeeded', succeeded_at=NOW())
    ↓
PaymentUseCase.grantProduct(botID, userID, productID)
    ↓
UserRepo.SetPaidStatus(userID, isPaid=true)
    → DB: is_paid=true, free_msg_count=0
    ↓
TelegramService.SendMessage("✅ Платёж успешно обработан!")
```

---

### 4. Subscription expiry (джоба)

```
Scheduler (03:00 daily) → subscription_expirer.Run()
    ↓
astroUsecase.RevokeExpiredSubscriptions()
    ↓
UserRepo.GetUsersWithExpiredSubscriptions()
    → SQL: WHERE last_payment_date < NOW() - 30 days
    ↓
UserRepo.RevokeExpiredSubscriptions()
    → SQL: UPDATE tg_users SET is_paid=false WHERE id IN (...)
    ↓
For each expired user:
    UserRepo.GetByID(userID) → get chat_id
    PaymentRepo.GetBotIDForUser(userID) → get bot_id
    TelegramService.SendMessage(botID, chatID, "Подписка закончилась...")
    ↓
[Wait 0.1s between messages] (rate limit compliance)
```

---

## 🔐 Безопасность и валидация

### Валидация в Pre-checkout
1. **Платёж существует** — проверка по `payload` (UUID из metadata)
2. **Платёж принадлежит пользователю** — `payment.user_id == query.user_id`
3. **Сумма совпадает** — `payment.amount == query.amount`
4. **Валюта совпадает** — `payment.currency == query.currency`
5. **Статус pending** — защита от повторной обработки

### Идемпотентность
- `HandleSuccessfulPayment` проверяет `status == 'pending'` перед обработкой
- Если статус уже `succeeded` → игнорируется (защита от дубликатов webhook)

### Обработка ошибок
- Если `grantProduct()` не удался после оплаты → логируется ошибка, отправляется алерт администратору
- Платёж остаётся `succeeded` (деньги уже списаны), доступ будет выдан вручную

---

## 🚀 Инициализация

### Порядок инициализации (app/init.go)
1. Инициализация репозиториев (`PaymentRepo`, `UserRepo`)
2. Инициализация провайдера (`TelegramStarsProvider`)
3. Инициализация `PaymentUseCase`
4. Интеграция в `TelegramService` через `SetPaymentUseCase()`
5. Интеграция в `AstroUseCase` через `SetPaymentService()` и `SetPaymentRepo()`

---

## 📝 Конфигурация

### Environment variables
```env
# Лимит бесплатных сообщений
ASTRO_FREE_MESSAGES_LIMIT=15

# Боты (каждый бот может иметь свой токен)
TG_BOTS_BOTS_0_BOT_ID=astro1
TG_BOTS_BOTS_0_TOKEN=...
```

---

## 🧪 Тестирование

### Ручная инициация платежа
```
Команда: /buy или /test_payment
→ Создаётся тестовый invoice (amount=1 звезда)
```

### Тестовые интервалы (для разработки)
- В `subscription_expirer.go` можно временно установить `NextRun()` на 20 секунд
- В SQL-запросах можно временно заменить `'30 days'` на `'10 seconds'`

**Важно:** После тестирования вернуть на продакшн значения!
