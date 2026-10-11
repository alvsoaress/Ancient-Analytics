package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type fakeUserStore struct {
	user      User
	lookupErr error
}

func (s *fakeUserStore) UpsertSteamUser(context.Context, string) (User, error) { return s.user, nil }
func (s *fakeUserStore) GetBySteamID(context.Context, string) (User, error) {
	return s.user, s.lookupErr
}
func (s *fakeUserStore) Ping(context.Context) error { return nil }

func TestUserServiceRejectsMissingSessionAndStartsSteamLogin(t *testing.T) {
	h := NewHandler(Config{BaseURL: "https://example.com", SessionSecret: "test-secret", HTTPClient: http.DefaultClient, Store: &fakeUserStore{}})
	me := httptest.NewRecorder()
	h.ServeHTTP(me, httptest.NewRequest(http.MethodGet, "/api/users/me", nil))
	if me.Code != http.StatusUnauthorized {
		t.Fatalf("me=%d", me.Code)
	}
	login := httptest.NewRecorder()
	h.ServeHTTP(login, httptest.NewRequest(http.MethodGet, "/api/users/auth/steam", nil))
	if login.Code != http.StatusFound {
		t.Fatalf("login=%d", login.Code)
	}
}

func TestUserServiceReturnsPersistedUserForValidSession(t *testing.T) {
	stored := User{ID: "user-1", SteamID: "76561198000000000", CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	h := NewHandler(Config{BaseURL: "https://example.com", SessionSecret: "test-secret", Store: &fakeUserStore{user: stored}})

	req := httptest.NewRequest(http.MethodGet, "/api/users/me", nil)
	req.AddCookie(&http.Cookie{Name: "aa_session", Value: sign(session{SteamID: stored.SteamID, ExpiresAt: time.Now().Add(time.Hour).Unix()}, "test-secret")})
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("me=%d body=%s", res.Code, res.Body.String())
	}
	if got := res.Body.String(); got == "" || !contains(got, `"steamId":"76561198000000000"`) {
		t.Fatalf("response does not include persisted user: %s", got)
	}
}
