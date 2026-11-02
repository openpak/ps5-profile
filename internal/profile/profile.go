// Package profile serves the user-profile slice of the PSN web API: the
// /api/userProfile/ paths of m.np.playstation.com, the dms.api.playstation.com
// device list and the legacy profile2 endpoint of us-prof.np.community.playstation.net.
//
// Account model: the account core link in namespace "ps5" has SubjectID = the
// PSN account id (64-bit decimal) and PublicCode = the online id.
package profile

import (
	"context"
	"net/http"
	"strings"
	"sync"

	"github.com/openpak/ps5-common/accountcore"
	"github.com/openpak/ps5-common/httpx"
	"github.com/openpak/ps5-common/ids"
	"github.com/openpak/ps5-common/oauth"
)

// Error codes in the PSN error body (values not verified against Sony).
const (
	codeBadRequest = 2240001
	codeNotFound   = 2240002
	codeInternal   = 2240003
)

// Avatar is one avatar size.
type Avatar struct {
	Size string `json:"size"`
	URL  string `json:"url"`
}

// Profile is the answer of /api/userProfile/v1/internal/users/{accountId}/profiles.
type Profile struct {
	OnlineID             string   `json:"onlineId"`
	AboutMe              string   `json:"aboutMe"`
	Avatars              []Avatar `json:"avatars"`
	Languages            []string `json:"languages"`
	IsPlus               bool     `json:"isPlus"`
	IsOfficiallyVerified bool     `json:"isOfficiallyVerified"`
	IsMe                 bool     `json:"isMe"`
}

// Server holds the dependencies of the handlers.
type Server struct {
	Core      accountcore.Core
	Verifier  oauth.Verifier
	AvatarURL string // base URL of avatar images, e.g. https://image.api.playstation.com/avatar; may be empty

	mu     sync.Mutex
	byName map[string]string // lower-case online id -> account id, learnt from lookups and tokens
}

// New returns a Server.
func New(core accountcore.Core, v oauth.Verifier, avatarURL string) *Server {
	return &Server{Core: core, Verifier: v, AvatarURL: strings.TrimRight(avatarURL, "/"), byName: map[string]string{}}
}

// Handler returns the full mux: health, the three hosts' paths.
func (s *Server) Handler() http.Handler {
	m := httpx.Mux()
	auth := oauth.Require(s.Verifier)
	m.Handle("GET /api/userProfile/v1/internal/users/me/profiles", auth(http.HandlerFunc(s.profile)))
	m.Handle("GET /api/userProfile/v1/internal/users/{accountId}/profiles", auth(http.HandlerFunc(s.profile)))
	m.Handle("GET /userProfile/v1/users/{onlineId}/profile2", auth(http.HandlerFunc(s.profile2)))
	m.Handle("GET /api/v1/devices/accounts/me", auth(http.HandlerFunc(s.devices)))
	return m
}

type user struct{ accountID, onlineID string }

func (s *Server) remember(u user) {
	s.mu.Lock()
	s.byName[ids.NormalizeOnlineID(u.onlineID)] = u.accountID
	s.mu.Unlock()
}

// byAccount resolves an account id; ok false means unknown.
func (s *Server) byAccount(ctx context.Context, accountID string, c oauth.Claims) (user, bool, error) {
	if accountID == c.AccountID && c.OnlineID != "" {
		u := user{accountID, c.OnlineID}
		s.remember(u)
		return u, true, nil
	}
	l, err := s.Core.GetLinkBySubject(ctx, accountcore.Namespace, accountID)
	if err != nil {
		if accountcore.IsNotFound(err) {
			return user{}, false, nil
		}
		return user{}, false, err
	}
	if l.State != accountcore.StateActive || l.PublicCode == "" {
		return user{}, false, nil
	}
	u := user{accountID, l.PublicCode}
	s.remember(u)
	return u, true, nil
}

func (s *Server) writeLookupError(w http.ResponseWriter, found bool, err error) bool {
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, codeInternal, "Internal error")
		return true
	}
	if !found {
		httpx.WriteError(w, http.StatusNotFound, codeNotFound, "Resource not found")
		return true
	}
	return false
}

func (s *Server) avatars(accountID string) []Avatar {
	if s.AvatarURL == "" {
		return []Avatar{}
	}
	out := []Avatar{}
	for _, sz := range []string{"xl", "l", "m", "s"} {
		out = append(out, Avatar{Size: sz, URL: s.AvatarURL + "/" + accountID + "_" + sz + ".png"})
	}
	return out
}

func (s *Server) profile(w http.ResponseWriter, r *http.Request) {
	c, _ := oauth.FromContext(r.Context())
	id := r.PathValue("accountId")
	if id == "" || id == "me" {
		id = c.AccountID
	} else if _, err := ids.ParseAccountID(id); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, codeBadRequest, "Invalid account id")
		return
	}
	u, found, err := s.byAccount(r.Context(), id, c)
	if s.writeLookupError(w, found, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, Profile{
		OnlineID: u.onlineID, AboutMe: "", Avatars: s.avatars(id), Languages: []string{"en-US"},
		IsMe: id == c.AccountID,
	})
}

// EarnedTrophies is the per-grade count in trophySummary.
type EarnedTrophies struct {
	Bronze   int `json:"bronze"`
	Silver   int `json:"silver"`
	Gold     int `json:"gold"`
	Platinum int `json:"platinum"`
}

// TrophySummary is the profile2 trophy block. ps5-trophy owns real values; here it is level 1, zero earned.
type TrophySummary struct {
	Level          int            `json:"level"`
	Progress       int            `json:"progress"`
	EarnedTrophies EarnedTrophies `json:"earnedTrophies"`
}

// AvatarURL is one profile2 avatar entry.
type AvatarURL struct {
	Size      string `json:"size"`
	AvatarURL string `json:"avatarUrl"`
}

func (s *Server) profile2(w http.ResponseWriter, r *http.Request) {
	c, _ := oauth.FromContext(r.Context())
	name := r.PathValue("onlineId")
	if ids.ValidateOnlineID(name) != nil {
		httpx.WriteError(w, http.StatusBadRequest, codeBadRequest, "Invalid online id")
		return
	}
	key := ids.NormalizeOnlineID(name)
	var accountID string
	if ids.NormalizeOnlineID(c.OnlineID) == key {
		accountID = c.AccountID
	} else {
		s.mu.Lock()
		accountID = s.byName[key]
		s.mu.Unlock()
	}
	if accountID == "" {
		httpx.WriteError(w, http.StatusNotFound, codeNotFound, "Resource not found")
		return
	}
	u, found, err := s.byAccount(r.Context(), accountID, c)
	if s.writeLookupError(w, found, err) {
		return
	}
	av := []AvatarURL{}
	for _, a := range s.avatars(accountID) {
		av = append(av, AvatarURL{Size: a.Size, AvatarURL: a.URL})
	}
	full := map[string]any{
		"onlineId":             u.onlineID,
		"accountId":            accountID,
		"npId":                 ids.NpID{OnlineID: u.onlineID, Domain: ids.DefaultDomain}.String(),
		"avatarUrls":           av,
		"plus":                 0,
		"aboutMe":              "",
		"languagesUsed":        []string{"en"},
		"trophySummary":        TrophySummary{Level: 1},
		"isOfficiallyVerified": false,
	}
	out := full
	if f := r.URL.Query().Get("fields"); f != "" {
		out = map[string]any{}
		for _, name := range topLevel(f) {
			if v, ok := full[name]; ok {
				out[name] = v
			}
		}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"profile": out})
}

// topLevel returns the field names of a profile2 fields list, dropping the
// parenthesised sub-selections: "a,b(@default,x),c" -> a, b, c.
func topLevel(f string) []string {
	var out []string
	depth, start := 0, 0
	flush := func(end int) {
		n := f[start:end]
		if i := strings.IndexByte(n, '('); i >= 0 {
			n = n[:i]
		}
		if n = strings.TrimSpace(n); n != "" {
			out = append(out, n)
		}
	}
	for i := 0; i < len(f); i++ {
		switch f[i] {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				flush(i)
				start = i + 1
			}
		}
	}
	flush(len(f))
	return out
}

// devices answers the dms.api.playstation.com device list. The real body is
// not documented in the sources read; OpenPak keeps no device registry yet, so
// the list is empty.
func (s *Server) devices(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"accountDevices": []any{}})
}
