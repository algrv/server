package ccsignals

import (
	"testing"
	"time"
)

func TestCCSignal_IsValid(t *testing.T) {
	tests := []struct {
		signal CCSignal
		want   bool
	}{
		{SignalCredit, true},
		{SignalDirect, true},
		{SignalEcosystem, true},
		{SignalOpen, true},
		{SignalNoAI, true},
		{"", false},
		{"invalid", false},
		{"cc-invalid", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.signal), func(t *testing.T) {
			got := tt.signal.IsValid()
			if got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCCSignal_AllowsAI(t *testing.T) {
	tests := []struct {
		signal CCSignal
		want   bool
	}{
		{SignalCredit, true},
		{SignalDirect, true},
		{SignalEcosystem, true},
		{SignalOpen, true},
		{SignalNoAI, false},
		{"", true},        // empty allows AI (fail open)
		{"unknown", true}, // unknown allows AI (fail open)
	}

	for _, tt := range tests {
		t.Run(string(tt.signal), func(t *testing.T) {
			got := tt.signal.AllowsAI()
			if got != tt.want {
				t.Errorf("AllowsAI() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.PasteDeltaThreshold != 200 {
		t.Errorf("PasteDeltaThreshold = %d, want 200", config.PasteDeltaThreshold)
	}

	if config.PasteLineThreshold != 50 {
		t.Errorf("PasteLineThreshold = %d, want 50", config.PasteLineThreshold)
	}

	if config.UnlockThreshold != 0.30 {
		t.Errorf("UnlockThreshold = %v, want 0.30", config.UnlockThreshold)
	}

	if config.LockTTL != 1*time.Hour {
		t.Errorf("LockTTL = %v, want 1h", config.LockTTL)
	}
}
