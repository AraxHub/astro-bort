package telegram

type Config struct {
	BotToken       string `envconfig:"BOT_TOKEN"`
	UseWebhook     bool   `envconfig:"USE_WEBHOOK"` // webhook prod || polling local
	WebhookURL     string `envconfig:"WEBHOOK_URL"`
	PollingTimeout int    `envconfig:"POLLING_TIMEOUT"`
}
