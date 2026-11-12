package service

import (
	"testing"
	"time"

	"helpdesk/backend/internal/domain"
)

func TestIssuedTokenParsesAndRejectsTampering(t *testing.T) {
	s := New(nil, "a secret that is long enough for an HMAC signing key", time.Hour)
	token, err := s.issueToken(domain.User{ID: 42, Role: domain.Agent})
	if err != nil {
		t.Fatal(err)
	}
	id, err := ParseToken(token, s.jwtSecret)
	if err != nil || id != 42 {
		t.Fatalf("ParseToken() = (%d, %v), want (42, nil)", id, err)
	}
	if _, err := ParseToken(token+"x", s.jwtSecret); err == nil {
		t.Fatal("tampered token was accepted")
	}
	if _, err := ParseToken(token, []byte("wrong signing key")); err == nil {
		t.Fatal("token signed with a different secret was accepted")
	}
}
