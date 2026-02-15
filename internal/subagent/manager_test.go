package subagent

import "testing"

func TestParseFactoryInputDefaultsRaisedTimeout(t *testing.T) {
	raw := `{"mega_prompt":"task A"}`

	in, err := ParseFactoryInput(raw)
	if err != nil {
		t.Fatalf("ParseFactoryInput returned error: %v", err)
	}

	if in.TimeoutSec != defaultTimeoutSec {
		t.Fatalf("expected timeout default %d, got %d", defaultTimeoutSec, in.TimeoutSec)
	}
	if defaultTimeoutSec != 600 {
		t.Fatalf("expected defaultTimeoutSec to be 600, got %d", defaultTimeoutSec)
	}
}

func TestSplitTasksByDefaultSymbol(t *testing.T) {
	input := FactoryInput{
		MegaPrompt: "first\n---TASK---\nsecond\n---TASK---\n\nthird",
	}

	got, err := splitTasks(input)
	if err != nil {
		t.Fatalf("splitTasks returned error: %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(got))
	}
	if got[0] != "first" || got[1] != "second" || got[2] != "third" {
		t.Fatalf("unexpected split result: %#v", got)
	}
}

func TestParseMiniEditorHelperInputDefaults(t *testing.T) {
	raw := `{"prompt":"fix file"}` + "\n"
	in, err := ParseMiniEditorHelperInput(raw)
	if err != nil {
		t.Fatalf("ParseMiniEditorHelperInput error: %v", err)
	}
	if in.Prompt != "fix file" {
		t.Fatalf("unexpected prompt: %q", in.Prompt)
	}
	if in.TimeoutSec != defaultMiniTimeoutSec {
		t.Fatalf("expected default mini timeout %d, got %d", defaultMiniTimeoutSec, in.TimeoutSec)
	}
}
