package astroApi

// todo рефактор юрла
type Config struct {
	BaseURL    string `envconfig:"BASE_URL"`
	ApiVersion string `envconfig:"VERSION"`
	SkipSSL    string `envconfig:"SKIP_SSL"` // Railway требует строки вместо bool
}

func (c *Config) ShouldSkipSSL() bool {
	return c.SkipSSL == "true" || c.SkipSSL == "1" || c.SkipSSL == "True"
}
