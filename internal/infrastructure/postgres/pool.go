package postgres

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	pgxdecimal "github.com/jackc/pgx-shopspring-decimal"
	"github.com/tu-usuario/inventory-pro/pkg/config"
)

// NewPool crea un pool de conexiones PostgreSQL usando la configuración de la app.
// Si está definido DATABASE_URL, se usa y se fuerza IPv4 cuando sea posible (Docker suele no tener IPv6).
// Si no, se construye el DSN desde DB_HOST, DB_PORT, etc. y se intenta usar IPv4 si aplica.
func NewPool(ctx context.Context, cfg config.DBConfig) (*pgxpool.Pool, error) {
	var dsn string
	if cfg.DatabaseURL != "" {
		dsn = databaseURLWithIPv4(cfg.DatabaseURL)
	} else {
		host := cfg.Host
		if ipv4, err := resolveIPv4(cfg.Host); err == nil {
			host = ipv4
		}
		dsnCfg := cfg
		dsnCfg.Host = host
		dsn = dsnCfg.DSN()
	}

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse DSN: %w", err)
	}

	// Forzar IPv4 en el dial: Docker suele no tener IPv6 y Supabase puede resolver solo AAAA.
	poolConfig.ConnConfig.DialFunc = func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		ipv4, err := resolveIPv4(host)
		if err != nil {
			// Sin IPv4: intentar dial normal (por si el resolver devuelve IPv4 en runtime)
			dialer := &net.Dialer{}
			return dialer.DialContext(ctx, network, addr)
		}
		dialer := &net.Dialer{}
		return dialer.DialContext(ctx, "tcp4", net.JoinHostPort(ipv4, port))
	}
	
	poolConfig.MaxConns = 25
	poolConfig.MinConns = 2
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute
	poolConfig.HealthCheckPeriod = time.Minute

	// Registrar codec para NUMERIC/DECIMAL -> shopspring/decimal (todas las conexiones del pool).
	poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		pgxdecimal.Register(conn.TypeMap())
		return nil
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("crear pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping DB: %w", err)
	}
	return pool, nil
}

// resolveIPv4 resuelve un hostname a su dirección IPv4. Prueba primero el resolver por defecto
// y luego un resolver externo (8.8.8.8) por si el DNS del contenedor solo devuelve IPv6.
func resolveIPv4(host string) (string, error) {
	if ip := net.ParseIP(host); ip != nil {
		if ip.To4() != nil {
			return host, nil
		}
		return "", fmt.Errorf("es IPv6")
	}
	// Intentar con resolver por defecto
	if ip, err := resolveIPv4WithResolver(host, nil); err == nil {
		return ip, nil
	}
	// Dentro de Docker el DNS puede devolver solo IPv6; intentar con DNS público
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{}
			return d.DialContext(ctx, "udp", "8.8.8.8:53")
		},
	}
	return resolveIPv4WithResolver(host, resolver)
}

func resolveIPv4WithResolver(host string, r *net.Resolver) (string, error) {
	var ips []net.IP
	var err error
	if r != nil {
		ips, err = r.LookupIP(context.Background(), "ip4", host)
	} else {
		ips, err = net.LookupIP(host)
	}
	if err != nil {
		return "", err
	}
	for _, ip := range ips {
		if ip.To4() != nil {
			return ip.String(), nil
		}
	}
	return "", fmt.Errorf("no hay IPv4")
}

// databaseURLWithIPv4 reemplaza el hostname de la URL por su IPv4 si existe, para entornos sin IPv6 (ej. Docker).
func databaseURLWithIPv4(databaseURL string) string {
	u, err := url.Parse(databaseURL)
	if err != nil {
		return databaseURL
	}
	hostname := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "5432"
	}
	ipv4, err := resolveIPv4(hostname)
	if err != nil {
		return databaseURL
	}
	u.Host = net.JoinHostPort(ipv4, port)
	return u.String()
}
