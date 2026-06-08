package config

type Config struct {
	LLM      LLM      `mapstructure:"llm"`
	Telegram Telegram `mapstructure:"telegram"`
}

type LLM struct {
	Provider string `mapstructure:"provider"`
	Model    string `mapstructure:"model"`
	APIKey   string `mapstructure:"api_key"`
	BaseURL  string `mapstructure:"base_url"`
}

type Telegram struct {
	Token  string `mapstructure:"token"`
	ChatID string `mapstructure:"chat_id"`
}
