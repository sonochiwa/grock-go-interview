# CDN и Storage

## CDN (Content Delivery Network)

```
Без CDN:
  [User в Токио] ──200ms──→ [Server в Европе]

С CDN:
  [User в Токио] ──10ms──→ [CDN Edge в Токио] ──(cache hit)
                            или
                  ──10ms──→ [CDN Edge] ──200ms──→ [Origin] (cache miss, 1 раз)
```

### Что кэшировать в CDN

- Статика: JS, CSS, изображения, видео
- API ответы (с правильными Cache-Control headers)
- HTML страницы (для SSR/SSG)

### Cache-Control headers

```
Cache-Control: public, max-age=31536000    → CDN + browser, 1 год
Cache-Control: private, max-age=3600       → только browser, 1 час
Cache-Control: no-cache                     → валидируй каждый раз (ETag/If-Modified-Since)
Cache-Control: no-store                     → не кэшируй вообще (PII, secrets)
```

## Object Storage (S3)

```
Характеристики:
  - Бесконечный объём
  - 99.999999999% durability (11 nines)
  - ~100ms latency
  - $0.023/GB/month (S3 Standard)

Когда: файлы, изображен��я, бэкапы, логи, data lake
Когда НЕ: частые обновления, low-latency access, транзакции
```

### Паттерн: Presigned URL для upload

```go
// Сервер генерирует подписанный URL
url, _ := s3client.PresignPutObject(ctx, &s3.PutObjectInput{
    Bucket: aws.String("my-bucket"),
    Key:    aws.String("uploads/avatar.jpg"),
}, presign.WithExpires(15*time.Minute))

// Клиент загружает напрямую в S3 (не через сервер!)
// PUT url → S3
```
