package handlers

import (
	"testing"
	"time"
)

func TestReaperConfigFromEnvDefaultsWhenNothingIsSet(t *testing.T) {
	got := ReaperConfigFromEnv()
	if got != DefaultReaperConfig() {
		t.Fatalf("got %+v, want the defaults %+v", got, DefaultReaperConfig())
	}
}

func TestReaperConfigFromEnvAppliesOverrides(t *testing.T) {
	t.Setenv(EnvReapEnabled, "false")
	t.Setenv(EnvReapInterval, "30s")
	t.Setenv(EnvReapAbandonAfter, "1h30m")
	t.Setenv(EnvReapDeleteAfter, "72h")
	t.Setenv(EnvReapDropPlayerAfter, "45s")
	t.Setenv(EnvReapTimelineKeep, "10")
	t.Setenv(EnvReapGuestsAfter, "720h")

	want := ReaperConfig{
		Enabled:         false,
		Interval:        30 * time.Second,
		AbandonAfter:    90 * time.Minute,
		DeleteAfter:     72 * time.Hour,
		DropPlayerAfter: 45 * time.Second,
		TimelineKeep:    10,
		GuestsAfter:     30 * 24 * time.Hour,
	}
	if got := ReaperConfigFromEnv(); got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

// A typo must not quietly become a zero: a zero Interval panics the ticker and
// a zero window would reap a room the instant it went quiet.
func TestReaperConfigFromEnvKeepsDefaultsForUnusableValues(t *testing.T) {
	def := DefaultReaperConfig()
	for _, tc := range []struct {
		name string
		env  string
		val  string
	}{
		{"unparsable duration", EnvReapInterval, "5 minutes"},
		{"bare number as a duration", EnvReapAbandonAfter, "600"},
		{"zero duration", EnvReapInterval, "0s"},
		{"negative duration", EnvReapDropPlayerAfter, "-5m"},
		{"empty", EnvReapDeleteAfter, ""},
		{"unparsable count", EnvReapTimelineKeep, "lots"},
		{"negative count", EnvReapTimelineKeep, "-1"},
		{"unparsable bool", EnvReapEnabled, "yes please"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(tc.env, tc.val)
			if got := ReaperConfigFromEnv(); got != def {
				t.Fatalf("%s=%q gave %+v, want the defaults %+v", tc.env, tc.val, got, def)
			}
		})
	}
}

// Zero is a real answer for the event cap — keep the snapshots, drop the log.
func TestReaperConfigFromEnvAcceptsZeroTimelineKeep(t *testing.T) {
	t.Setenv(EnvReapTimelineKeep, "0")
	if got := ReaperConfigFromEnv(); got.TimelineKeep != 0 {
		t.Fatalf("TimelineKeep = %d, want 0", got.TimelineKeep)
	}
}

// A disabled reaper must return rather than sweep, and must not block the
// caller who started it.
func TestRunReaperReturnsImmediatelyWhenDisabled(t *testing.T) {
	h, r := newTestHandler(t)
	room := postCreate(t, h, r, "left alone")

	cfg := testCfg()
	cfg.Enabled = false
	cfg.Interval = time.Millisecond

	done := make(chan struct{})
	go func() {
		h.RunReaper(t.Context(), cfg)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("RunReaper did not return with the reaper disabled")
	}

	time.Sleep(20 * time.Millisecond)
	after, err := h.store.GetRoom(room.RoomKey)
	if err != nil || after == nil {
		t.Fatalf("room is gone: %v", err)
	}
	if after.Status != StatusActive {
		t.Fatalf("room status = %q, want %q — a disabled reaper swept anyway", after.Status, StatusActive)
	}
}
