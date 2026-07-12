package config

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Auth     AuthConfig     `mapstructure:"auth"`
	Database DatabaseConfig `mapstructure:"database"`
	Logging  LoggingConfig  `mapstructure:"logging"`
	LLM      LLMConfig      `mapstructure:"llm"`
	CNN      CNNConfig      `mapstructure:"cnn"`
	CORS     CORSConfig     `mapstructure:"cors"`
}

type AppConfig struct {
	Name string `mapstructure:"name"`
	Env  string `mapstructure:"env"`
	Port int    `mapstructure:"port"`
}

type AuthConfig struct {
	JWTSecret string `mapstructure:"jwt_secret"`
}

type DatabaseConfig struct {
	DSN string `mapstructure:"dsn"`
}

type LoggingConfig struct {
	Level string `mapstructure:"level"`
	Debug bool   `mapstructure:"debug"`
}

type LLMConfig struct {
	Provider string        `mapstructure:"provider"`
	APIURL   string        `mapstructure:"api_url"`
	APIKey   string        `mapstructure:"api_key"`
	Model    string        `mapstructure:"model"`
	Timeout  time.Duration `mapstructure:"timeout"`
}

type CNNConfig struct {
	Mode string `mapstructure:"mode"`
	URL  string `mapstructure:"url"`
}

type CORSConfig struct {
	AllowedOrigins []string `mapstructure:"allowed_origins"`
}

var AppConfigInstance *Config

func LoadConfig(path string) (*Config, error) {
	var cfg Config

	viper.SetConfigFile(path)

	viper.AutomaticEnv()

	viper.SetEnvKeyReplacer(
		strings.NewReplacer(".", "_"),
	)

	// =====================================================
	// DEFAULTS
	// =====================================================

	viper.SetDefault("app.name", "smart-farming-engine")
	viper.SetDefault("app.env", "production")
	viper.SetDefault("app.port", 8181)
	viper.SetDefault("database.dsn", "")
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.debug", false)
	viper.SetDefault("auth.jwt_secret", "")
	viper.SetDefault("llm.provider", "groq")
	viper.SetDefault("llm.api_url", "https://api.groq.com/openai/v1")
	viper.SetDefault("llm.api_key", "")
	viper.SetDefault("llm.model", "qwen/qwen3-32b")

	// Environment variable eksplisit agar credential tidak perlu ditulis di config.yml.
	_ = viper.BindEnv("app.env", "APP_ENV")
	_ = viper.BindEnv("app.port", "APP_PORT")
	_ = viper.BindEnv("database.dsn", "DATABASE_DSN", "DB_DSN")
	_ = viper.BindEnv("logging.level", "LOG_LEVEL")
	_ = viper.BindEnv("logging.debug", "LOG_DEBUG")
	_ = viper.BindEnv("auth.jwt_secret", "JWT_SECRET", "AUTH_JWT_SECRET")
	_ = viper.BindEnv("llm.provider", "LLM_PROVIDER")
	_ = viper.BindEnv("llm.api_url", "LLM_API_URL")
	_ = viper.BindEnv("llm.api_key", "LLM_API_KEY", "GROQ_API_KEY")
	_ = viper.BindEnv("llm.model", "LLM_MODEL")
	_ = viper.BindEnv("llm.timeout", "LLM_TIMEOUT")
	_ = viper.BindEnv("cnn.mode", "CNN_MODE")
	_ = viper.BindEnv("cnn.url", "CNN_URL")

	// Backend-v2 memakai CNN on-device di Flutter secara default.
	// Backend tetap dapat memakai server API jika cnn.mode=server_api dan cnn.url diisi.
	viper.SetDefault("cnn.mode", "on_device_flutter")
	viper.SetDefault(
		"cnn.url",
		"",
	)

	viper.SetDefault(
		"llm.timeout",
		30*time.Second,
	)

	viper.SetDefault(
		"cors.allowed_origins",
		[]string{
			"http://localhost",
			"http://localhost:3000",
			"http://localhost:5173",

			"https://petanitech.com",
			"https://www.petanitech.com",

			"https://app.petanitech.com",
		},
	)

	// =====================================================
	// CONFIG FILE (OPTIONAL)
	// =====================================================

	if err := viper.ReadInConfig(); err != nil {
		log.Printf(
			"config file not found, using defaults/env vars: %v",
			err,
		)
	}

	// =====================================================
	// UNMARSHAL
	// =====================================================

	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf(
			"unable to decode config: %w",
			err,
		)
	}

	if err := validateConfig(&cfg); err != nil {
		return nil, err
	}

	AppConfigInstance = &cfg

	log.Printf(
		"config loaded successfully (env=%s)",
		cfg.App.Env,
	)

	return AppConfigInstance, nil
}

func validateConfig(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	if strings.TrimSpace(cfg.Database.DSN) == "" {
		return fmt.Errorf("database dsn is required; set DATABASE_DSN or database.dsn")
	}

	if strings.EqualFold(strings.TrimSpace(cfg.App.Env), "production") && strings.TrimSpace(cfg.Auth.JWTSecret) == "" {
		return fmt.Errorf("jwt secret is required in production; set JWT_SECRET")
	}

	return nil
}
