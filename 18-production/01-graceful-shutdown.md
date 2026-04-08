# Graceful Shutdown

## Зачем

```
Без graceful shutdown:
  SIGTERM → процесс убит → in-flight запросы потеряны
  → данные не сохранены, транзакции оборваны

С graceful shutdown:
  SIGTERM → перестать принимать новые запросы
         → дождаться завершения текущих
         → закрыть соединения (DB, Redis, Kafka)
         → выйти
```

## Полный пример

```go
func main() {
    // 1. Инициализация зависимостей
    db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
    if err != nil {
        log.Fatal(err)
    }

    redisClient := redis.NewClient(&redis.Options{Addr: os.Getenv("REDIS_URL")})
    kafkaProducer, _ := sarama.NewSyncProducer(brokers, config)

    // 2. HTTP server
    handler := setupRouter(db, redisClient)
    srv := &http.Server{
        Addr:         ":8080",
        Handler:      handler,
        ReadTimeout:  10 * time.Second,
        WriteTimeout: 30 * time.Second,
        IdleTimeout:  60 * time.Second,
    }

    // 3. gRPC server
    grpcSrv := grpc.NewServer(/* interceptors */)
    pb.RegisterMyServiceServer(grpcSrv, newService(db))

    // 4. Health check — NOT_SERVING при shutdown
    healthServer := health.NewServer()
    healthpb.RegisterHealthServer(grpcSrv, healthServer)
    healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

    // 5. Запуск серверов
    go func() {
        log.Println("HTTP server listening on :8080")
        if err := srv.ListenAndServe(); err != http.ErrServerClosed {
            log.Fatal(err)
        }
    }()

    go func() {
        lis, _ := net.Listen("tcp", ":50051")
        log.Println("gRPC server listening on :50051")
        if err := grpcSrv.Serve(lis); err != nil {
            log.Fatal(err)
        }
    }()

    // 6. Ожидание сигнала
    ctx, stop := signal.NotifyContext(context.Background(),
        syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    <-ctx.Done()
    log.Println("shutdown signal received")

    // 7. Graceful shutdown (порядок важен!)

    // 7a. Перестать принимать новые запросы
    healthServer.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)

    // 7b. Дать LB время убрать нас из ротации
    time.Sleep(5 * time.Second)

    // 7c. Shutdown с таймаутом
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // HTTP: дожидается завершения in-flight запросов
    if err := srv.Shutdown(shutdownCtx); err != nil {
        log.Printf("HTTP shutdown error: %v", err)
    }

    // gRPC: дожидается завершения in-flight RPCs
    grpcSrv.GracefulStop()

    // 7d. Закрыть зависимости (после серверов!)
    kafkaProducer.Close()
    redisClient.Close()
    db.Close()

    log.Println("server stopped gracefully")
}
```

## Порядок shutdown

```
1. Пометить NOT_SERVING (health check)
2. Подождать 3-5s (LB drain)
3. Остановить приём новых запросов
4. Дождаться завершения in-flight запросов (с таймаутом)
5. Остановить background workers (cancel context)
6. Flush буферы (logs, metrics, traces)
7. Закрыть downstream connections (DB, Redis, Kafka)
8. Exit

ВАЖНО: зависимости закрывать ПОСЛЕ серверов!
  Иначе: in-flight запрос → DB closed → ошибка
```

## Worker с graceful shutdown

```go
type Worker struct {
    tasks <-chan Task
    done  chan struct{}
    wg    sync.WaitGroup
}

func (w *Worker) Start(ctx context.Context, concurrency int) {
    for i := 0; i < concurrency; i++ {
        w.wg.Add(1)
        go func() {
            defer w.wg.Done()
            for {
                select {
                case <-ctx.Done():
                    return
                case task, ok := <-w.tasks:
                    if !ok {
                        return
                    }
                    w.processTask(ctx, task)
                }
            }
        }()
    }
}

func (w *Worker) Shutdown() {
    w.wg.Wait() // дожидаемся завершения всех горутин
}
```

## Kubernetes: preStop hook

```yaml
# Pod spec
lifecycle:
  preStop:
    exec:
      command: ["sh", "-c", "sleep 5"]
# Kubernetes:
# 1. Убирает pod из Service endpoints
# 2. Выполняет preStop hook (sleep 5)
# 3. Отправляет SIGTERM
# 4. Ждёт terminationGracePeriodSeconds (default 30s)
# 5. SIGKILL если не завершился

terminationGracePeriodSeconds: 60  # для долгих запросов
```
