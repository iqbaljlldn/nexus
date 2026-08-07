package passwordhash

import (
	"strings"
	"testing"
)

func TestHashAndVerify(t *testing.T) {
	password := "supersecurepassword123"

	hashed, err := Hash(password)
	if err != nil {
		t.Fatalf("Hash failed: %v", err)
	}

	if !strings.HasPrefix(hashed, "$argon2id$v=19$m=47104,t=3,p=2$") {
		t.Errorf("Hash output does not match expected format, got: %s", hashed)
	}

	// Test successful verification
	verified, err := Verify(password, hashed)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	if !verified {
		t.Errorf("Password verification failed for correct password")
	}

	// Test failed verification with wrong password
	wrongVerified, err := Verify("wrongpassword", hashed)
	if err != nil {
		t.Fatalf("Verify failed with error on wrong password: %v", err)
	}

	if wrongVerified {
		t.Errorf("Password verification succeeded for incorrect password")
	}

	// Test invalid hash format
	_, err = Verify(password, "invalidhash")
	if err != ErrInvalidHash {
		t.Errorf("Expected ErrInvalidHash for invalid format, got %v", err)
	}
}
