package document

import "testing"

func TestIsGreeting(t *testing.T) {
	tests := []struct {
		message string
		want    bool
	}{
		{"hello", true},
		{"helllo", true},
		{"heello", true},
		{"halo", true},
		{"hai", true},
		{"hi", true},
		{"hey", true},
		{"selamat pagi", true},
		{"Hello!", true},
		{"this is about documents", false},
		{"history of AI", false},
		{"apa isi dokumen tentang machine learning", false},
	}

	for _, tt := range tests {
		if got := isGreeting(tt.message); got != tt.want {
			t.Errorf("isGreeting(%q) = %v, want %v", tt.message, got, tt.want)
		}
	}
}
