package config

import (
	"strings"
	"time"

	"github.com/ncondes/fifawcp/internal/infrastructure/env"
)

type Config struct {
	APIBaseURL string
	Server     ServerConfig
	Port       string
	Env        string
	DB         DBConfig
	Redis      RedisConfig
	JWT        JWTConfig
	Auth       AuthConfig
	Mailer     MailerConfig
	Cron       CronConfig
	RateLimit  RateLimitConfig
}

type ServerConfig struct {
	ContextTimeout time.Duration
	WriteTimeout   time.Duration
	ReadTimeout    time.Duration
	IdleTimeout    time.Duration
	CORS           CORSConfig
}

type CORSConfig struct {
	AllowedOrigins []string
}

type DBConfig struct {
	Address      string
	MaxOpenConns int
	MaxIdleConns int
	MaxLifetime  time.Duration
	QueryTimeout time.Duration
}

type RedisConfig struct {
	Address      string
	Password     string
	DB           int
	PoolSize     int
	QueryTimeout time.Duration
	UserCacheTTL time.Duration
}

type JWTConfig struct {
	Secret             string
	Audience           string
	Issuer             string
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration
}

type AuthConfig struct {
	SessionTTL     time.Duration
	OTPTTL         time.Duration
	MaxOTPAttempts int
	OTPCooldown    time.Duration
}

type MailerConfig struct {
	APIKey      string
	FromAddress string
}

type CronConfig struct {
	CleanupSessionsSchedule string
}

type RateLimitConfig struct {
	Enabled    bool
	StrictIP   RateLimitTier
	ModerateIP RateLimitTier
	RelaxedIP  RateLimitTier
}

type RateLimitTier struct {
	RequestsPerWindow int
	Window            time.Duration
}

func NewConfig() *Config {
	return &Config{
		APIBaseURL: env.GetString("API_BASE_URL", "http://localhost:8080"),
		Port:       env.GetString("PORT", "8080"),
		Env:        env.GetString("ENV", "development"),
		Server: ServerConfig{
			ContextTimeout: env.GetDuration("SERVER_CONTEXT_TIMEOUT", 30*time.Second),
			WriteTimeout:   env.GetDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),
			ReadTimeout:    env.GetDuration("SERVER_READ_TIMEOUT", 30*time.Second),
			IdleTimeout:    env.GetDuration("SERVER_IDLE_TIMEOUT", 60*time.Second),
			CORS: CORSConfig{
				AllowedOrigins: strings.Split(env.GetString("CORS_ALLOWED_ORIGINS", "*"), ","),
			},
		},
		DB: DBConfig{
			Address:      env.GetString("DB_ADDRESS", "postgres://postgres:password@localhost:5432/pickems?sslmode=disable"),
			MaxOpenConns: env.GetInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns: env.GetInt("DB_MAX_IDLE_CONNS", 5),
			MaxLifetime:  env.GetDuration("DB_MAX_LIFETIME", 5*time.Minute),
			QueryTimeout: env.GetDuration("DB_QUERY_TIMEOUT", 5*time.Second),
		},
		Redis: RedisConfig{
			Address:      env.GetString("REDIS_ADDRESS", "localhost:6379"),
			Password:     env.GetString("REDIS_PASSWORD", "password"),
			DB:           env.GetInt("REDIS_DB", 0),
			PoolSize:     env.GetInt("REDIS_POOL_SIZE", 10),
			QueryTimeout: env.GetDuration("REDIS_QUERY_TIMEOUT", 2*time.Second),
			UserCacheTTL: env.GetDuration("REDIS_USER_CACHE_TTL", 15*time.Minute),
		},
		JWT: JWTConfig{
			Secret:             env.GetString("JWT_SECRET", "secret"),
			Audience:           env.GetString("JWT_AUDIENCE", "fifa-wcp"),
			Issuer:             env.GetString("JWT_ISSUER", "fifa-wcp"),
			AccessTokenExpiry:  env.GetDuration("JWT_ACCESS_TOKEN_EXPIRY", 15*time.Minute),
			RefreshTokenExpiry: env.GetDuration("JWT_REFRESH_TOKEN_EXPIRY", 7*24*time.Hour),
		},
		Auth: AuthConfig{
			SessionTTL:     env.GetDuration("AUTH_SESSION_TTL", 7*24*time.Hour),
			OTPTTL:         env.GetDuration("AUTH_OTP_TTL", 10*time.Minute),
			MaxOTPAttempts: env.GetInt("AUTH_MAX_OTP_ATTEMPTS", 3),
			OTPCooldown:    env.GetDuration("AUTH_OTP_COOLDOWN", 30*time.Second),
		},
		Mailer: MailerConfig{
			APIKey:      env.GetString("MAILER_API_KEY", ""),
			FromAddress: env.GetString("MAILER_FROM_ADDRESS", ""),
		},
		Cron: CronConfig{
			CleanupSessionsSchedule: env.GetString("CRON_CLEANUP_SESSIONS_SCHEDULE", "0 0 * * *"), // Every day at midnight
		},
		RateLimit: RateLimitConfig{
			Enabled: env.GetBool("RATE_LIMIT_ENABLED", true),
			StrictIP: RateLimitTier{
				RequestsPerWindow: env.GetInt("RATE_LIMIT_STRICT_IP_REQUESTS_PER_WINDOW", 10),
				Window:            env.GetDuration("RATE_LIMIT_STRICT_IP_WINDOW", 1*time.Hour),
			},
			ModerateIP: RateLimitTier{
				RequestsPerWindow: env.GetInt("RATE_LIMIT_MODERATE_IP_REQUESTS_PER_WINDOW", 20),
				Window:            env.GetDuration("RATE_LIMIT_MODERATE_IP_WINDOW", 15*time.Minute),
			},
			RelaxedIP: RateLimitTier{
				RequestsPerWindow: env.GetInt("RATE_LIMIT_RELAXED_IP_REQUESTS_PER_WINDOW", 60),
				Window:            env.GetDuration("RATE_LIMIT_RELAXED_IP_WINDOW", 1*time.Hour),
			},
		},
	}
}

func (c *Config) IsProd() bool {
	return strings.EqualFold(c.Env, "production") || strings.EqualFold(c.Env, "prod")
}
