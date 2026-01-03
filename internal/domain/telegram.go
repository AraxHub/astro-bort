package domain

// дока - https://core.telegram.org/bots/api

// Update - входящее обновление от Telegram Bot API
type Update struct {
	UpdateID      int64          `json:"update_id"`
	Message       *Message       `json:"message,omitempty"`
	CallbackQuery *CallbackQuery `json:"callback_query,omitempty"`
	// Можно добавить другие типы обновлений по мере необходимости:
	// EditedMessage      *Message `json:"edited_message,omitempty"`
	// и т.д.
}

// CallbackQuery - callback query от Telegram Bot API
type CallbackQuery struct {
	ID      string        `json:"id"`
	From    *TelegramUser `json:"from,omitempty"`
	Message *Message      `json:"message,omitempty"`
	Data    *string       `json:"data,omitempty"` // данные callback кнопки
}

// Message - сообщение от Telegram Bot API
type Message struct {
	MessageID int64         `json:"message_id"`
	From      *TelegramUser `json:"from,omitempty"`     // отправитель (Telegram User)
	Chat      *Chat         `json:"chat"`               // чат
	Date      int64         `json:"date"`               // Unix timestamp
	Text      *string       `json:"text,omitempty"`     // текст сообщения
	Entities  []Entity      `json:"entities,omitempty"` // сущности (команды, упоминания и т.д.)
}

// User - пользователя Telegram (не domain.User)
type TelegramUser struct {
	ID           int64   `json:"id"`
	IsBot        bool    `json:"is_bot"`
	FirstName    string  `json:"first_name"`
	LastName     *string `json:"last_name,omitempty"`
	Username     *string `json:"username,omitempty"`
	LanguageCode *string `json:"language_code,omitempty"`
}

// Chat - чат в Telegram
type Chat struct {
	ID        int64   `json:"id"`
	Type      string  `json:"type"` // "private", "group", "supergroup", "channel"
	Title     *string `json:"title,omitempty"`
	Username  *string `json:"username,omitempty"`
	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
}

// Entity - сущность в сообщении (команда, упоминание и т.д.)
type Entity struct {
	Type   string `json:"type"`   // "bot_command", "mention", "url" и т.д.
	Offset int    `json:"offset"` // смещение в UTF-16 кодовых единицах
	Length int    `json:"length"` // длина в UTF-16 кодовых единицах
}

type BotType string

const (
	BotTypeAstro BotType = "astro"
)

func (bt BotType) String() string {
	return string(bt)
}

func (bt BotType) IsValid() bool {
	switch bt {
	case BotTypeAstro:
		return true
	default:
		return false
	}
}

type BotId string
