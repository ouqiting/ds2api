package prompt

import (
	"strings"
	"testing"
)

func TestSplitByRoleBoundary_NoSplitWhenUnderThreshold(t *testing.T) {
	prompt := "<System>:hello<User>:world<Assistant>:"
	segments := SplitByRoleBoundary(prompt, 1000)
	if len(segments) != 1 || segments[0] != prompt {
		t.Fatalf("expected single segment, got %d segments: %v", len(segments), segments)
	}
}

func TestSplitByRoleBoundary_SplitsAtRoleBoundary(t *testing.T) {
	long1 := strings.Repeat("A", 50)
	long2 := strings.Repeat("B", 50)
	prompt := "<User>:" + long1 + "<Assistant>:" + long2 + "<User>:final question<Assistant>:"
	segments := SplitByRoleBoundary(prompt, 60)
	if len(segments) < 2 {
		t.Fatalf("expected at least 2 segments, got %d", len(segments))
	}
	joined := strings.Join(segments, "")
	if joined != prompt {
		t.Fatalf("segments do not reconstruct original prompt:\njoined:  %q\noriginal:%q", joined, prompt)
	}
	for i, seg := range segments {
		if len([]rune(seg)) > 60 {
			t.Fatalf("segment %d exceeds maxChars: %d > 60", i, len([]rune(seg)))
		}
	}
}

func TestSplitByRoleBoundary_HardCutsOversizedBlock(t *testing.T) {
	longText := strings.Repeat("X", 300)
	prompt := "<User>:" + longText + "<Assistant>:"
	segments := SplitByRoleBoundary(prompt, 100)
	if len(segments) < 3 {
		t.Fatalf("expected at least 3 segments for oversized block, got %d", len(segments))
	}
	joined := strings.Join(segments, "")
	if joined != prompt {
		t.Fatalf("segments do not reconstruct original prompt")
	}
	for i, seg := range segments {
		if len([]rune(seg)) > 100 {
			t.Fatalf("segment %d exceeds maxChars: %d > 100", i, len([]rune(seg)))
		}
	}
}

func TestSplitByRoleBoundary_PreservesTrailingAssistantMarker(t *testing.T) {
	long1 := strings.Repeat("A", 50)
	prompt := "<System>:" + long1 + "<User>:question<Assistant>:"
	segments := SplitByRoleBoundary(prompt, 60)
	if len(segments) < 2 {
		t.Fatalf("expected at least 2 segments, got %d", len(segments))
	}
	last := segments[len(segments)-1]
	if !strings.HasSuffix(last, "<Assistant>:") {
		t.Fatalf("expected last segment to end with <Assistant>:, got %q", last)
	}
}

func TestSplitByRoleBoundary_SingleBlockNoMarkers(t *testing.T) {
	longText := strings.Repeat("X", 200)
	segments := SplitByRoleBoundary(longText, 50)
	if len(segments) < 3 {
		t.Fatalf("expected at least 3 segments, got %d", len(segments))
	}
	joined := strings.Join(segments, "")
	if joined != longText {
		t.Fatalf("segments do not reconstruct original text")
	}
}

func TestSplitByRoleBoundary_EmptyPrompt(t *testing.T) {
	segments := SplitByRoleBoundary("", 100)
	if len(segments) != 1 || segments[0] != "" {
		t.Fatalf("expected single empty segment, got %v", segments)
	}
}

func TestSplitByRoleBoundary_ExactThreshold(t *testing.T) {
	prompt := "<User>:hello<Assistant>:"
	segments := SplitByRoleBoundary(prompt, len([]rune(prompt)))
	if len(segments) != 1 {
		t.Fatalf("expected single segment when at exact threshold, got %d", len(segments))
	}
}
