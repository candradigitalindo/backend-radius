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
	VPN       VPNConfig
	LegacyVPN LegacyVPNConfig
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
	URL      string // Base URL for callback/webhook URLs (e.g. https://app.dradius.net)
	CWMPURL  string // Full ACS URL shown to ONT devices (e.g. https://app.dradius.net/cwmp); falls back to IP:port if empty
	CWMPPort string // Port for ACS/CWMP fallback URL (e.g. 7547)
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

// LegacyVPNConfig is the L2TP/SSTP concentrator (accel-ppp container) for
// routers that cannot run WireGuard (RouterOS 6) and sit behind NAT. Routers
// get a static tunnel IP from Subnet via a chap-secrets file this app writes.
type LegacyVPNConfig struct {
	Subnet      string // e.g. 10.78.0.0/24 — must differ from the WireGuard subnet
	GatewayIP   string // accel-ppp gw-ip-address; RADIUS address tenants point to
	SecretsFile string // chap-secrets path on the shared volume; empty = disabled
	L2TPPort    string
	SSTPPort    string
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
	URL      string // NBI (REST) base URL, e.g. http://genieacs-nbi:7557
	CWMPURL  string // CWMP (TR-069) listener the /cwmp endpoint proxies to, e.g. http://genieacs-cwmp:7547
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

	// Manual bank transfer (used when Provider == "bank_transfer")
	BankName          string
	BankAccountNumber string
	BankAccountHolder string
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
			CWMPURL:  getEnv("CWMP_URL", ""),
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
		LegacyVPN: LegacyVPNConfig{
			Subnet:      getEnv("LEGACY_VPN_SUBNET", "10.78.0.0/24"),
			GatewayIP:   getEnv("LEGACY_VPN_GW", "10.78.0.1"),
			SecretsFile: getEnv("LEGACY_VPN_SECRETS_FILE", ""),
			L2TPPort:    getEnv("LEGACY_VPN_L2TP_PORT", "1701"),
			SSTPPort:    getEnv("LEGACY_VPN_SSTP_PORT", "5443"),
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
			CWMPURL:  getEnv("GENIEACS_CWMP_URL", "http://127.0.0.1:17547"),
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
			Provider:   getEnv("PG_PROVIDER", "bank_transfer"),
			APIKey:     getEnv("PG_API_KEY", ""),
			SecretKey:  getEnv("PG_SECRET_KEY", ""),
			MerchantID: getEnv("PG_MERCHANT_ID", ""),
			Sandbox:    getEnvBool("PG_SANDBOX", true),

			BankName:          getEnv("BANK_TRANSFER_BANK_NAME", "BCA"),
			BankAccountNumber: getEnv("BANK_TRANSFER_ACCOUNT_NUMBER", "1750566584"),
			BankAccountHolder: getEnv("BANK_TRANSFER_ACCOUNT_HOLDER", "Candra Syahputra"),
		},
	}

	return cfg, nil
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
