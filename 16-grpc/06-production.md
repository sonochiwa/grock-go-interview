# gRPC: Production

## TLS

```go
// Server с TLS
creds, _ := credentials.NewServerTLSFromFile("server.crt", "server.key")
server := grpc.NewServer(grpc.Creds(creds))

// mTLS (mutual TLS — клиент тоже предоставляет сертификат)
cert, _ := tls.LoadX509KeyPair("server.crt", "server.key")
certPool := x509.NewCertPool()
caCert, _ := os.ReadFile("ca.crt")
certPool.AppendCertsFromPEM(caCert)

tlsConfig := &tls.Config{
    Certificates: []tls.Certificate{cert},
    ClientAuth:   tls.RequireAndVerifyClientCert,
    ClientCAs:    certPool,
}
server := grpc.NewServer(grpc.Creds(credentials.NewTLS(tlsConfig)))

// Client с TLS
creds, _ := credentials.NewClientTLSFromFile("ca.crt", "server.example.com")
conn, _ := grpc.NewClient("server:50051", grpc.WithTransportCredentials(creds))
```

## Health Check

```go
import "google.golang.org/grpc/health"
import healthpb "google.golang.org/grpc/health/grpc_health_v1"

// Регистрация health server
healthServer := health.NewServer()
healthpb.RegisterHealthServer(server, healthServer)

// Установка статуса для конкретного сервиса
healthServer.SetServingStatus("user.v1.UserService", healthpb.HealthCheckResponse_SERVING)

// При shutdown
healthServer.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)

// Kubernetes grpc health check (с grpc-health-probe)
// livenessProbe:
//   exec:
//     command: ["/bin/grpc_health_probe", "-addr=:50051"]
//   periodSeconds: 10

// Или нативно в K8s 1.24+:
// livenessProbe:
//   grpc:
//     port: 50051
```

## Reflection (для дебага)

```go
import "google.golang.org/grpc/reflection"

server := grpc.NewServer()
pb.RegisterUserServiceServer(server, &userServer{})
reflection.Register(server) // включить reflection

// Теперь можно использовать grpcurl:
// grpcurl -plaintext localhost:50051 list
// grpcurl -plaintext localhost:50051 describe user.v1.UserService
// grpcurl -plaintext -d '{"id": "123"}' localhost:50051 user.v1.UserService/GetUser

// ВАЖНО: отключить в production (security risk)!
```

## Load Balancing

```
Client-side LB (gRPC native):
  [Client] → DNS → [Server 1, Server 2, Server 3]
  Client сам выбирает к какому серверу обращаться

  Стратегии:
    - pick_first: первый здоровый (default)
    - round_robin: по кругу

  conn, _ := grpc.NewClient("dns:///my-service:50051",
      grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"round_robin":{}}]}`),
  )

Proxy-based LB (Envoy, Nginx):
  [Client] → [Envoy Proxy] → [Server 1, Server 2, Server 3]
  Proxy маршрутизирует запросы

  ВАЖНО: HTTP/2 multiplexing проблема!
  HTTP/2 = одно TCP соединение → LB (L4) не балансирует отдельные RPC
  Решение: L7 LB (Envoy понимает gRPC) или client-side LB

Look-aside LB (xDS, Kubernetes):
  [Client] → [LB service (etcd/Consul)] → получает endpoint list
  Client-side LB на основе service discovery
```

## gRPC-Gateway (REST + gRPC)

```protobuf
// Proto с HTTP аннотациями
import "google/api/annotations.proto";

service UserService {
  rpc GetUser(GetUserRequest) returns (GetUserResponse) {
    option (google.api.http) = {
      get: "/api/v1/users/{id}"
    };
  }
  rpc CreateUser(CreateUserRequest) returns (CreateUserResponse) {
    option (google.api.http) = {
      post: "/api/v1/users"
      body: "*"
    };
  }
}
```

```go
// Reverse proxy: REST → gRPC
mux := runtime.NewServeMux()
opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
err := pb.RegisterUserServiceHandlerFromEndpoint(ctx, mux, "localhost:50051", opts)

// HTTP server для REST
http.ListenAndServe(":8080", mux)

// Теперь работают оба:
// gRPC:  grpcurl localhost:50051 user.v1.UserService/GetUser
// REST:  curl localhost:8080/api/v1/users/123
```

## Observability

```go
// OpenTelemetry + gRPC
import "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"

// Server
server := grpc.NewServer(
    grpc.StatsHandler(otelgrpc.NewServerHandler()),
)

// Client
conn, _ := grpc.NewClient(addr,
    grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
)

// Prometheus metrics
import grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"

srvMetrics := grpcprom.NewServerMetrics()
server := grpc.NewServer(
    grpc.ChainUnaryInterceptor(srvMetrics.UnaryServerInterceptor()),
    grpc.ChainStreamInterceptor(srvMetrics.StreamServerInterceptor()),
)
srvMetrics.InitializeMetrics(server)

// Метрики:
// grpc_server_handled_total{grpc_code, grpc_method, grpc_service}
// grpc_server_handling_seconds{...}
// grpc_server_msg_received_total{...}
```

## Полный production server

```go
func main() {
    // Interceptors
    server := grpc.NewServer(
        grpc.Creds(loadTLS()),
        grpc.ChainUnaryInterceptor(
            recovery.UnaryServerInterceptor(),
            otelgrpc.UnaryServerInterceptor(),
            srvMetrics.UnaryServerInterceptor(),
            logging.UnaryServerInterceptor(logger),
            auth.UnaryServerInterceptor(authFunc),
        ),
        grpc.ChainStreamInterceptor(
            recovery.StreamServerInterceptor(),
            otelgrpc.StreamServerInterceptor(),
            srvMetrics.StreamServerInterceptor(),
            logging.StreamServerInterceptor(logger),
        ),
        grpc.KeepaliveParams(keepalive.ServerParameters{
            MaxConnectionIdle: 5 * time.Minute,
            Time:              1 * time.Minute,
            Timeout:           20 * time.Second,
        }),
        grpc.MaxRecvMsgSize(4 * 1024 * 1024), // 4MB max message
    )

    // Register services
    pb.RegisterUserServiceServer(server, newUserServer())
    healthServer := health.NewServer()
    healthpb.RegisterHealthServer(server, healthServer)
    reflection.Register(server) // убрать в production

    // Graceful shutdown
    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    lis, _ := net.Listen("tcp", ":50051")

    go func() {
        log.Println("gRPC server listening on :50051")
        if err := server.Serve(lis); err != nil {
            log.Fatal(err)
        }
    }()

    <-ctx.Done()
    healthServer.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
    server.GracefulStop()
    log.Println("server stopped")
}
```

## Частые вопросы

**Q: gRPC vs REST для public API?**
A: REST для public API (браузеры, curl, любой клиент). gRPC для internal microservices (performance, type safety). gRPC-Gateway если нужно оба.

**Q: Как дебажить gRPC?**
A: grpcurl (CLI), Postman (поддерживает gRPC), Evans (REPL), reflection API. В production: distributed tracing (Jaeger/Tempo).

**Q: Connection pooling в gRPC?**
A: Одно gRPC соединение мультиплексирует запросы через HTTP/2. Обычно достаточно одного соединения. Для высокой нагрузки — несколько `grpc.ClientConn`.
