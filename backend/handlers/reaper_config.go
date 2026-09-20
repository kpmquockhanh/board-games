package handlers

import (
	"log"
	"os"
	"strconv"
	"time"
)

// Environment variables that tune the reaper. Every one of them is optional:
// an unset variable leaves the default from DefaultReaperConfig in place, and
// a value that cannot be read is logged and ignored rather than taken half-way
// or allowed to stop the server from booting.
const (
	EnvReapEnabled         = "REAP_ENABLED"
	EnvReapInterval        = "REAP_INTERVAL"
	EnvReapAbandonAfter    = "REAP_ABANDON_AFTER"
	EnvReapDeleteAfter     = "REAP_DELETE_AFTER"
	EnvReapDropPlayerAfter = "REAP_DROP_PLAYER_AFTER"
	EnvReapTimelineKeep    = "REAP_TIMELINE_KEEP"
)

// ReaperConfigFromEnv is DefaultReaperConfig with any REAP_* overrides applied.
// Durations are written the way Go writes them — "30s", "10m", "24h", "1h30m"
// — so that a window is stated in whatever unit suits it.
//
// Deployments differ in what they want here: a public instance wants seats
// reclaimed briskly, while someone debugging a game locally wants the reaper
// to keep its hands off the room they have been staring at for an hour.
func ReaperConfigFromEnv() ReaperConfig {
	cfg := DefaultReaperConfig()

	cfg.Enabled = envBool(EnvReapEnabled, cfg.Enabled)
	cfg.Interval = envDuration(EnvReapInterval, cfg.Interval)
	cfg.AbandonAfter = envDuration(EnvReapAbandonAfter, cfg.AbandonAfter)
	cfg.DeleteAfter = envDuration(EnvReapDeleteAfter, cfg.DeleteAfter)
	cfg.DropPlayerAfter = envDuration(EnvReapDropPlayerAfter, cfg.DropPlayerAfter)
	cfg.TimelineKeep = envInt(EnvReapTimelineKeep, cfg.TimelineKeep)

	// Worth saying out loud rather than silently obeying: holding a seat for
	// longer than the room itself survives means the room is abandoned out
	// from under a player whose seat was still being kept for them, so the
	// forfeit phase never gets to run.
	if cfg.DropPlayerAfter >= cfg.AbandonAfter {
		log.Printf("[reaper] warning: %s (%s) is not shorter than %s (%s), so idle rooms will be abandoned before a dropped player is ever forfeited",
			EnvReapDropPlayerAfter, cfg.DropPlayerAfter, EnvReapAbandonAfter, cfg.AbandonAfter)
	}

	return cfg
}

// envDuration reads a duration, keeping def if the variable is unset, unparsable
// or not positive. Zero is refused along with the negatives: a zero Interval
// panics the ticker, and a zero window would reap rooms the moment they went
// quiet.
func envDuration(name string, def time.Duration) time.Duration {
	raw, ok := os.LookupEnv(name)
	if !ok || raw == "" {
		return def
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		log.Printf("[reaper] ignoring %s=%q: %v (using %s)", name, raw, err, def)
		return def
	}
	if d <= 0 {
		log.Printf("[reaper] ignoring %s=%q: must be positive (using %s)", name, raw, def)
		return def
	}
	return d
}

// envInt reads a count, keeping def if the variable is unset, unparsable or
// negative. Zero is allowed: it means keep no ordinary events at all.
func envInt(name string, def int) int {
	raw, ok := os.LookupEnv(name)
	if !ok || raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		log.Printf("[reaper] ignoring %s=%q: %v (using %d)", name, raw, err, def)
		return def
	}
	if n < 0 {
		log.Printf("[reaper] ignoring %s=%q: must not be negative (using %d)", name, raw, def)
		return def
	}
	return n
}

// envBool accepts the forms strconv does: 1/0, t/f, true/false, TRUE/False.
func envBool(name string, def bool) bool {
	raw, ok := os.LookupEnv(name)
	if !ok || raw == "" {
		return def
	}
	b, err := strconv.ParseBool(raw)
	if err != nil {
		log.Printf("[reaper] ignoring %s=%q: %v (using %t)", name, raw, err, def)
		return def
	}
	return b
}
