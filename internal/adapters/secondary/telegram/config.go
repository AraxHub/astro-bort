package telegram

type Config struct {
	BotToken       string `envconfig:"BOT_TOKEN"`
	UseWebhook     string `envconfig:"USE_WEBHOOK"` // Railway требует строки
	WebhookURL     string `envconfig:"WEBHOOK_URL"`
	PollingTimeout int    `envconfig:"POLLING_TIMEOUT"`
}

// IsWebhookEnabled парсит строку UseWebhook в boolean
func (c *Config) IsWebhookEnabled() bool {
	return c.UseWebhook == "true" || c.UseWebhook == "1" || c.UseWebhook == "True"
}
