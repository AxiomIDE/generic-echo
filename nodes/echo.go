package nodes

import (
	"context"
	"sort"

	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/reflect/protoreflect"

	"axiom-official/generic-echo/axiom"
	gen "axiom-official/generic-echo/gen"
)

// Echo is a GENERIC (ADR-118) node: both its input and output are field-less
// placeholder messages (EchoInputPlaceholder / EchoOutputPlaceholder), because
// a truly generic node cannot know an Instance's real field names at publish
// time. An Instance binds each port to a real, named facade message via
// `axiom instance create --input-message ... --input-field name=kind
// --output-message ... --output-field name=kind`; at compile time the
// registry auto-derives adapters that reshape the facade's real, named
// field values onto/from THIS message's raw wire bytes, keyed by field
// NUMBER only (deterministic sorted-name numbering) — never by name, since
// this message declares none. Echo's job is exactly what a real generic
// connector's job would be: walk whatever numbered string fields are
// present, structurally, and produce the same shape back out. It never
// assumes a specific field exists — decodeFields/encodeFields are the only
// thing this node knows how to do, regardless of what an Instance names its
// facade fields.
func Echo(ctx context.Context, ax axiom.Context, input *gen.EchoInputPlaceholder) (*gen.EchoOutputPlaceholder, error) {
	fields, err := decodeFields(input.ProtoReflect().GetUnknown())
	if err != nil {
		return nil, err
	}

	out := make(map[protowire.Number]string, len(fields))
	for num, val := range fields {
		out[num] = "echo: " + val
	}

	output := &gen.EchoOutputPlaceholder{}
	output.ProtoReflect().SetUnknown(protoreflect.RawFields(encodeFields(out)))
	return output, nil
}

// decodeFields structurally walks raw, LEN-encoded (string) proto wire fields
// by NUMBER — the only thing a truly generic node's code can rely on, since
// its own published port declares no fields of its own. Any non-string
// (non-bytes-wire-type) field is skipped rather than erroring: a generic
// node processes structurally, it doesn't reject shapes it doesn't recognize.
func decodeFields(raw []byte) (map[protowire.Number]string, error) {
	fields := make(map[protowire.Number]string)
	for len(raw) > 0 {
		num, typ, n := protowire.ConsumeTag(raw)
		if n < 0 {
			return nil, protowire.ParseError(n)
		}
		raw = raw[n:]
		switch typ {
		case protowire.BytesType:
			v, n := protowire.ConsumeBytes(raw)
			if n < 0 {
				return nil, protowire.ParseError(n)
			}
			fields[num] = string(v)
			raw = raw[n:]
		default:
			n := protowire.ConsumeFieldValue(num, typ, raw)
			if n < 0 {
				return nil, protowire.ParseError(n)
			}
			raw = raw[n:]
		}
	}
	return fields, nil
}

// encodeFields is the inverse of decodeFields: renders a field-number->string
// map back into raw wire bytes, sorted by field number for deterministic
// output (matching how the registry's synthesized facades number their own
// fields).
func encodeFields(fields map[protowire.Number]string) []byte {
	nums := make([]protowire.Number, 0, len(fields))
	for num := range fields {
		nums = append(nums, num)
	}
	sort.Slice(nums, func(i, j int) bool { return nums[i] < nums[j] })

	var b []byte
	for _, num := range nums {
		b = protowire.AppendTag(b, num, protowire.BytesType)
		b = protowire.AppendString(b, fields[num])
	}
	return b
}
