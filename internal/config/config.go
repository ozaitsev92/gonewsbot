package config

import (
	"sync"
	"time"

	"github.com/cristalhq/aconfig"
	"github.com/cristalhq/aconfig/aconfigdotenv"
)

type Config struct {
	TelegramBotToken     string        `env:"TELEGRAM_BOT_TOKEN" required:"true"`
	TelegramChannelID    int64         `env:"TELEGRAM_CHANNEL_ID" required:"true"`
	DatabaseDSN          string        `env:"DATABASE_DSN" required:"true"`
	FetchInterval        time.Duration `env:"FETCH_INTERVAL" default:"10m"`
	NotificationInterval time.Duration `env:"NOTIFICATION_INTERVAL" default:"1m"`
	FilterKeywords       []string      `env:"FILTER_KEYWORDS"`
	OpenAIKey            string        `env:"OPENAI_KEY" required:"true"`
	OpenAIPrompt         string        `env:"OPENAI_PROMPT" required:"true"`
	OpenAIModel          string        `env:"OPENAI_MODEL" default:"gpt-3.5-turbo"`
	HTTPBindAddress      string        `env:"HTTP_BIND_ADDRESS" default:":8080"`
}

var cfg Config
var once sync.Once

func Get() Config {
	once.Do(func() {
		dotenvLoader := aconfigdotenv.New()

		loader := aconfig.LoaderFor(&cfg, aconfig.Config{
			EnvPrefix: "GONEWSBOT",
			FileFlag:  "config",
			Files:     []string{".env"},
			FileDecoders: map[string]aconfig.FileDecoder{
				".env": dotenvLoader,
			},
			AllowUnknownFields: true,
		})

		if err := loader.Load(); err != nil {
			panic("failed to load config: " + err.Error())
		}
	})

	return cfg
}
