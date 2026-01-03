# Webhook Architecture

## Проблема

Telegram webhook требует установки через `setWebhook` для каждого бота. При использовании нескольких ботов нужно решить, как идентифицировать бота при получении обновлений.

## Решение: Secret Token (Вариант 1) ⭐ Рекомендуется

### Концепция

Использовать стандартный механизм Telegram - `secret_token` для идентификации бота:
- Один endpoint: `/webhook` (без `bot_id` в URL)
- При установке webhook устанавливаем `secret_token = bot_id`
- Telegram отправляет secret в заголовке `X-Telegram-Bot-Api-Secret-Token`
- Извлекаем `bot_id` из заголовка

### Преимущества

1. ✅ **Один endpoint** - проще роутинг, нет множества эндпоинтов
2. ✅ **Безопасность** - Telegram валидирует secret, стандартный механизм
3. ✅ **Стандартный подход** - используем встроенную функциональность Telegram API
4. ✅ **Не нужно хранить bot_id в URL** - чище и безопаснее

### Реализация

#### Endpoint
```go
router.POST("/webhook", c.handleWebhook)
```

#### Контроллер
```go
func (c *Controller) handleWebhook(ctx *gin.Context) {
    // Извлекаем bot_id из secret_token заголовка
    secretToken := ctx.GetHeader("X-Telegram-Bot-Api-Secret-Token")
    if secretToken == "" {
        ctx.JSON(http.StatusUnauthorized, gin.H{"error": "secret token required"})
        return
    }
    
    botID := domain.BotId(secretToken)
    // ... обработка update
}
```

#### Установка webhook при старте
```go
// Для каждого бота при UseWebhook=true
for _, botConfig := range botConfigs {
    webhookURL := fmt.Sprintf("%s/webhook", cfg.Telegram.WebhookURL)
    err := telegramClient.SetWebhook(ctx, webhookURL, string(botConfig.BotID))
    // secret_token = bot_id
}
```

### API Telegram

```
POST https://api.telegram.org/bot{token}/setWebhook
{
  "url": "https://domain.com/webhook",
  "secret_token": "astro"  // bot_id
}
```

Telegram будет отправлять:
```
POST https://domain.com/webhook
Headers:
  X-Telegram-Bot-Api-Secret-Token: astro
Body:
  { "update_id": 123, "message": {...} }
```

## Альтернативные варианты

### Вариант 2: bot_id в URL (текущий)

- Endpoint: `/webhook/:bot_id`
- При старте устанавливаем webhook для каждого бота: `https://domain.com/webhook/astro`
- Работает, но нужно вызывать `setWebhook` для каждого бота

**Плюсы:**
- Просто и явно
- Легко дебажить

**Минусы:**
- Нужно устанавливать webhook для каждого бота
- Много эндпоинтов (хотя это не проблема)

### Вариант 3: Определение бота по содержимому Update

**Проблема:** Update не содержит информацию о том, какой бот его отправил. Ненадёжно.

## TODO

- [ ] Реализовать метод `SetWebhook` в Telegram Client
- [ ] Добавить установку webhook при старте приложения (если `UseWebhook=true`)
- [ ] Обновить контроллер для использования secret_token вместо bot_id в URL
- [ ] Обновить роутинг: `/webhook/:bot_id` → `/webhook`
