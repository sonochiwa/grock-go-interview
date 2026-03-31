package password_hashing

import "testing"

func TestHashAndCheck(t *testing.T) {
	hash, err := HashPassword("secret123")
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}
	if hash == "" {
		t.Fatal("hash is empty")
	}
	if hash == "secret123" {
		t.Fatal("hash should not be plaintext!")
	}
	if !CheckPassword(hash, "secret123") {
		t.Error("CheckPassword should return true for correct password")
	}
	if CheckPassword(hash, "wrong") {
		t.Error("CheckPassword should return false for wrong password")
	}
}

func TestDifferentHashes(t *testing.T) {
	h1, _ := HashPassword("same")
	h2, _ := HashPassword("same")
	if h1 == h2 {
		t.Error("same password should produce different hashes (salt)")
	}
}

func TestGenerateToken(t *testing.T) {
	t1, err := GenerateToken(32)
	if err != nil {
		t.Fatalf("GenerateToken error: %v", err)
	}
	if len(t1) == 0 {
		t.Fatal("token is empty")
	}

	t2, _ := GenerateToken(32)
	if t1 == t2 {
		t.Error("tokens should be unique")
	}
}
