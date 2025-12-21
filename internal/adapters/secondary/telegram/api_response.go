package telegram

// APIResponse базовая структура ответа от Telegram API
type APIResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description,omitempty"`
	ErrorCode   int    `json:"error_code,omitempty"`
}

