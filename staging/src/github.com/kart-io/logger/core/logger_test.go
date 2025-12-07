package core

import (
	"testing"
)

func TestLevel_String(t *testing.T) {
	tests := []struct {
		level Level
		want  string
	}{
		{DebugLevel, "debug"},
		{InfoLevel, "info"},
		{WarnLevel, "warn"},
		{ErrorLevel, "error"},
		{FatalLevel, "fatal"},
		{Level(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.level.String(); got != tt.want {
				t.Errorf("Level.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		text    string
		want    Level
		wantErr bool
	}{
		{"DEBUG", DebugLevel, false},
		{"debug", DebugLevel, false},
		{"INFO", InfoLevel, false},
		{"info", InfoLevel, false},
		{"WARN", WarnLevel, false},
		{"WARNING", WarnLevel, false},
		{"ERROR", ErrorLevel, false},
		{"FATAL", FatalLevel, false},
		{"invalid", InfoLevel, true},
		{"", InfoLevel, true},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			got, err := ParseLevel(tt.text)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseLevel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseLevel() = %v, want %v", got, tt.want)
			}
		})
	}
}
