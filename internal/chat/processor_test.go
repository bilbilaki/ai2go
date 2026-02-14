package chat

import "testing"

func TestParseAskUserArgsWithOptions(t *testing.T) {
	question, options, err := parseAskUserArgs(`{"question":"Pick one","options":["A","B",""]}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if question != "Pick one" {
		t.Fatalf("unexpected question: %q", question)
	}
	if len(options) != 2 || options[0] != "A" || options[1] != "B" {
		t.Fatalf("unexpected options: %#v", options)
	}
}

func TestParseAskUserArgsRequiresQuestion(t *testing.T) {
	_, _, err := parseAskUserArgs(`{"options":["A"]}`)
	if err == nil {
		t.Fatal("expected error for missing question")
	}
}
