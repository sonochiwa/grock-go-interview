# Observability

## Три столпа

```
1. Logs   — что произошло (дискретные события)
2. Metrics — сколько (агрегированные числа)
3. Traces — путь запроса (распределённый контекст)
```

## Logging: log/slog (Go 1.21+)

```go
import "log/slog"

// Создание logger
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
    AddSource: true, // добавляет file:line
}))
slog.SetDefault(logger) // глобальный logger

// Использование
slog.Info("user created",
    "user_id", user.ID,
    "email", user.Email,
    "duration_ms", duration.Milliseconds(),
)
// {"time":"2024-01-15T10:30:00Z","level":"INFO","msg":"user created","user_id":"123","email":"a@b.com","duration_ms":45}

slog.Error("failed to process order",
    "err", err,
    "order_id", orderID,
)

// Structured groups
slog.Info("request",
    slog.Group("http",
        slog.String("method", r.Method),
        slog.String("path", r.URL.Path),
        slog.Int("status", status),
    ),
)

// Logger с предустановленными полями (для request scope)
requestLogger := slog.With(
    "request_id", requestID,
    "user_id", userID,
)
requestLogger.Info("processing order")  // автоматически добавит request_id и user_id

// Context-aware logging
ctx = WithLogger(ctx, requestLogger)
// В глубине стека:
LoggerFromContext(ctx).Info("step completed")
```

### Уровни и когда использовать

```
DEBUG — детальная информация для разработки
        cache hit/miss, SQL queries, полные request/response

INFO  — значимые бизнес-события
        user created, order processed, server started

WARN  — потенциальные проблемы (не ошибка, но требует внимания)
        deprecated API used, slow query, retry attempt

ERROR — ошибки, требующие действий
        request failed, connection lost, panic recovered
        НЕ логируй ERROR если обработал ошибку нормально!
```

## Metrics: Prometheus

```go
import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

// Counter — только растёт (requests total, errors total)
var requestsTotal = promauto.NewCounterVec(
    prometheus.CounterOpts{
        Name: "http_requests_total",
        Help: "Total HTTP requests",
    },
    []string{"method", "path", "status"},
)

// Histogram — распределение значений (latency)
var requestDuration = promauto.NewHistogramVec(
    prometheus.HistogramOpts{
        Name:    "http_request_duration_seconds",
        Help:    "HTTP request duration",
        Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5},
    },
    []string{"method", "path"},
)

// Gauge — текущее значение (connections, queue size)
var activeConnections = promauto.NewGauge(
    prometheus.GaugeOpts{
        Name: "active_connections",
        Help: "Number of active connections",
    },
)

// Middleware
func metricsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        sw := &statusWriter{ResponseWriter: w, status: 200}

        next.ServeHTTP(sw, r)

        duration := time.Since(start).Seconds()
        requestsTotal.WithLabelValues(r.Method, r.URL.Path, strconv.Itoa(sw.status)).Inc()
        requestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
    })
}

// Endpoint для scraping
http.Handle("/metrics", promhttp.Handler())
```

### RED Method (для сервисов)

```
Rate    — requests per second          (counter)
Errors  — error rate                   (counter)
Duration — request latency distribution (histogram)

Ключевые метрики:
  http_requests_total{method, path, status}
  http_request_duration_seconds{method, path}
  grpc_server_handled_total{grpc_code, grpc_method}
  grpc_server_handling_seconds{grpc_method}
```

### USE Method (для ресурсов)

```
Utilization — % использования (CPU, memory, disk)
Saturation  — очередь/backlog (goroutines, queue length)
Errors      — ошибки ресурса (disk errors, OOM)
```

## Tracing: OpenTelemetry

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/trace"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Инициализация
func initTracer(ctx context.Context) (*sdktrace.TracerProvider, error) {
    exporter, err := otlptracehttp.New(ctx,
        otlptracehttp.WithEndpoint("tempo:4318"),
        otlptracehttp.WithInsecure(),
    )
    if err != nil {
        return nil, err
    }

    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceNameKey.String("order-service"),
            semconv.ServiceVersionKey.String("1.0.0"),
        )),
        sdktrace.WithSampler(sdktrace.TraceIDRatioBased(0.1)), // 10% sampling
    )

    otel.SetTracerProvider(tp)
    return tp, nil
}

// Использование
var tracer = otel.Tracer("order-service")

func processOrder(ctx context.Context, orderID string) error {
    ctx, span := tracer.Start(ctx, "processOrder",
        trace.WithAttributes(
            attribute.String("order.id", orderID),
        ),
    )
    defer span.End()

    // Вложенный span
    ctx, dbSpan := tracer.Start(ctx, "db.query")
    order, err := repo.Get(ctx, orderID)
    if err != nil {
        dbSpan.RecordError(err)
        dbSpan.SetStatus(codes.Error, err.Error())
    }
    dbSpan.End()

    // Передаём ctx дальше → trace propagation
    return paymentClient.Charge(ctx, order)
}
```

### Trace propagation

```
Request flow:
  [API Gateway] --trace-id--> [Order Service] --trace-id--> [Payment Service]
                                    ↓ trace-id
                              [Database query]

HTTP headers:
  traceparent: 00-<trace-id>-<span-id>-01

gRPC: через metadata (автоматически с otelgrpc)
Kafka: через headers (нужно manual propagation)
```

## Стек observability

```
Logs:    slog → stdout → Fluentd/Vector → Loki/Elasticsearch
Metrics: Prometheus → Grafana
Traces:  OpenTelemetry → Tempo/Jaeger → Grafana

Grafana dashboard:
  - Request rate (RPS)
  - Error rate (%)
  - Latency p50, p95, p99
  - Goroutines, memory, GC
  - DB connections, cache hit rate
```
