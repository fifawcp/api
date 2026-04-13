package config

import (
	"time"

	"github.com/ncondes/fifa-world-cup-pickems/internal/infrastructure/env"
)

type Config struct {
	Server ServerConfig
	Port   string
	Env    string
	DB     DBConfig
	Redis  RedisConfig
	JWT    JWTConfig
	Auth   AuthConfig
}

type ServerConfig struct {
	ContextTimeout time.Duration
	WriteTimeout   time.Duration
	ReadTimeout    time.Duration
	IdleTimeout    time.Duration
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

func NewConfig() *Config {
	return &Config{
		Port: env.GetString("PORT", "8080"),
		Env:  env.GetString("ENV", "development"),
		Server: ServerConfig{
			ContextTimeout: env.GetDuration("SERVER_CONTEXT_TIMEOUT", 30*time.Second),
			WriteTimeout:   env.GetDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),
			ReadTimeout:    env.GetDuration("SERVER_READ_TIMEOUT", 30*time.Second),
			IdleTimeout:    env.GetDuration("SERVER_IDLE_TIMEOUT", 60*time.Second),
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
	}
}
