package gptbridge

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// checklistTranslation declares Claude Code's real checklist tool so the engine
// resolves the call by its original name.
func checklistTranslation() Translation {
	translation := testTranslation("checklist")
	translation.DynamicTools = []DynamicTool{{
		Type: "function", Name: "TaskUpdate", Description: "update a task",
		InputSchema: json.RawMessage(`{"type":"object"}`),
	}}
	return translation
}

func callTaskUpdate(rpc *fakeEngineRPC, arguments string) {
	rpc.onTurnStart = func(threadID, turnID string) {
		rpc.requests <- ServerRequest{
			ID:     fakeRequestID("rpc"),
			Method: "item/tool/call",
			Params: json.RawMessage(fmt.Sprintf(
				`{"threadId":%q,"turnId":%q,"callId":"call","tool":"TaskUpdate","arguments":%s}`,
				threadID, turnID, arguments,
			)),
		}
	}
}

// A TaskUpdate carrying only a progress note changes nothing: Claude Code
// answers "Updated task #5 description" and never says the status still
// stands, so the model reads its own completion report back as success while
// the task stays in_progress for the rest of the session. The bridge refuses
// the shape instead of relaying it, which is the only thing that makes the
// model name a status.
func TestEngineRefusesAChecklistUpdateThatNamesNoStatus(t *testing.T) {
	for _, shape := range []struct {
		name      string
		arguments string
	}{
		{"description", `{"taskId":"5","description":"Implementation and scoped verification finished."}`},
		{"metadata", `{"taskId":"5","metadata":{"commit":"7db3397"}}`},
		{"note beside a dependency edit", `{"taskId":"5","addBlockedBy":["1"],"description":"done"}`},
	} {
		t.Run(shape.name, func(t *testing.T) {
			rpc := newFakeEngineRPC()
			callTaskUpdate(rpc, shape.arguments)
			rpc.onRespond = func(count int) {
				if count == 1 {
					completeTextTurn(rpc, "thread-1", "turn-1", "noted")
				}
			}
			engine := newTestEngine(t, rpc)

			message, err := engine.Execute(context.Background(), checklistTranslation(), nil)
			if err != nil {
				t.Fatal(err)
			}
			if message.StopReason != "end_turn" {
				t.Fatalf("stop reason = %q, want end_turn: the refused call must never reach Claude", message.StopReason)
			}
			for _, block := range message.Content {
				if block.Type == "tool_use" {
					t.Fatalf("refused call was relayed to Claude: %+v", block)
				}
			}
			responses := append([]fakeResponse(nil), rpc.responses...)
			if len(responses) != 1 {
				t.Fatalf("responses = %+v, want the refusal answered once", responses)
			}
			result := string(responses[0].Result)
			if !strings.Contains(result, `"success":false`) {
				t.Fatalf("refusal is not an unsuccessful result: %s", result)
			}
			for _, want := range []string{"status", "TaskUpdate"} {
				if !strings.Contains(result, want) {
					t.Fatalf("refusal omits %q: %s", want, result)
				}
			}
			if engine.PendingTurns() != 0 {
				t.Fatalf("refused call left %d suspended turns", engine.PendingTurns())
			}
		})
	}
}

// The refusal is scoped to the shape that silently loses work. A call that
// names a status is the whole point of the tool, and a call that only wires
// dependencies or claims an owner changes no status by design — Claude's own
// models issue those, so refusing them would break ordinary bookkeeping.
func TestEngineRelaysEveryOtherChecklistUpdate(t *testing.T) {
	for _, shape := range []struct {
		name      string
		arguments string
	}{
		{"status alone", `{"taskId":"5","status":"completed"}`},
		{"status with a note", `{"taskId":"5","status":"completed","description":"finished"}`},
		{"dependency wiring", `{"taskId":"2","addBlockedBy":["1"]}`},
		{"owner claim", `{"taskId":"2","owner":"agent-a"}`},
	} {
		t.Run(shape.name, func(t *testing.T) {
			rpc := newFakeEngineRPC()
			callTaskUpdate(rpc, shape.arguments)
			engine := newTestEngine(t, rpc)

			message, err := engine.Execute(context.Background(), checklistTranslation(), nil)
			if err != nil {
				t.Fatal(err)
			}
			if message.StopReason != "tool_use" || len(message.Content) != 1 {
				t.Fatalf("response = %+v, want the call relayed to Claude", message)
			}
			if message.Content[0].Name != "TaskUpdate" {
				t.Fatalf("relayed tool = %q", message.Content[0].Name)
			}
		})
	}
}

// The shipped case, verbatim. Session 62943587 closed tasks #8 and #7 with an
// explicit "completed" and then wrote the same kind of completion report for #5
// with no status at all — its last checklist write of the session. The pane was
// left reading "8 tasks (7 done, 1 in progress, 0 open)" with the parent task
// stranded. The three payloads are copied from records 3853, 3855 and 3857.
func TestEngineRefusesTheChecklistNoteThatStrandedTaskFive(t *testing.T) {
	const strandedFive = `{"taskId":"5","description":"Local approved picker implementation, scoped tests, ` +
		`built browser checks and final read-only reviews finished. Latest section fix: 27 targeted units ` +
		`and 11 built E2E passed, light/dark rapid-click/manual-scroll/reduced-motion QA passed with zero ` +
		`fresh runtime errors. Changed-file lint and diff check exit 0."}`
	const closedSeven = `{"taskId":"7","status":"completed","description":"Scoped implementation and ` +
		`verification complete. Review now has no remaining actionable finding."}`

	rpc := newFakeEngineRPC()
	callTaskUpdate(rpc, strandedFive)
	rpc.onRespond = func(count int) {
		if count == 1 {
			completeTextTurn(rpc, "thread-1", "turn-1", "acknowledged")
		}
	}
	engine := newTestEngine(t, rpc)
	message, err := engine.Execute(context.Background(), checklistTranslation(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if message.StopReason != "end_turn" {
		t.Fatalf("the note that stranded task #5 still reaches Claude: %+v", message)
	}

	// Its well-formed sibling from the same breath must still go through, or
	// the refusal would block the model from ever closing a task.
	sibling := newFakeEngineRPC()
	callTaskUpdate(sibling, closedSeven)
	siblingEngine := newTestEngine(t, sibling)
	closed, err := siblingEngine.Execute(context.Background(), checklistTranslation(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if closed.StopReason != "tool_use" || len(closed.Content) != 1 {
		t.Fatalf("closing task #7 was blocked: %+v", closed)
	}
}

// The predicate itself, over the shapes the engine tests cannot all afford to
// drive end to end. A status that names nothing must not open the gate.
func TestUnstatusedChecklistNote(t *testing.T) {
	for _, shape := range []struct {
		arguments string
		refuse    bool
	}{
		{`{"taskId":"5","description":"finished"}`, true},
		{`{"taskId":"5","metadata":{"commit":"abc"}}`, true},
		{`{"taskId":"5","description":"finished","metadata":{"commit":"abc"}}`, true},
		{`{"taskId":"5","status":null,"description":"finished"}`, true},
		{`{"taskId":"5","status":"","description":"finished"}`, true},
		{`{"taskId":"5","status":7,"description":"finished"}`, true},
		{`{"taskId":"5","status":"completed","description":"finished"}`, false},
		{`{"taskId":"5","status":"in_progress"}`, false},
		{`{"taskId":"5","addBlockedBy":["1"]}`, false},
		{`{"taskId":"5","subject":"Run tests"}`, false},
		{`{"taskId":"5","activeForm":"Running tests"}`, false},
		{`{"taskId":"5"}`, false},
	} {
		if got := unstatusedChecklistNote(taskUpdateTool, json.RawMessage(shape.arguments)); got != shape.refuse {
			t.Errorf("unstatusedChecklistNote(%s) = %t, want %t", shape.arguments, got, shape.refuse)
		}
		if unstatusedChecklistNote("Bash", json.RawMessage(shape.arguments)) {
			t.Errorf("a non-checklist tool was inspected: %s", shape.arguments)
		}
	}
}

// Only Claude Code's checklist tool is inspected. Every other tool's arguments
// belong to the host that hosts it.
func TestEngineRelaysANoteToAToolThatIsNotTheChecklist(t *testing.T) {
	rpc := newFakeEngineRPC()
	rpc.onTurnStart = func(threadID, turnID string) {
		rpc.requests <- ServerRequest{
			ID:     fakeRequestID("rpc"),
			Method: "item/tool/call",
			Params: json.RawMessage(fmt.Sprintf(
				`{"threadId":%q,"turnId":%q,"callId":"call","tool":"Echo","arguments":{"description":"note"}}`,
				threadID, turnID,
			)),
		}
	}
	engine := newTestEngine(t, rpc)

	message, err := engine.Execute(context.Background(), testTranslation("tools"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if message.StopReason != "tool_use" || len(message.Content) != 1 {
		t.Fatalf("response = %+v, want the call relayed", message)
	}
}
