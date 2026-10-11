package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const steamOpenID = "https://steamcommunity.com/openid/login"

type Config struct {
	BaseURL, SessionSecret, SteamAPIKey string
	SecureCookies                       bool
	HTTPClient                          *http.Client
	Store                               UserStore
}
type session struct {
	SteamID   string `json:"steamId"`
	ExpiresAt int64  `json:"expiresAt"`
}
type playerResponse struct {
	Response struct {
		Players []struct {
			SteamID     string `json:"steamid"`
			PersonaName string `json:"personaname"`
			AvatarFull  string `json:"avatarfull"`
			ProfileURL  string `json:"profileurl"`
		} `json:"players"`
	} `json:"response"`
}

func NewHandler(c Config) http.Handler {
	if c.HTTPClient == nil {
		c.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	}
	c.BaseURL = strings.TrimRight(c.BaseURL, "/")
	m := http.NewServeMux()
	m.HandleFunc("GET /livez", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	m.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		if c.BaseURL == "" || c.SessionSecret == "" || c.Store == nil || c.Store.Ping(r.Context()) != nil {
			http.Error(w, "not configured", 503)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
	})
	m.HandleFunc("GET /api/users/auth/steam", func(w http.ResponseWriter, r *http.Request) {
		if c.BaseURL == "" {
			http.Error(w, "not configured", 503)
			return
		}
		q := url.Values{"openid.ns": {"http://specs.openid.net/auth/2.0"}, "openid.mode": {"checkid_setup"}, "openid.return_to": {c.BaseURL + "/api/users/auth/steam/callback"}, "openid.realm": {c.BaseURL}, "openid.identity": {"http://specs.openid.net/auth/2.0/identifier_select"}, "openid.claimed_id": {"http://specs.openid.net/auth/2.0/identifier_select"}}
		http.Redirect(w, r, steamOpenID+"?"+q.Encode(), 302)
	})
	m.HandleFunc("GET /api/users/auth/steam/callback", func(w http.ResponseWriter, r *http.Request) {
		id, err := verify(r, c)
		if err != nil {
			http.Error(w, "steam authentication failed", 401)
			return
		}
		if _, err := c.Store.UpsertSteamUser(r.Context(), id); err != nil {
			http.Error(w, "user service unavailable", http.StatusServiceUnavailable)
			return
		}
		token := sign(session{id, time.Now().Add(24 * time.Hour).Unix()}, c.SessionSecret)
		http.SetCookie(w, &http.Cookie{Name: "aa_session", Value: token, Path: "/", HttpOnly: true, Secure: c.SecureCookies, SameSite: http.SameSiteLaxMode, MaxAge: 86400})
		json.NewEncoder(w).Encode(map[string]string{"steamId": id})
	})
	m.HandleFunc("GET /api/users/me", func(w http.ResponseWriter, r *http.Request) {
		s, err := readSession(r, c.SessionSecret)
		if err != nil {
			http.Error(w, "unauthorized", 401)
			return
		}
		if c.Store == nil {
			http.Error(w, "user service unavailable", http.StatusServiceUnavailable)
			return
		}
		user, err := c.Store.GetBySteamID(r.Context(), s.SteamID)
		if errors.Is(err, ErrUserNotFound) {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "user service unavailable", http.StatusServiceUnavailable)
			return
		}
		if c.SteamAPIKey == "" {
			json.NewEncoder(w).Encode(user)
			return
		}
		u := "https://api.steampowered.com/ISteamUser/GetPlayerSummaries/v0002/?key=" + url.QueryEscape(c.SteamAPIKey) + "&steamids=" + s.SteamID
		resp, err := c.HTTPClient.Get(u)
		if err != nil || resp.StatusCode != 200 {
			http.Error(w, "profile unavailable", 502)
			return
		}
		defer resp.Body.Close()
		var p playerResponse
		json.NewDecoder(resp.Body).Decode(&p)
		json.NewEncoder(w).Encode(struct {
			User    User `json:"user"`
			Players any  `json:"players"`
		}{User: user, Players: p.Response.Players})
	})
	return m
}
func verify(r *http.Request, c Config) (string, error) {
	claimed := r.URL.Query().Get("openid.claimed_id")
	prefix := "https://steamcommunity.com/openid/id/"
	if !strings.HasPrefix(claimed, prefix) {
		return "", errors.New("invalid claimed id")
	}
	if _, e := strconv.ParseUint(strings.TrimPrefix(claimed, prefix), 10, 64); e != nil {
		return "", e
	}
	q := r.URL.Query()
	q.Set("openid.mode", "check_authentication")
	resp, e := c.HTTPClient.PostForm(steamOpenID, q)
	if e != nil {
		return "", e
	}
	defer resp.Body.Close()
	b := make([]byte, 128)
	n, _ := resp.Body.Read(b)
	if !strings.Contains(string(b[:n]), "is_valid:true") {
		return "", errors.New("invalid assertion")
	}
	return strings.TrimPrefix(claimed, prefix), nil
}
func sign(s session, key string) string {
	b, _ := json.Marshal(s)
	p := base64.RawURLEncoding.EncodeToString(b)
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(p))
	return p + "." + base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}
func readSession(r *http.Request, key string) (session, error) {
	c, e := r.Cookie("aa_session")
	if e != nil {
		return session{}, e
	}
	x := strings.Split(c.Value, ".")
	if len(x) != 2 {
		return session{}, errors.New("bad")
	}
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(x[0]))
	sig, _ := base64.RawURLEncoding.DecodeString(x[1])
	if !hmac.Equal(sig, h.Sum(nil)) {
		return session{}, errors.New("bad")
	}
	b, _ := base64.RawURLEncoding.DecodeString(x[0])
	var s session
	if json.Unmarshal(b, &s) != nil || s.ExpiresAt < time.Now().Unix() {
		return session{}, errors.New("expired")
	}
	return s, nil
}
func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	store, err := OpenUserStore(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		panic("database connection failed")
	}
	defer store.Close()
	h := NewHandler(Config{BaseURL: os.Getenv("APP_BASE_URL"), SessionSecret: os.Getenv("SESSION_SECRET"), SteamAPIKey: os.Getenv("STEAM_WEB_API_KEY"), SecureCookies: true, Store: store})
	if err := http.ListenAndServe(":8083", h); err != nil {
		panic(err)
	}
}
