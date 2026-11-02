package profile

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/openpak/ps5-common/accountcore"
	"github.com/openpak/ps5-common/oauth"
)

func setup(t *testing.T) (*Server, string) {
	t.Helper()
	sg, err := oauth.NewSigner([]byte(strings.Repeat("k", 32)))
	if err != nil {
		t.Fatal(err)
	}
	mem := accountcore.NewMemory()
	l, _ := mem.ReserveLink(nil, "oa1", accountcore.Namespace, "222", "i")
	mem.ActivateLink(nil, l.LinkID, "i")
	mem.SetLinkPublicCode(nil, l.LinkID, "Friend_2", "i")
	tok, _ := sg.Sign(oauth.Claims{AccountID: "111", OnlineID: "Me_One", Expires: time.Now().Add(time.Hour).Unix()})
	return New(mem, sg, "https://img.example/a"), tok
}

func get(s *Server, path, tok string) *httptest.ResponseRecorder {
	r := httptest.NewRequest("GET", path, nil)
	if tok != "" {
		r.Header.Set("Authorization", "Bearer "+tok)
	}
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	return w
}

func TestProfileMe(t *testing.T) {
	s, tok := setup(t)
	w := get(s, "/api/userProfile/v1/internal/users/me/profiles", tok)
	var p Profile
	json.Unmarshal(w.Body.Bytes(), &p)
	if w.Code != 200 || p.OnlineID != "Me_One" || !p.IsMe || len(p.Avatars) != 4 || w.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
}

func TestProfileByID(t *testing.T) {
	s, tok := setup(t)
	w := get(s, "/api/userProfile/v1/internal/users/222/profiles", tok)
	var p Profile
	json.Unmarshal(w.Body.Bytes(), &p)
	if w.Code != 200 || p.OnlineID != "Friend_2" || p.IsMe {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	if w := get(s, "/api/userProfile/v1/internal/users/999/profiles", tok); w.Code != 404 {
		t.Fatalf("unknown: %d", w.Code)
	}
	if w := get(s, "/api/userProfile/v1/internal/users/abc/profiles", tok); w.Code != 400 {
		t.Fatalf("bad id: %d", w.Code)
	}
}

func TestAuthRequired(t *testing.T) {
	s, _ := setup(t)
	for _, p := range []string{"/api/userProfile/v1/internal/users/me/profiles", "/userProfile/v1/users/abc/profile2", "/api/v1/devices/accounts/me"} {
		if w := get(s, p, ""); w.Code != 401 {
			t.Errorf("%s: %d", p, w.Code)
		}
	}
}

func TestProfile2(t *testing.T) {
	s, tok := setup(t)
	w := get(s, "/userProfile/v1/users/me_one/profile2", tok)
	var out struct{ Profile map[string]any }
	json.Unmarshal(w.Body.Bytes(), &out)
	if w.Code != 200 || out.Profile["onlineId"] != "Me_One" || out.Profile["npId"] != "Me_One@np" || out.Profile["accountId"] != "111" {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	// other user: unknown until resolved by id
	if w := get(s, "/userProfile/v1/users/Friend_2/profile2", tok); w.Code != 404 {
		t.Fatalf("before cache: %d", w.Code)
	}
	get(s, "/api/userProfile/v1/internal/users/222/profiles", tok)
	if w := get(s, "/userProfile/v1/users/Friend_2/profile2", tok); w.Code != 200 {
		t.Fatalf("after cache: %d", w.Code)
	}
	if w := get(s, "/userProfile/v1/users/a/profile2", tok); w.Code != 400 {
		t.Fatalf("bad online id: %d", w.Code)
	}
}

func TestProfile2Fields(t *testing.T) {
	s, tok := setup(t)
	w := get(s, "/userProfile/v1/users/Me_One/profile2?fields=npId,onlineId,trophySummary(@default,level),bogus", tok)
	var out struct{ Profile map[string]any }
	json.Unmarshal(w.Body.Bytes(), &out)
	if len(out.Profile) != 3 || out.Profile["trophySummary"] == nil {
		t.Fatalf("%s", w.Body)
	}
}

func TestDevices(t *testing.T) {
	s, tok := setup(t)
	w := get(s, "/api/v1/devices/accounts/me?includeFields=device", tok)
	if w.Code != http.StatusOK || strings.TrimSpace(w.Body.String()) != `{"accountDevices":[]}` {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
}

func TestTopLevel(t *testing.T) {
	got := topLevel("a,b(@default,x(y,z)),c")
	if strings.Join(got, "|") != "a|b|c" {
		t.Fatal(got)
	}
}
