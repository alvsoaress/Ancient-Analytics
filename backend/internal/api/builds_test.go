package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alvsoaress/ancient-analytics-backend/internal/builds"
	"github.com/alvsoaress/ancient-analytics-backend/internal/meta"
)

type fakeBuildsService struct {
	result           builds.Result
	err              error
	receivedID       *int
	receivedPosition *meta.Position
}

func (f fakeBuildsService) Hero(ctx context.Context, id int, position meta.Position) (builds.Result, error) {
	if f.receivedID != nil {
		*f.receivedID = id
	}
	if f.receivedPosition != nil {
		*f.receivedPosition = position
	}
	return f.result, f.err
}
func TestHeroBuildsRoute(t *testing.T) {
	id := 0
	position := meta.Position(0)
	service := fakeBuildsService{receivedID: &id, receivedPosition: &position, result: builds.Result{Hero: builds.Hero{HeroID: 62, Name: "Bounty Hunter"}, Builds: []builds.Build{}, Source: "opendota", Scope: "professional_sample"}}
	handler := NewHandler(fakeHeroesClient{}, fakeMetaService{}, fakePlayersService{}, fakeMatchesService{}, service)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/heroes/62/builds?position=support4", nil))
	var body builds.Result
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if response.Code != 200 || response.Header().Get("Content-Type") != "application/json" || id != 62 || position != meta.Support4 || body.Hero.HeroID != 62 || body.Builds == nil {
		t.Fatalf("route: %d %s", response.Code, response.Body.String())
	}
}
func TestHeroBuildsValidatesHTTPInputBeforeService(t *testing.T) {
	tests := []struct{ path, code string }{{"0/builds", "INVALID_HERO_ID"}, {"-1/builds", "INVALID_HERO_ID"}, {"abc/builds", "INVALID_HERO_ID"}, {"1.5/builds", "INVALID_HERO_ID"}, {"2147483648/builds", "INVALID_HERO_ID"}, {"62/builds?position=", "INVALID_POSITION"}, {"62/builds?position=unknown", "INVALID_POSITION"}, {"62/builds?position=mid&position=carry", "INVALID_POSITION"}, {"62/builds?position=%ZZ", "INVALID_POSITION"}}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			id := 0
			handler := NewHandler(fakeHeroesClient{}, fakeMetaService{}, fakePlayersService{}, fakeMatchesService{}, fakeBuildsService{receivedID: &id})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/heroes/"+test.path, nil))
			if response.Code != 400 || id != 0 || !strings.Contains(response.Body.String(), `"code":"`+test.code+`"`) {
				t.Fatalf("validation: %d %s", response.Code, response.Body.String())
			}
		})
	}
}
func TestHeroBuildsErrorResponses(t *testing.T) {
	for _, test := range []struct {
		err    error
		status int
		code   string
	}{{builds.ErrNotFound, 404, "HERO_NOT_FOUND"}, {builds.ErrInvalidHeroID, 400, "INVALID_HERO_ID"}, {builds.ErrInvalidPosition, 400, "INVALID_POSITION"}, {errors.New("internal database token"), 502, "UPSTREAM_UNAVAILABLE"}} {
		response := httptest.NewRecorder()
		NewHandler(fakeHeroesClient{}, fakeMetaService{}, fakePlayersService{}, fakeMatchesService{}, fakeBuildsService{err: test.err}).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/heroes/62/builds", nil))
		if response.Code != test.status || !strings.Contains(response.Body.String(), `"code":"`+test.code+`"`) || strings.Contains(response.Body.String(), "internal database") {
			t.Fatalf("error: %d %s", response.Code, response.Body.String())
		}
	}
}
