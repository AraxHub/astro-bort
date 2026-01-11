package alerter

type Config struct {
	BotToken        string `envconfig:"BOT_TOKEN"`
	ChatID          int64  `envconfig:"CHAT_ID"`
	MessageThreadID *int64 `envconfig:"MESSAGE_THREAD_ID"`
}
