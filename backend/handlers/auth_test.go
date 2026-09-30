package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"ping/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

// fakeProvider stands in for Google or Discord: it hands out a token for any
// code whose PKCE verifier checks out, and says the token belongs to whoever
// the code was issued for.
type fakeProvider struct {
	srv   *httptest.Server
	codes map[string]string // code -> subject
	// challenge is the PKCE challenge the last login started with.
	challenge string
}

func newFakeProvider(t *testing.T) *fakeProvider {
	t.Helper()
	f := &fakeProvider{codes: map[string]string{}}
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		subject, ok := f.codes[r.Form.Get("code")]
		if !ok || oauth2.S256ChallengeFromVerifier(r.Form.Get("code_verifier")) != f.challenge {
			http.Error(w, `{"error":"invalid_grant"}`, http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"access_token": "at-" + subject, "token_type": "Bearer"})
	})
	mux.HandleFunc("/me", func(w http.ResponseWriter, r *http.Request) {
		subject := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer at-")
		json.NewEncoder(w).Encode(map[string]string{"sub": subject, "name": "Name of " + subject})
	})
	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeProvider) provider() *Provider {
	return &Provider{
		Name: "fake",
		OAuth: oauth2.Config{
			ClientID: "id", ClientSecret: "secret",
			Endpoint: oauth2.Endpoint{AuthURL: f.srv.URL + "/authorize", TokenURL: f.srv.URL + "/token"},
		},
		Profile: f.srv.URL + "/me",
		Parse: func(body []byte) (models.Identity, error) {
			var p struct{ Sub, Name string }
			err := json.Unmarshal(body, &p)
			return models.Identity{Provider: "fake", Subject: p.Sub, Name: p.Name}, err
		},
	}
}

func authServer(t *testing.T) (*Handler, *gin.Engine, *fakeProvider) {
	t.Helper()
	h, r := newTestHandler(t)
	f := newFakeProvider(t)
	h.UseAuth(&Auth{BaseURL: "http://ping.test", Providers: map[string]*Provider{"fake": f.provider()}})
	return h, r, f
}

// browser is a cookie jar for requests straight to the router.
type browser map[string]string

func (b browser) do(r *gin.Engine, req *http.Request) *httptest.ResponseRecorder {
	for name, value := range b {
		req.AddCookie(&http.Cookie{Name: name, Value: value})
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	for _, c := range w.Result().Cookies() {
		if c.MaxAge < 0 {
			delete(b, c.Name)
		} else {
			b[c.Name] = c.Value
		}
	}
	return w
}

// login goes through the whole flow as subject, returning where it ended.
func (b browser) login(t *testing.T, r *gin.Engine, f *fakeProvider, subject, returnTo string) string {
	t.Helper()
	w := b.do(r, httptest.NewRequest(http.MethodGet, "/api/auth/fake/login?return="+url.QueryEscape(returnTo), nil))
	if w.Code != http.StatusFound {
		t.Fatalf("login: %d %s", w.Code, w.Body)
	}
	out, _ := url.Parse(w.Header().Get("Location"))
	q := out.Query()
	f.challenge = q.Get("code_challenge")
	code := "code-" + subject
	f.codes[code] = subject
	w = b.do(r, httptest.NewRequest(http.MethodGet,
		"/api/auth/fake/callback?code="+code+"&state="+url.QueryEscape(q.Get("state")), nil))
	if w.Code != http.StatusFound {
		t.Fatalf("callback: %d %s", w.Code, w.Body)
	}
	return w.Header().Get("Location")
}

type authSession struct {
	User *struct {
		ID          string   `json:"id"`
		Guest       bool     `json:"guest"`
		DisplayName string   `json:"display_name"`
		Providers   []string `json:"providers"`
	} `json:"user"`
	Providers []string `json:"providers"`
}

func (b browser) session(t *testing.T, r *gin.Engine) authSession {
	t.Helper()
	w := b.do(r, httptest.NewRequest(http.MethodGet, "/api/auth/session", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("session: %d %s", w.Code, w.Body)
	}
	var out authSession
	json.Unmarshal(w.Body.Bytes(), &out)
	return out
}

func (b browser) join(t *testing.T, r *gin.Engine, roomKey, player string) {
	t.Helper()
	w := joinFromBrowser(t, r, roomKey, player, b[sessionCookie])
	if c := sessionCookieSet(w); c != nil {
		b[sessionCookie] = c.Value
	}
}

func TestLoginStartsWithStateAndPKCE(t *testing.T) {
	_, r, _ := authServer(t)
	b := browser{}
	w := b.do(r, httptest.NewRequest(http.MethodGet, "/api/auth/fake/login?return=/hotpot", nil))
	if w.Code != http.StatusFound {
		t.Fatalf("login: %d", w.Code)
	}
	out, _ := url.Parse(w.Header().Get("Location"))
	q := out.Query()
	if q.Get("state") == "" || q.Get("code_challenge") == "" || q.Get("code_challenge_method") != "S256" {
		t.Fatalf("login redirect lacks state or PKCE: %s", out)
	}
	if q.Get("redirect_uri") != "http://ping.test/api/auth/fake/callback" {
		t.Fatalf("redirect_uri = %q", q.Get("redirect_uri"))
	}
	if b[flowCookie] == "" {
		t.Fatal("no flow cookie")
	}
	if w := b.do(r, httptest.NewRequest(http.MethodGet, "/api/auth/nope/login", nil)); w.Code != http.StatusNotFound {
		t.Fatalf("unknown provider: %d", w.Code)
	}
}

// Logging in upgrades the guest in place: same account, same seats, now
// signed in. The session is replaced, so the cookie from before is dead.
func TestLoggingInUpgradesTheGuest(t *testing.T) {
	h, r, f := authServer(t)
	room := postCreate(t, h, r, "table")
	b := browser{}
	b.join(t, r, room.RoomKey, "ana")
	guest := b.session(t, r).User
	before := b[sessionCookie]

	if to := b.login(t, r, f, "sub-1", "/exploding-kitchen?room="+room.RoomKey); to != "/exploding-kitchen?room="+room.RoomKey {
		t.Fatalf("sent back to %q", to)
	}
	u := b.session(t, r).User
	if u == nil || u.ID != guest.ID || u.Guest {
		t.Fatalf("after login: %+v, want %s signed in", u, guest.ID)
	}
	if u.DisplayName != "ana" {
		t.Fatalf("the name from the table was replaced: %q", u.DisplayName)
	}
	if len(u.Providers) != 1 || u.Providers[0] != "fake" {
		t.Fatalf("providers: %v", u.Providers)
	}
	if b[sessionCookie] == before {
		t.Fatal("the session was not replaced on login")
	}
	if old := (browser{sessionCookie: before}).session(t, r).User; old != nil {
		t.Fatal("the session from before the login still signs in")
	}
	if seatUser(t, h, room.ID, "ana") != guest.ID {
		t.Fatal("the seat lost its account")
	}
}

// A second browser logging in to an account it already has brings its
// guest's seats along, and the guest goes.
func TestLoggingInOnAnotherBrowserFoldsItsGuestIn(t *testing.T) {
	h, r, f := authServer(t)
	room := postCreate(t, h, r, "table")
	laptop, phone := browser{}, browser{}
	laptop.join(t, r, room.RoomKey, "ana")
	laptop.login(t, r, f, "sub-1", "/")
	account := laptop.session(t, r).User.ID

	phone.join(t, r, room.RoomKey, "bo")
	if seatUser(t, h, room.ID, "bo") == account {
		t.Fatal("setup: the phone should start as its own guest")
	}
	phone.login(t, r, f, "sub-1", "/")

	if got := phone.session(t, r).User; got == nil || got.ID != account {
		t.Fatalf("phone signed in as %+v, want %s", got, account)
	}
	if seatUser(t, h, room.ID, "bo") != account {
		t.Fatal("the phone's seat did not move to the account")
	}
	if laptop.session(t, r).User.ID != account {
		t.Fatal("signing in on the phone signed the laptop out")
	}
}

// Someone else signing in on a browser that is already signed in, from a
// tab whose menu still offers it, gets their own account. Their login must
// not be attached to the account already there, or they would be handed its
// seats and history, and its owner could sign in as them.
func TestANewLoginNeverJoinsTheAccountAlreadySignedIn(t *testing.T) {
	h, r, f := authServer(t)
	room := postCreate(t, h, r, "table")
	shared := browser{}
	shared.join(t, r, room.RoomKey, "ana")
	shared.login(t, r, f, "sub-ana", "/")
	ana := shared.session(t, r).User.ID

	shared.login(t, r, f, "sub-bo", "/")
	bo := shared.session(t, r).User
	if bo == nil || bo.Guest || bo.ID == ana {
		t.Fatalf("signing in as bo gave %+v; ana's account is %s", bo, ana)
	}
	if seats := shared.seats(t, r); len(seats) != 0 {
		t.Fatalf("bo was handed ana's seats: %+v", seats)
	}
	if seatUser(t, h, room.ID, "ana") != ana {
		t.Fatal("ana's seat left her account")
	}

	shared.login(t, r, f, "sub-ana", "/")
	if got := shared.session(t, r).User; got == nil || got.ID != ana {
		t.Fatalf("signing back in as ana gave %+v, want %s", got, ana)
	}
}

// Without the state it started with, a callback signs nobody in: otherwise
// anyone could send a victim's browser a link that signs it in as them.
func TestACallbackWithTheWrongStateSignsNobodyIn(t *testing.T) {
	_, r, f := authServer(t)
	b := browser{}
	w := b.do(r, httptest.NewRequest(http.MethodGet, "/api/auth/fake/login", nil))
	// The code would otherwise be good, so only the state stands in the way.
	out, _ := url.Parse(w.Header().Get("Location"))
	f.challenge = out.Query().Get("code_challenge")
	f.codes["code-attacker"] = "attacker"
	w = b.do(r, httptest.NewRequest(http.MethodGet, "/api/auth/fake/callback?code=code-attacker&state=guessed", nil))
	if w.Code != http.StatusFound {
		t.Fatalf("callback: %d", w.Code)
	}
	if u := b.session(t, r).User; u != nil {
		t.Fatalf("signed in as %+v", u)
	}

	// Nor does one with no login started in this browser at all.
	fresh := browser{}
	fresh.do(r, httptest.NewRequest(http.MethodGet, "/api/auth/fake/callback?code=code-attacker&state=guessed", nil))
	if u := fresh.session(t, r).User; u != nil {
		t.Fatalf("signed in without a flow: %+v", u)
	}
}

func TestLoginNeverReturnsToAnotherSite(t *testing.T) {
	_, r, f := authServer(t)
	for _, to := range []string{"https://evil.test/", "//evil.test/x", "/\\evil.test", "evil.test"} {
		if got := (browser{}).login(t, r, f, "sub-1", to); got != "/" {
			t.Errorf("return %q sent the browser to %q", to, got)
		}
	}
}

func TestLoggingOutEndsOnlyThisBrowsersSession(t *testing.T) {
	h, r, f := authServer(t)
	room := postCreate(t, h, r, "table")
	b := browser{}
	b.join(t, r, room.RoomKey, "ana")
	b.login(t, r, f, "sub-1", "/")
	account := b.session(t, r).User.ID

	cross := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	cross.Header.Set("Sec-Fetch-Site", "cross-site")
	if w := b.do(r, cross); w.Code != http.StatusForbidden {
		t.Fatalf("a cross-site logout was let through: %d", w.Code)
	}
	if b.session(t, r).User == nil {
		t.Fatal("the refused logout signed the browser out")
	}

	if w := b.do(r, httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)); w.Code != http.StatusNoContent {
		t.Fatalf("logout: %d %s", w.Code, w.Body)
	}
	if u := b.session(t, r).User; u != nil {
		t.Fatalf("still signed in as %+v", u)
	}
	if seatUser(t, h, room.ID, "ana") != account {
		t.Fatal("logging out took the seat from the account")
	}
	// Logging back in finds the same account.
	b.login(t, r, f, "sub-1", "/")
	if b.session(t, r).User.ID != account {
		t.Fatal("logging back in gave a different account")
	}
}

func TestSessionListsProvidersWithoutMakingAGuest(t *testing.T) {
	_, r, _ := authServer(t)
	b := browser{}
	s := b.session(t, r)
	if s.User != nil || b[sessionCookie] != "" {
		t.Fatal("asking for the session made a guest")
	}
	if len(s.Providers) != 1 || s.Providers[0] != "fake" {
		t.Fatalf("providers: %v", s.Providers)
	}
}

// Behind a proxy the provider must send people back to the address they used,
// not the one the proxy reached us on.
func TestTheCallbackGoesToTheAddressTheBrowserUsed(t *testing.T) {
	h, r, _ := authServer(t)
	h.auth.BaseURL = ""
	req := httptest.NewRequest(http.MethodGet, "/api/auth/fake/login", nil)
	req.Host = "localhost:8080"
	req.Header.Set("X-Forwarded-Host", "localhost:3000")
	w := (browser{}).do(r, req)
	out, _ := url.Parse(w.Header().Get("Location"))
	if got := out.Query().Get("redirect_uri"); got != "http://localhost:3000/api/auth/fake/callback" {
		t.Fatalf("redirect_uri = %q", got)
	}
}
