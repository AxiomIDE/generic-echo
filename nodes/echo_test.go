package nodes_test

import (
	"context"
	"testing"

	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/reflect/protoreflect"

	"axiom-official/generic-echo/axiom"
	gen "axiom-official/generic-echo/gen"
	"axiom-official/generic-echo/nodes"
)

// testContext is a testing.T-backed axiom.Context for unit tests. Populate
// secretsMap with any secrets your node needs during the test.
type testContext struct {
	t          *testing.T
	secretsMap map[string]string
}

func newTestContext(t *testing.T) *testContext {
	return &testContext{t: t, secretsMap: map[string]string{}}
}

// testLogger forwards log output to testing.T so it is captured per-test.
type testLogger struct{ t *testing.T }

func (l *testLogger) Debug(msg string, args ...any) { l.t.Logf("DEBUG  %s %v", msg, args) }
func (l *testLogger) Info(msg string, args ...any)  { l.t.Logf("INFO   %s %v", msg, args) }
func (l *testLogger) Warn(msg string, args ...any)  { l.t.Logf("WARN   %s %v", msg, args) }
func (l *testLogger) Error(msg string, args ...any) { l.t.Logf("ERROR  %s %v", msg, args) }

// testSecrets is a simple in-memory axiom.Secrets backed by testContext.secretsMap.
type testSecrets struct{ m map[string]string }

func (s testSecrets) Get(name string) (string, bool) { v, ok := s.m[name]; return v, ok }

// testAgent is a no-op axiom.Agent for unit tests.
// ADR-002 (2026-04-19): memory is under Agent, not directly on Context.
type testAgent struct{}

func (testAgent) Memory() axiom.AgentMemory { return testAgentMemory{} }

type testAgentMemory struct{}

func (testAgentMemory) Session(_ string) axiom.SessionMemory { return testSessionMemory{} }
func (testAgentMemory) Search(_ context.Context, _ string, _ int) ([]axiom.MemoryEntry, error) {
	return nil, nil
}
func (testAgentMemory) Write(_ context.Context, _ string, _ float32) (string, error) {
	return "", nil
}

type testSessionMemory struct{}

func (testSessionMemory) Search(_ context.Context, _ string, _ int) ([]axiom.MemoryEntry, error) {
	return nil, nil
}
func (testSessionMemory) Write(_ context.Context, _ string, _ float32) (string, error) {
	return "", nil
}
func (testSessionMemory) History() axiom.SessionHistory { return testSessionHistory{} }
func (testSessionMemory) End(_ context.Context) error   { return nil }

type testSessionHistory struct{}

func (testSessionHistory) Last(_ context.Context, _ int) ([]axiom.ConversationTurn, error) {
	return nil, nil
}
func (testSessionHistory) Append(_ context.Context, _, _ string) error { return nil }

func (c *testContext) Log() axiom.Logger            { return &testLogger{c.t} }
func (c *testContext) Secrets() axiom.Secrets       { return testSecrets{c.secretsMap} }
func (c *testContext) Agent() axiom.Agent           { return testAgent{} }
func (c *testContext) ExecutionID() string          { return "test-execution-id" }
func (c *testContext) FlowID() string               { return "test-flow-id" }
func (c *testContext) TenantID() string             { return "test-tenant-id" }
func (c *testContext) Reflection() axiom.Reflection { return testReflection{} }
func (c *testContext) Mutation() axiom.Mutation     { return testMutation{} }

// testReflection/testMutation are no-op doubles for the two Context surfaces
// this node doesn't exercise (Echo neither inspects the running flow nor
// mutates it) — present only so *testContext satisfies axiom.Context.
type testReflection struct{}

func (testReflection) Flow() axiom.FlowReflection { return testFlowReflection{} }

type testFlowReflection struct{}

func (testFlowReflection) Nodes() []axiom.ReflectionNode     { return nil }
func (testFlowReflection) Edges() []axiom.ReflectionEdge     { return nil }
func (testFlowReflection) LoopEdges() []axiom.ReflectionEdge { return nil }
func (testFlowReflection) Position() axiom.FlowPosition      { return axiom.FlowPosition{} }
func (testFlowReflection) GraphID() string                   { return "test-graph-id" }

type testMutation struct{}

func (testMutation) Flow() axiom.FlowMutation { return testFlowMutation{} }

type testFlowMutation struct{}

func (testFlowMutation) AddNode(_, _ string, _ *axiom.CanvasPosition) uint32 { return 0 }
func (testFlowMutation) AddEdge(_, _ uint32, _ *axiom.EdgeCondition)         {}

// TESTS — delete this block when done ─────────────────────────────────────────
// Tests are required to push this package. The push pipeline runs your
// tests as a quality gate — a package will not be pushed if tests fail or
// do not meet the minimum requirements.
//
// Requirements checked before pushing:
//   - At least one test per node
//   - All tests must pass
//   - Output fields must be meaningfully asserted — not just error-checked
//
// The generated test below is a starting point. Replace the TODO comment with
// real assertions that verify your node returns correct data for known inputs.
// Think: given a specific input, what should the output fields contain?
//
// Run your tests locally at any time:
//   axiom test

// TestEcho simulates what the compiler's auto-derived facade->generic
// adapter actually does at runtime (ADR-118): it writes the Instance's
// bound input facade field as a raw, numbered proto wire field on
// EchoInputPlaceholder's unknown fields (Echo's own port declares none of
// its own). Echo must read that field STRUCTURALLY, by number, and emit the
// transformed value the same way — proving it never assumes a field name,
// only a field number, which is the only thing a truly generic node's code
// can rely on.
func TestEcho(t *testing.T) {
	ctx := context.Background()
	ax := newTestContext(t)

	var raw []byte
	raw = protowire.AppendTag(raw, 1, protowire.BytesType)
	raw = protowire.AppendString(raw, "hello")

	input := &gen.EchoInputPlaceholder{}
	input.ProtoReflect().SetUnknown(protoreflect.RawFields(raw))

	got, err := nodes.Echo(ctx, ax, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	num, typ, n := protowire.ConsumeTag(got.ProtoReflect().GetUnknown())
	if n < 0 {
		t.Fatalf("output carries no readable wire field: %v", protowire.ParseError(n))
	}
	if num != 1 || typ != protowire.BytesType {
		t.Fatalf("output field = (number=%d, type=%v), want (number=1, type=BytesType)", num, typ)
	}
	rest := got.ProtoReflect().GetUnknown()[n:]
	val, n := protowire.ConsumeBytes(rest)
	if n < 0 {
		t.Fatalf("output field value unreadable: %v", protowire.ParseError(n))
	}
	if string(val) != "echo: hello" {
		t.Errorf("output field value = %q, want %q", val, "echo: hello")
	}
}

var _ axiom.Context = (*testContext)(nil) // compile-time interface check
