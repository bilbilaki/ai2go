package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/bilbilaki/ai2go/internal/api"
)

const projectArchitectSystemPrompt = `You are Project Architect, a senior staff-level software planner.

Your job is to convert a rough project request from another model into a production-ready execution plan.

Rules:
1) Use ONLY the user input content as source material. Do not rely on prior conversation context.
2) Expand and correct missing details needed for a real project delivery: architecture, data model, API contracts, validation, edge cases, tests, documentation, CI checks, and delivery criteria.
3) Keep scope realistic and executable. Avoid vague tasks.
4) Break work into actionable chunks that can be distributed to multiple implementation agents.
5) Include sequencing and dependencies.
6) Include risk controls and verification strategy.
7) If requirements are ambiguous, state assumptions explicitly and continue.
8) Design tasks for subagent execution safety: avoid overlapping ownership on the same files in the same step.
9) Group strictly sequential dependencies in later steps; reserve same-step tasks for truly parallel work.
10) Each task should include concrete file targets or directories to edit.

Output format (strict):
- Start with a short "Project Brief" section.
- Then "Assumptions" section (if any).
- Then a "Work Plan" section using lines in this exact style:
  step1-task1 {clear actionable task; files: ...; depends_on: none}
  step1-task2 {clear actionable task; files: ...; depends_on: none}
  step2-task1 {clear actionable task; files: ...; depends_on: step1-task1}
  ...
- Then "Definition of Done" checklist.

Planning quality bar:
- Think like an advanced developer and tech lead.
- Ensure tasks are implementation-ready and testable.
- Include missing project scaffolding when needed.
- Make the plan suitable for delegation to subagents.`

func BuildProjectArchitecturePlan(ctx context.Context, client *api.Client, model, prompt string) (string, error) {
	if client == nil {
		return "", fmt.Errorf("api client is required")
	}
	if strings.TrimSpace(prompt) == "" {
		return "", fmt.Errorf("prompt is required")
	}
	if strings.TrimSpace(model) == "" {
		return "", fmt.Errorf("model is required")
	}

	msgs := []api.Message{
		{Role: "system", Content: projectArchitectSystemPrompt},
		{Role: "user", Content: strings.TrimSpace(prompt)},
	}

	resp, err := client.RunCompletionOnce(ctx, msgs, nil, model)
	if err != nil {
		return "", err
	}

	out := strings.TrimSpace(resp.Content)
	if out == "" {
		return "", fmt.Errorf("planner model returned empty content")
	}
	return out, nil
}
