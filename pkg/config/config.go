package config

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

// Config agrupa la configuración de la aplicación (lectura vía Viper desde env y opcionalmente archivo).
type Config struct {
	App   AppConfig
	DB    DBConfig
	JWT   JWTConfig
	HTTP  HTTPConfig
	DIAN  DIANConfig
}

// DIANConfig configuración para factura electrónica DIAN (Colombia).
type DIANConfig struct {
	TechnicalKey string // Clave técnica de la resolución de facturación (obligatoria para CUFE)
	Environment  string // "1" = Producción, "2" = Pruebas (habilitación)
	CertPath     string // Ruta al certificado .pem o .p12 (vacío = no firmar, simulado)
	CertKeyPath  string // Ruta a la llave privada .pem (si CertPath es solo el certificado)
	CertPassword string // Contraseña del .p12 (si CertPath es .p12)
}

// AppConfig configuración general de la aplicación.
type AppConfig struct {
	Env  string // development, staging, production
	Name string
}

// DBConfig configuración de PostgreSQL.
// Si DatabaseURL no está vacío, se usa como connection string completo (ej. DATABASE_URL de Supabase).
type DBConfig struct {
	DatabaseURL string // Opcional: postgresql://user:password@host:port/dbname?sslmode=require
	Host        string
	Port        int
	User        string
	Password    string
	DBName      string
	SSLMode     string
}

// ConnectionString devuelve el DSN a usar: DATABASE_URL si está definido, si no el construido con DSN().
func (c DBConfig) ConnectionString() string {
	if c.DatabaseURL != "" {
		return c.DatabaseURL
	}
	return c.DSN()
}

// DSN devuelve el connection string para PostgreSQL con URL encoding para caracteres especiales.
func (c DBConfig) DSN() string {
	// Usar url.UserPassword para manejar correctamente caracteres especiales en la contraseña
	userInfo := url.UserPassword(c.User, c.Password)
	
	u := &url.URL{
		Scheme:   "postgres",
		User:     userInfo,
		Host:     fmt.Sprintf("%s:%d", c.Host, c.Port),
		Path:     "/" + c.DBName,
		RawQuery: fmt.Sprintf("sslmode=%s", c.SSLMode),
	}
	
	return u.String()
}

// JWTConfig configuración de JWT.
type JWTConfig struct {
	Secret     string
	Expiration int // minutos
	Issuer     string
}

// HTTPConfig configuración del servidor HTTP.
type HTTPConfig struct {
	Host string
	Port int
}

// Addr devuelve la dirección de escucha (host:port).
func (c HTTPConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// Load lee la configuración desde variables de entorno (y opcionalmente desde archivo).
// Las env vars tienen prioridad. Nombres esperados: APP_ENV, DB_HOST, DB_PORT, JWT_SECRET, etc.
func Load() (*Config, error) {
	v := viper.New()

	// Opcional: archivo de configuración (.env o config.env)
	v.SetConfigName(".env")
	v.SetConfigType("env")
	v.AddConfigPath(".")
	_ = v.ReadInConfig() // ignoramos error si no existe
	
	// También intenta config.env
	v.SetConfigName("config")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	_ = v.ReadInConfig() // ignoramos error si no existe

	// Bind de variables de entorno (Viper las lee automáticamente si AutomaticEnv está activo)
	v.AutomaticEnv()
	// Permite usar APP_ENV, DB_HOST, JWT_SECRET, etc.
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Valores por defecto
	setDefaults(v)

	cfg := &Config{
		App: AppConfig{
			Env:  getString(v, "APP_ENV", "development"),
			Name: getString(v, "APP_NAME", "inventory-pro"),
		},
		DB: DBConfig{
			DatabaseURL: getString(v, "DATABASE_URL", ""),
			Host:        getString(v, "DB_HOST", "localhost"),
			Port:        getInt(v, "DB_PORT", 5432),
			User:        getString(v, "DB_USER", "postgres"),
			Password:    getString(v, "DB_PASSWORD", ""),
			DBName:      getString(v, "DB_NAME", "inventory_pro"),
			SSLMode:     getString(v, "DB_SSLMODE", "disable"),
		},
		JWT: JWTConfig{
			Secret:     getString(v, "JWT_SECRET", ""),
			Expiration: getInt(v, "JWT_EXPIRATION_MINUTES", 60),
			Issuer:     getString(v, "JWT_ISSUER", "inventory-pro"),
		},
		HTTP: HTTPConfig{
			Host: getString(v, "HTTP_HOST", "0.0.0.0"),
			Port: getInt(v, "HTTP_PORT", 8080),
		},
		DIAN: DIANConfig{
			TechnicalKey:  getString(v, "DIAN_TECHNICAL_KEY", ""),
			Environment:  getString(v, "DIAN_ENVIRONMENT", "2"),
			CertPath:     getString(v, "DIAN_CERT_PATH", ""),
			CertKeyPath:  getString(v, "DIAN_CERT_KEY_PATH", ""),
			CertPassword: getString(v, "DIAN_CERT_PASSWORD", ""),
		},
	}

	return cfg, nil
}

func setDefaults(v *viper.Viper) {
	// Ya aplicados en la construcción del struct; aquí se pueden centralizar si se prefiere
	_ = v
}

func getString(v *viper.Viper, key, def string) string {
	if v.IsSet(key) {
		return v.GetString(key)
	}
	return def
}

func getInt(v *viper.Viper, key string, def int) int {
	if v.IsSet(key) {
		switch v.Get(key).(type) {
		case int:
			return v.GetInt(key)
		case string:
			n, _ := strconv.Atoi(v.GetString(key))
			return n
		default:
			return v.GetInt(key)
		}
	}
	return def
}
