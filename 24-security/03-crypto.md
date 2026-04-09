# Cryptography в Go

## Hashing

```go
import "crypto/sha256"

// SHA-256 (для checksum, fingerprint — НЕ для паролей!)
hash := sha256.Sum256([]byte("hello"))
fmt.Printf("%x\n", hash)

// HMAC (для подписи сообщений)
import "crypto/hmac"

func signMessage(message, secret []byte) []byte {
    mac := hmac.New(sha256.New, secret)
    mac.Write(message)
    return mac.Sum(nil)
}

func verifySignature(message, secret, signature []byte) bool {
    expected := signMessage(message, secret)
    return hmac.Equal(expected, signature) // constant-time comparison!
}
// ❌ НИКОГДА: bytes.Equal(expected, signature) — timing attack!
```

## Encryption

```go
import "crypto/aes"
import "crypto/cipher"
import "crypto/rand"

// AES-256-GCM (рекомендуется — authenticated encryption)
func Encrypt(plaintext, key []byte) ([]byte, error) {
    block, err := aes.NewCipher(key) // key = 32 bytes для AES-256
    if err != nil {
        return nil, err
    }

    aesGCM, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }

    nonce := make([]byte, aesGCM.NonceSize()) // 12 bytes
    if _, err := rand.Read(nonce); err != nil {
        return nil, err
    }

    // nonce prepended to ciphertext
    return aesGCM.Seal(nonce, nonce, plaintext, nil), nil
}

func Decrypt(ciphertext, key []byte) ([]byte, error) {
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, err
    }

    aesGCM, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }

    nonceSize := aesGCM.NonceSize()
    if len(ciphertext) < nonceSize {
        return nil, errors.New("ciphertext too short")
    }

    nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
    return aesGCM.Open(nil, nonce, ciphertext, nil)
}
```

## Secure Random

```go
import "crypto/rand"

// ✅ Crypto-safe random
func generateToken(n int) (string, error) {
    bytes := make([]byte, n)
    if _, err := rand.Read(bytes); err != nil {
        return "", err
    }
    return base64.URLEncoding.EncodeToString(bytes), nil
}

// ❌ НИКОГДА: math/rand для security
// math/rand предсказуем!
```

## TLS

```go
// Минимальный TLS server
srv := &http.Server{
    Addr: ":443",
    TLSConfig: &tls.Config{
        MinVersion: tls.VersionTLS12,
        CipherSuites: []uint16{
            tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
            tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
            tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
            tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
        },
    },
}
srv.ListenAndServeTLS("cert.pem", "key.pem")

// Let's Encrypt (автоматические сертификаты)
import "golang.org/x/crypto/acme/autocert"

manager := &autocert.Manager{
    Cache:      autocert.DirCache("certs"),
    Prompt:     autocert.AcceptTOS,
    HostPolicy: autocert.HostWhitelist("myapp.com", "www.myapp.com"),
}

srv := &http.Server{
    Addr:      ":443",
    TLSConfig: manager.TLSConfig(),
}
srv.ListenAndServeTLS("", "") // сертификаты из manager
```

## Что использовать

```
Пароли:          bcrypt (простой) или argon2id (memory-hard)
Подпись (HMAC):  crypto/hmac + SHA-256
Шифрование:      AES-256-GCM
Hashing данных:  SHA-256 (checksum, fingerprint)
Random:          crypto/rand (НИКОГДА math/rand для security)
JWT:             RS256 (microservices) или HS256 (single service)
TLS:             ≥ 1.2, prefer 1.3
Comparison:      hmac.Equal() (constant-time)

❌ НЕ использовать:
  MD5, SHA1 — для security (OK для checksum)
  ECB mode — для шифрования
  DES, 3DES — устарели
  Свои crypto алгоритмы — никогда
```
