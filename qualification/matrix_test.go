package qualification

import "testing"

func TestDeriveStatus(t *testing.T) {
	if got := DeriveStatus(Dimensions{Tools: true, Pricing: true, Conversation: true, CodingLoop: true, Usage: true}); got != StatusExperimental {
		t.Fatalf("expected experimental, got %s", got)
	}
}
