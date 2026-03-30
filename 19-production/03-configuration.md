# Configuration

## Приоритет (12-Factor App)

```
1. Defaults в коде (самый низкий приоритет)
2. Config file (YAML/TOML)
3. Environment variables (рекомендуется для production)
4. CLI flags (самый высокий приоритет)
```

## Struct-based config

```go
type Config struct {
    Server   ServerConfig   `yaml:"server"`
    Database DatabaseConfig `yaml:"database"`
    Redis    RedisConfig    `yaml:"redis"`
    Kafka    KafkaConfig    `yaml:"kafka"`
    Log      LogConfig      `yaml:"log"`
}

type ServerConfig struct {
    HTTPPort     int           `yaml:"http_port" env:"HTTP_PORT" default:"8080"`
    GRPCPort     int           `yaml:"grpc_port" env:"GRPC_PORT" default:"50051"`
    ReadTimeout  time.Duration `yaml:"read_timeout" default:"10s"`
    WriteTimeout time.Duration `yaml:"write_timeout" default:"30s"`
}

type DatabaseConfig struct {
    DSN             string        `yaml:"dsn" env:"DATABASE_URL" required:"true"`
    MaxOpenConns    int           `yaml:"max_open_conns" default:"25"`
    MaxIdleConns    int           `yaml:"max_idle_conns" default:"10"`
    ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" default:"5m"`
}
```

## Загрузка с env-override

```go
// Простой подход: env > config file > defaults
func LoadConfig(path string) (*Config, error) {
    cfg := &Config{
        Server: ServerConfig{
            HTTPPort:     8080,
            ReadTimeout:  10 * time.Second,
            WriteTimeout: 30 * time.Second,
        },
        Database: DatabaseConfig{
            MaxOpenConns: 25,
            MaxIdleConns: 10,
        },
    }

    // Config file
    if path != "" {
        data, err := os.ReadFile(path)
        if err != nil {
            return nil, err
        }
        if err := yaml.Unmarshal(data, cfg); err != nil {
            return nil, err
        }
    }

    // Env overrides
    if v := os.Getenv("DATABASE_URL"); v != "" {
        cfg.Database.DSN = v
    }
    if v := os.Getenv("HTTP_PORT"); v != "" {
        port, _ := strconv.Atoi(v)
        cfg.Server.HTTPPort = port
    }

    return cfg, nil
}

// Или с библиотекой: github.com/caarlos0/env/v11
func LoadConfig() (*Config, error) {
    cfg := &Config{}
    if err := env.Parse(cfg); err != nil {
        return nil, err
    }
    return cfg, nil
}
```

## Validation

```go
func (c *Config) Validate() error {
    if c.Database.DSN == "" {
        return errors.New("database DSN is required")
    }
    if c.Server.HTTPPort < 1 || c.Server.HTTPPort > 65535 {
        return fmt.Errorf("invalid HTTP port: %d", c.Server.HTTPPort)
    }
    if c.Database.MaxOpenConns < c.Database.MaxIdleConns {
        return errors.New("max_open_conns must be >= max_idle_conns")
    }
    return nil
}
```

## Secrets

```
НИКОГДА в коде или config files!

Где хранить:
  - Environment variables (простейший)
  - Kubernetes Secrets (base64, не шифрование!)
  - Vault (HashiCorp) — production standard
  - AWS Secrets Manager / GCP Secret Manager
  - SOPS (encrypted YAML)

Go:
  dbPassword := os.Getenv("DB_PASSWORD")
  // Или Vault client:
  secret, _ := vaultClient.Logical().Read("secret/data/myapp")
  dbPassword := secret.Data["password"].(string)
```

## Feature Flags

```go
// Простой подход: config + atomic
type Features struct {
    NewCheckout atomic.Bool
    DarkMode    atomic.Bool
}

var features Features

// Из config/env
features.NewCheckout.Store(os.Getenv("FF_NEW_CHECKOUT") == "true")

// Использование
if features.NewCheckout.Load() {
    return newCheckoutFlow(ctx, order)
}
return legacyCheckoutFlow(ctx, order)

// Production: используй специализированный сервис
// - LaunchDarkly, Unleash, Flipt
// - Gradual rollout (1% → 10% → 50% → 100%)
// - A/B testing
// - Per-user / per-segment targeting
```
