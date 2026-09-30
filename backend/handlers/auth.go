package handlers

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"ping/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

// Logging in is OAuth 2 with PKCE against a provider's login page. It only
// ever adds to what the session cookie already does (see user.go): the account
// it signs in to is the one recorded on seats, and a seat is still proven by
// its own token alone.

// Env vars that turn on a provider. One without both an id and a secret is
// off, and the app offers no login at all when every one is.
const (
	EnvAuthBaseURL         = "AUTH_BASE_URL"
	EnvGoogleClientID      = "AUTH_GOOGLE_CLIENT_ID"
	EnvGoogleClientSecret  = "AUTH_GOOGLE_CLIENT_SECRET"
	EnvDiscordClientID     = "AUTH_DISCORD_CLIENT_ID"
	EnvDiscordClientSecret = "AUTH_DISCORD_CLIENT_SECRET"
)

// The flow cookie carries a login's state and PKCE verifier from the redirect
// out to the provider's redirect back. It is scoped to the auth routes and
// lives only as long as someone might take to log in.
const (
	flowCookie   = "ping_oauth"
	flowCookieAt = "/api/auth"
	flowMaxAge   = 10 * time.Minute
)

// Provider is one place to log in with.
type Provider struct {
	Name    string
	OAuth   oauth2.Config
	Profile string // URL of the logged-in user's profile
	// Parse reads who the profile is about.
	Parse func(body []byte) (models.Identity, error)
}

// Auth is the login setup: the providers on offer, and the public address the
// providers send people back to ("" to take it from each request).
type Auth struct {
	BaseURL   string
	Providers map[string]*Provider
}

// Names lists the providers on offer.
func (a *Auth) Names() []string {
	names := []string{}
	if a == nil {
		return names
	}
	for name := range a.Providers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// AuthFromEnv turns on each provider whose client id and secret are set.
func AuthFromEnv() *Auth {
	a := &Auth{
		BaseURL:   strings.TrimRight(os.Getenv(EnvAuthBaseURL), "/"),
		Providers: map[string]*Provider{},
	}
	if id, secret := os.Getenv(EnvGoogleClientID), os.Getenv(EnvGoogleClientSecret); id != "" && secret != "" {
		a.Providers["google"] = googleProvider(id, secret)
	}
	if id, secret := os.Getenv(EnvDiscordClientID), os.Getenv(EnvDiscordClientSecret); id != "" && secret != "" {
		a.Providers["discord"] = discordProvider(id, secret)
	}
	return a
}

func googleProvider(id, secret string) *Provider {
	return &Provider{
		Name: "google",
		OAuth: oauth2.Config{
			ClientID:     id,
			ClientSecret: secret,
			Scopes:       []string{"openid", "profile", "email"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://accounts.google.com/o/oauth2/v2/auth",
				TokenURL: "https://oauth2.googleapis.com/token",
			},
		},
		Profile: "https://openidconnect.googleapis.com/v1/userinfo",
		Parse: func(body []byte) (models.Identity, error) {
			var p struct {
				Sub, Name, Email string
			}
			err := json.Unmarshal(body, &p)
			return models.Identity{Provider: "google", Subject: p.Sub, Name: p.Name, Email: p.Email}, err
		},
	}
}

func discordProvider(id, secret string) *Provider {
	return &Provider{
		Name: "discord",
		OAuth: oauth2.Config{
			ClientID:     id,
			ClientSecret: secret,
			Scopes:       []string{"identify"},
			Endpoint: oauth2.Endpoint{
				AuthURL:   "https://discord.com/oauth2/authorize",
				TokenURL:  "https://discord.com/api/oauth2/token",
				AuthStyle: oauth2.AuthStyleInParams,
			},
		},
		Profile: "https://discord.com/api/users/@me",
		Parse: func(body []byte) (models.Identity, error) {
			var p struct {
				ID         string `json:"id"`
				Username   string `json:"username"`
				GlobalName string `json:"global_name"`
			}
			err := json.Unmarshal(body, &p)
			name := p.GlobalName
			if name == "" {
				name = p.Username
			}
			return models.Identity{Provider: "discord", Subject: p.ID, Name: name}, err
		},
	}
}

// UseAuth sets which providers the app logs in with.
func (h *Handler) UseAuth(a *Auth) { h.auth = a }

// RegisterAuth adds the login routes to the /api group.
func (h *Handler) RegisterAuth(api gin.IRoutes) {
	api.GET("/auth/session", h.AuthSession)
	api.GET("/auth/:provider/login", h.Login)
	api.GET("/auth/:provider/callback", h.LoginCallback)
	api.POST("/auth/logout", sameSiteOnly, h.Logout)
}

// sameSiteOnly refuses a request another site's page made. The session
// cookie is SameSite=Lax, so such a request would not carry it anyway; this
// holds even for a browser that ignores that.
func sameSiteOnly(c *gin.Context) {
	if c.GetHeader("Sec-Fetch-Site") == "cross-site" {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "cross-site request refused"})
		return
	}
	c.Next()
}

// AuthSession says who this browser is signed in as, if anyone, and how it
// could sign in. Unlike Me it never makes a guest, so a page can ask on load.
func (h *Handler) AuthSession(c *gin.Context) {
	out := gin.H{"user": nil, "providers": h.auth.Names()}
	if u := currentUser(c); u != nil {
		providers, err := h.store.UserProviders(u.ID)
		if err != nil {
			log.Printf("[auth] providers of %s: %v", u.ID, err)
		}
		out["user"] = gin.H{
			"id": u.ID, "guest": u.Guest, "display_name": u.DisplayName, "color": u.Color,
			"providers": providers,
		}
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) provider(c *gin.Context) *Provider {
	if h.auth == nil {
		return nil
	}
	return h.auth.Providers[c.Param("provider")]
}

// callbackURL is where the provider sends people back to. It must match what
// the provider was registered with, character for character, so set
// AUTH_BASE_URL in production. Without it the address comes from the request:
// the host the browser asked the proxy in front for, not the one the proxy
// asked us for. A made-up forwarded host only makes a URL the provider
// refuses, since it is not the registered one.
func (h *Handler) callbackURL(c *gin.Context, p *Provider) string {
	base := h.auth.BaseURL
	if base == "" {
		scheme := "http"
		if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		host := c.GetHeader("X-Forwarded-Host")
		if host == "" {
			host = c.Request.Host
		}
		base = scheme + "://" + host
	}
	return base + "/api/auth/" + p.Name + "/callback"
}

func (h *Handler) oauthConfig(c *gin.Context, p *Provider) *oauth2.Config {
	cfg := p.OAuth
	cfg.RedirectURL = h.callbackURL(c, p)
	return &cfg
}

// localPath keeps a return address on this site. Anything else, including a
// scheme-relative "//elsewhere", goes home instead of off to another site.
func localPath(raw string) string {
	if !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") || strings.HasPrefix(raw, "/\\") {
		return "/"
	}
	if u, err := url.Parse(raw); err != nil || u.Host != "" || u.Scheme != "" {
		return "/"
	}
	return raw
}

type loginFlow struct {
	State    string `json:"s"`
	Verifier string `json:"v"`
	Return   string `json:"r"`
}

func setFlowCookie(c *gin.Context, value string, maxAge int) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     flowCookie,
		Value:    value,
		Path:     flowCookieAt,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https",
		// Lax, so it comes back on the provider's top-level redirect.
		SameSite: http.SameSiteLaxMode,
	})
}

// Login sends the browser to the provider's login page.
func (h *Handler) Login(c *gin.Context) {
	p := h.provider(c)
	if p == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no such login provider"})
		return
	}
	state, _, err := newSecret()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not start login"})
		return
	}
	flow := loginFlow{State: state, Verifier: oauth2.GenerateVerifier(), Return: localPath(c.Query("return"))}
	raw, _ := json.Marshal(flow)
	setFlowCookie(c, base64.RawURLEncoding.EncodeToString(raw), int(flowMaxAge/time.Second))
	c.Redirect(http.StatusFound, h.oauthConfig(c, p).AuthCodeURL(state, oauth2.S256ChallengeOption(flow.Verifier)))
}

// LoginCallback is where the provider sends the browser back. Whatever goes
// wrong, the browser is sent back to where it was, only not signed in.
func (h *Handler) LoginCallback(c *gin.Context) {
	p := h.provider(c)
	if p == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no such login provider"})
		return
	}
	flow, ok := readFlow(c)
	setFlowCookie(c, "", -1) // one use, success or not
	if !ok {
		c.Redirect(http.StatusFound, "/")
		return
	}
	// The state proves this browser started the login, so nobody can sign
	// someone else's browser in to their own account.
	if subtle.ConstantTimeCompare([]byte(c.Query("state")), []byte(flow.State)) != 1 {
		log.Printf("[auth] %s login refused: state mismatch", p.Name)
		c.Redirect(http.StatusFound, flow.Return)
		return
	}
	if e := c.Query("error"); e != "" || c.Query("code") == "" {
		log.Printf("[auth] %s login not completed: %q", p.Name, e)
		c.Redirect(http.StatusFound, flow.Return)
		return
	}

	id, err := h.identify(c, p, flow)
	if err != nil {
		log.Printf("[auth] %s login failed: %v", p.Name, err)
		c.Redirect(http.StatusFound, flow.Return)
		return
	}

	token, hash, err := newSecret()
	if err != nil {
		c.Redirect(http.StatusFound, flow.Return)
		return
	}
	previous := ""
	if raw, err := c.Cookie(sessionCookie); err == nil && raw != "" {
		previous = hashSecret(raw)
	}
	u, err := h.store.SignIn(hash, previous, id)
	if err != nil {
		log.Printf("[auth] %s sign in: %v", p.Name, err)
		c.Redirect(http.StatusFound, flow.Return)
		return
	}
	setSessionCookie(c, token)
	log.Printf("[auth] %s signed in as %s", p.Name, u.ID)
	c.Redirect(http.StatusFound, flow.Return)
}

func readFlow(c *gin.Context) (loginFlow, bool) {
	var flow loginFlow
	raw, err := c.Cookie(flowCookie)
	if err != nil || raw == "" {
		return flow, false
	}
	data, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil || json.Unmarshal(data, &flow) != nil || flow.State == "" || flow.Verifier == "" {
		return flow, false
	}
	flow.Return = localPath(flow.Return)
	return flow, true
}

// identify trades the provider's code for a token, and the token for who
// logged in.
func (h *Handler) identify(c *gin.Context, p *Provider, flow loginFlow) (models.Identity, error) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()
	cfg := h.oauthConfig(c, p)
	tok, err := cfg.Exchange(ctx, c.Query("code"), oauth2.VerifierOption(flow.Verifier))
	if err != nil {
		return models.Identity{}, fmt.Errorf("exchange code: %w", err)
	}
	res, err := cfg.Client(ctx, tok).Get(p.Profile)
	if err != nil {
		return models.Identity{}, fmt.Errorf("fetch profile: %w", err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return models.Identity{}, fmt.Errorf("read profile: %w", err)
	}
	if res.StatusCode != http.StatusOK {
		return models.Identity{}, fmt.Errorf("profile: %s", res.Status)
	}
	id, err := p.Parse(body)
	if err != nil {
		return models.Identity{}, fmt.Errorf("parse profile: %w", err)
	}
	if id.Subject == "" {
		return models.Identity{}, fmt.Errorf("profile names nobody")
	}
	return id, nil
}

// Logout signs this browser out. Its seats are untouched: each tab still
// holds its seat's token, so nobody is taken out of a game by it.
func (h *Handler) Logout(c *gin.Context) {
	if raw, err := c.Cookie(sessionCookie); err == nil && raw != "" {
		if err := h.store.EndSession(hashSecret(raw)); err != nil {
			log.Printf("[auth] end session: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not sign out"})
			return
		}
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name: sessionCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode,
	})
	c.Status(http.StatusNoContent)
}
