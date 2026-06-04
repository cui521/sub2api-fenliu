package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"account-auto-dispatch/internal/api"
	"account-auto-dispatch/internal/core"
)

func main() {
	dataPath := env("AAD_DATA_PATH", filepath.Join("data", "accounts.json"))
	addr := env("AAD_ADDR", "127.0.0.1:18080")

	store, err := newStore(context.Background(), dataPath)
	if err != nil {
		log.Fatal(err)
	}

	runtime := core.NewMemoryRuntimeStore()
	prober, err := newProbeAdapter()
	if err != nil {
		log.Fatal(err)
	}
	scheduler := core.NewScheduler(store, runtime)
	checker := core.NewHealthChecker(store, prober, loadConfig())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if envBool("AAD_AUTO_CHECK_ENABLED", false) {
		go checker.Start(ctx)
	} else {
		log.Printf("automatic account probing disabled; manual probe endpoint remains available")
	}

	server := &http.Server{
		Addr:              addr,
		Handler:           api.NewServer(store, scheduler, checker),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("account-auto-dispatch listening on http://%s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func newStore(ctx context.Context, dataPath string) (core.AccountStore, error) {
	switch env("AAD_STORE", "json") {
	case "sub2_postgres":
		dsn := env("AAD_POSTGRES_DSN", "")
		return core.NewSub2PostgresStore(ctx, dsn)
	default:
		return core.NewJSONAccountStore(dataPath)
	}
}

func newProbeAdapter() (core.ProbeAdapter, error) {
	switch env("AAD_PROBE_MODE", "http") {
	case "sub2_admin":
		return core.NewSub2AdminProbeAdapter(
			env("AAD_SUB2_API_BASE_URL", "http://sub2api:8080/api/v1"),
			env("AAD_SUB2_ADMIN_EMAIL", ""),
			env("AAD_SUB2_ADMIN_PASSWORD", ""),
			10*time.Second,
		)
	default:
		return core.NewHTTPProbeAdapter(10 * time.Second), nil
	}
}

func loadConfig() core.Config {
	config := core.DefaultConfig()
	config.CheckerScanInterval = envDuration("AAD_CHECKER_SCAN_INTERVAL", config.CheckerScanInterval)
	config.ProbeLimitPerScan = envInt("AAD_PROBE_LIMIT_PER_SCAN", config.ProbeLimitPerScan)
	return config
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		log.Printf("invalid %s=%q, using %s", key, value, fallback)
		return fallback
	}
	return parsed
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		log.Printf("invalid %s=%q, using %d", key, value, fallback)
		return fallback
	}
	return parsed
}

func envBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	switch value {
	case "1", "true", "TRUE", "yes", "YES", "on", "ON":
		return true
	case "0", "false", "FALSE", "no", "NO", "off", "OFF":
		return false
	default:
		log.Printf("invalid %s=%q, using %t", key, value, fallback)
		return fallback
	}
}
