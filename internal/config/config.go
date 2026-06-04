package config

import (
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	App      AppConfig
	DB       DBConfig
	Redis    RedisConfig
	JWT      JWTConfig
	RADIUS   RADIUSConfig
	VPN      VPNConfig
	CORS     CORSConfig
	Log      LogConfig
	Storage  StorageConfig
	GenieACS GenieACSConfig
	WhatsApp WhatsAppConfig
	FCM      FCMConfig
	PG       PGConfig
}

type WhatsAppConfig struct {
	ServiceURL string
	APISecret  string
}

type AppConfig struct {
	Name     string
	Env      string
	Port     string
	Debug    bool
	URL      string // Base URL for callback/webhook URLs (e.g. https://api.example.com)
	CWMPPort string // Port for ACS/CWMP URL shown to customers (e.g. 7547)
}

type DBConfig struct {
	Host            string
	Port            string
	Name            string
	User            string
	Password        string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type JWTConfig struct {
	PrivateKeyPath string
	PublicKeyPath  string
	AccessExpiry   time.Duration
	RefreshExpiry  time.Duration
}

type RADIUSConfig struct {
	ListenAuth string
	ListenAcct string
	Secret     string
}

type VPNConfig struct {
	Interface  string
	Subnet     string
	ServerIP   string
	ListenPort string
	PublicIP   string
}

type CORSConfig struct {
	AllowedOrigins string
}

type LogConfig struct {
	Level  string
	Format string
}

type StorageConfig struct {
	Driver string
	Path   string
}

type GenieACSConfig struct {
	URL      string
	Username string
	Password string
}

type FCMConfig struct {
	ProjectID       string // Firebase project ID
	CredentialsFile string // path to service-account JSON
	Enabled         bool
}

type PGConfig struct {
	Provider   string
	APIKey     string
	SecretKey  string
	MerchantID string
	Sandbox    bool
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		App: AppConfig{
			Name:     getEnv("APP_NAME", "radius-server"),
			Env:      getEnv("APP_ENV", "development"),
			Port:     getEnv("APP_PORT", "3000"),
			Debug:    getEnvBool("APP_DEBUG", true),
			URL:      getEnv("APP_URL", "http://localhost:3000"),
			CWMPPort: getEnv("CWMP_PORT", "7547"),
		},
		DB: DBConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "5432"),
			Name:            getEnv("DB_NAME", "radius_server"),
			User:            getEnv("DB_USER", "postgres"),
			Password:        getEnv("DB_PASSWORD", ""),
			SSLMode:         getEnv("DB_SSL_MODE", "disable"),
			MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getEnvDuration("DB_CONN_MAX_LIFETIME", 30*time.Minute),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			PrivateKeyPath: getEnv("JWT_PRIVATE_KEY_PATH", "./keys/private.pem"),
			PublicKeyPath:  getEnv("JWT_PUBLIC_KEY_PATH", "./keys/public.pem"),
			AccessExpiry:   getEnvDuration("JWT_ACCESS_EXPIRY", 15*time.Minute),
			RefreshExpiry:  getEnvDuration("JWT_REFRESH_EXPIRY", 168*time.Hour),
		},
		RADIUS: RADIUSConfig{
			ListenAuth: getEnv("RADIUS_LISTEN_AUTH", ":1812"),
			ListenAcct: getEnv("RADIUS_LISTEN_ACCT", ":1813"),
			Secret:     getEnv("RADIUS_SECRET", ""),
		},
		VPN: VPNConfig{
			Interface:  getEnv("VPN_INTERFACE", "wg0"),
			Subnet:     getEnv("VPN_SUBNET", "10.10.0.0/24"),
			ServerIP:   getEnv("VPN_SERVER_IP", "10.10.0.1"),
			ListenPort: getEnv("VPN_LISTEN_PORT", "51820"),
			PublicIP:   getEnv("VPN_PUBLIC_IP", ""),
		},
		CORS: CORSConfig{
			AllowedOrigins: getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://localhost:5174"),
		},
		Log: LogConfig{
			Level:  getEnv("LOG_LEVEL", "debug"),
			Format: getEnv("LOG_FORMAT", "console"),
		},
		Storage: StorageConfig{
			Driver: getEnv("STORAGE_DRIVER", "local"),
			Path:   getEnv("STORAGE_PATH", "./storage"),
		},
		GenieACS: GenieACSConfig{
			URL:      getEnv("GENIEACS_URL", "http://localhost:7557"),
			Username: getEnv("GENIEACS_USERNAME", ""),
			Password: getEnv("GENIEACS_PASSWORD", ""),
		},
		WhatsApp: WhatsAppConfig{
			ServiceURL: getEnv("WA_SERVICE_URL", "http://localhost:3001"),
			APISecret:  getEnv("WA_API_SECRET", ""),
		},
		FCM: FCMConfig{
			ProjectID:       getEnv("FCM_PROJECT_ID", ""),
			CredentialsFile: getEnv("FCM_CREDENTIALS_FILE", "./keys/fcm-service-account.json"),
			Enabled:         getEnvBool("FCM_ENABLED", false),
		},
		PG: PGConfig{
			Provider:   getEnv("PG_PROVIDER", "tripay"),
			APIKey:     getEnv("PG_API_KEY", ""),
			SecretKey:  getEnv("PG_SECRET_KEY", ""),
			MerchantID: getEnv("PG_MERCHANT_ID", ""),
			Sandbox:    getEnvBool("PG_SANDBOX", true),
		},
	}

	return cfg, nil
}

func (c *Config) IsProduction() bool {
	return c.App.Env == "production"
}

func (c *Config) DSN() string {
	u := &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(c.DB.User, c.DB.Password),
		Host:     c.DB.Host + ":" + c.DB.Port,
		Path:     c.DB.Name,
		RawQuery: "sslmode=" + url.QueryEscape(c.DB.SSLMode),
	}
	return u.String()
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	val, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		return fallback
	}
	return b
}

func getEnvInt(key string, fallback int) int {
	val, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return i
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	val, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	d, err := time.ParseDuration(val)
	if err != nil {
		return fallback
	}
	return d
}
