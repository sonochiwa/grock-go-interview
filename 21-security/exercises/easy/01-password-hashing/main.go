package password_hashing

// TODO: HashPassword — хеширует пароль через bcrypt
func HashPassword(password string) (string, error) {
	return "", nil
}

// TODO: CheckPassword — проверяет пароль против хеша
func CheckPassword(hash, password string) bool {
	return false
}

// TODO: GenerateToken — генерирует crypto-safe random token
// n = количество случайных байт, результат в base64url
func GenerateToken(n int) (string, error) {
	return "", nil
}
