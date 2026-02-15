# Astro Bot

Telegram бот для астрологических консультаций на основе натальной карты пользователя.

## Контракты Kafka

### Отправка запросов в RAG (topic: `requests`)

**Key:** `request_id` (UUID)

**Headers:**
- `action` (обязательный): `"chat"` | `"prediction"` | `"rerank_natal"`
- `onboarding` (bool, опционально)
- `summarize` (bool, опционально)
- `more` (bool, опционально)
- `need_photo` (bool, опционально)

**Value (JSON):**
```json
{
  "request_id": "uuid",
  "bot_id": "astro1",
  "chat_id": 123456789,
  "request_text": "текст запроса",
  "natal_chart": { /* JSON */ }
}
```

### Формирование headers по бизнес-логике

| Сценарий | `onboarding` | `summarize` | `more` | `need_photo` | `request_text` |
|----------|--------------|-------------|--------|--------------|----------------|
| Онбординг (`< 2`) | `true` | `false` | `false` | `false` | текст вопроса |
| Summarize (`== 2`) | `false` | `true` | `false` | `false` | текст вопроса |
| Стандартный (`>= 3`) | `false` | `false` | `false` | `true` | текст вопроса |
| Кнопка "Расскажи обо мне" | `false` | `true` | `false` | `true` | `""` |
| Кнопка "Раскрой тему глубже" | `false` | `false` | `true` | `true` | `""` |
| Rerank натальной карты | - | - | - | - | `""` |

**Особые случаи:**
- **Rerank натальной карты:** `action: "rerank_natal"`, headers: `bot_id`, `chat_id`, value: `request_text: ""`, `natal_chart`
- **Пуш "Прогноз на неделю":** `action: "prediction"`, без дополнительных headers

### Получение ответов от RAG (topic: `responses`)

**Key:** `request_id` (UUID)

**Headers:**
- `action`: `"chat"` | `"image_type"` | `"Nothing"`

**Value (JSON):**
```json
{
  "request_id": "uuid",
  "bot_id": "astro1",
  "chat_id": 123456789,
  "response_text": "текст ответа или тема для фото"
}
```

**Обработка:**
- `action: "chat"` → текстовый ответ, удаление технического сообщения, клавиатура (если `onboarding_count >= 3`)
- `action: "image_type"` → отправка фото, техническое сообщение остаётся
- `action: "Nothing"` → игнорируется
