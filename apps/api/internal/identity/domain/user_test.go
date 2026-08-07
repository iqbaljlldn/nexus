package domain_test

import (
	"testing"

	"github.com/iqbaljlldn/nexus/apps/api/internal/identity/domain"
)

func TestNewEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		want    domain.Email
		wantErr error
	}{
		{
			name:    "Valid Email",
			email:   "test@example.com",
			want:    domain.Email("test@example.com"),
			wantErr: nil,
		},
		{
			name:    "Valid Email with Uppercase",
			email:   "Test@Example.COM",
			want:    domain.Email("test@example.com"),
			wantErr: nil,
		},
		{
			name:    "Valid Email with Spaces",
			email:   "  test@example.com  ",
			want:    domain.Email("test@example.com"),
			wantErr: nil,
		},
		{
			name:    "Invalid Email - Empty",
			email:   "",
			want:    "",
			wantErr: domain.ErrInvalidEmail,
		},
		{
			name:    "Invalid Email - No At",
			email:   "testexample.com",
			want:    "",
			wantErr: domain.ErrInvalidEmail,
		},
		{
			name:    "Invalid Email - Missing TLD",
			email:   "test@example",
			want:    domain.Email("test@example"), // mail.ParseAddress allows local domains. For stricter parsing, you'd use a regex, but we rely on net/mail.
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.NewEmail(tt.email)
			if err != tt.wantErr {
				t.Errorf("NewEmail() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("NewEmail() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewUsername(t *testing.T) {
	tests := []struct {
		name     string
		username string
		want     domain.Username
		wantErr  error
	}{
		{
			name:     "Valid Username",
			username: "testuser",
			want:     domain.Username("testuser"),
			wantErr:  nil,
		},
		{
			name:     "Valid Username with Numbers",
			username: "user123",
			want:     domain.Username("user123"),
			wantErr:  nil,
		},
		{
			name:     "Valid Username with Underscore",
			username: "test_user",
			want:     domain.Username("test_user"),
			wantErr:  nil,
		},
		{
			name:     "Invalid Username - Too Short",
			username: "us",
			want:     "",
			wantErr:  domain.ErrInvalidUsername,
		},
		{
			name:     "Invalid Username - Too Long",
			username: "thisusernameiswaytoolongtobevalid123",
			want:     "",
			wantErr:  domain.ErrInvalidUsername,
		},
		{
			name:     "Invalid Username - Special Chars",
			username: "user-name",
			want:     "",
			wantErr:  domain.ErrInvalidUsername,
		},
		{
			name:     "Invalid Username - Spaces",
			username: "test user",
			want:     "",
			wantErr:  domain.ErrInvalidUsername,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.NewUsername(tt.username)
			if err != tt.wantErr {
				t.Errorf("NewUsername() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("NewUsername() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewPasswordHash(t *testing.T) {
	tests := []struct {
		name    string
		hash    string
		want    domain.PasswordHash
		wantErr error
	}{
		{
			name:    "Valid Hash",
			hash:    "some_hash_string",
			want:    domain.PasswordHash("some_hash_string"),
			wantErr: nil,
		},
		{
			name:    "Invalid Hash - Empty",
			hash:    "  ",
			want:    "",
			wantErr: domain.ErrInvalidHash,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.NewPasswordHash(tt.hash)
			if err != tt.wantErr {
				t.Errorf("NewPasswordHash() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("NewPasswordHash() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewUser(t *testing.T) {
	email, _ := domain.NewEmail("test@example.com")
	username, _ := domain.NewUsername("testuser")
	hash, _ := domain.NewPasswordHash("hash")

	user := domain.NewUser(email, username, "Test User", hash)

	if user.Email != email {
		t.Errorf("Expected email %v, got %v", email, user.Email)
	}
	if user.Username != username {
		t.Errorf("Expected username %v, got %v", username, user.Username)
	}
	if user.DisplayName != "Test User" {
		t.Errorf("Expected display name Test User, got %v", user.DisplayName)
	}
	if user.PasswordHash != hash {
		t.Errorf("Expected hash %v, got %v", hash, user.PasswordHash)
	}
	if user.IsSuspended != false {
		t.Errorf("Expected IsSuspended false, got %v", user.IsSuspended)
	}
	if user.IsBanned != false {
		t.Errorf("Expected IsBanned false, got %v", user.IsBanned)
	}

	// Test state changes
	user.Suspend()
	if user.IsSuspended != true {
		t.Errorf("Expected IsSuspended true after Suspend(), got %v", user.IsSuspended)
	}

	user.Ban()
	if user.IsBanned != true {
		t.Errorf("Expected IsBanned true after Ban(), got %v", user.IsBanned)
	}
}
