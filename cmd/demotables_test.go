package cmd

import (
	"bytes"
	"context"
	"strings"
	"testing"

	output "github.com/ArjenSchwarz/go-output/v2"
)

// styleCaptureWriter implements output.Writer, buffering rendered bytes so the
// style-drift tests can inspect what TableWithStyle produced.
type styleCaptureWriter struct {
	buf bytes.Buffer
}

func (w *styleCaptureWriter) Write(_ context.Context, _ string, data []byte) error {
	_, err := w.buf.Write(data)
	return err
}

// renderWithStyle renders the demo document through an inline Output built
// with output.TableWithStyle(name), mirroring how demoTables renders, and
// returns the produced bytes.
func renderWithStyle(t *testing.T, name string) string {
	t.Helper()
	w := &styleCaptureWriter{}
	out := output.NewOutput(
		output.WithFormat(output.TableWithStyle(name)),
		output.WithWriter(w),
	)
	if err := out.Render(t.Context(), demoTableDocument()); err != nil {
		t.Fatalf("rendering with style %q failed: %v", name, err)
	}
	return w.buf.String()
}

// TestDemoTableStylesAcceptedByV2 guards demoTableStyles against drifting from
// the style names go-output v2 accepts. v2 silently falls back to the default
// box style for unknown names, so "renders without error" alone would not
// catch drift: every hardcoded name (except Default, which is
// indistinguishable from the fallback by construction) must render differently
// from what an unknown style name produces.
func TestDemoTableStylesAcceptedByV2(t *testing.T) {
	fallback := renderWithStyle(t, "NoSuchStyleName")

	for _, name := range demoTableStyles {
		t.Run(name, func(t *testing.T) {
			got := renderWithStyle(t, name)
			if !strings.Contains(got, "demo-s3-bucket") {
				t.Errorf("style %q output is missing expected table data:\n%s", name, got)
			}
			if name != "Default" && got == fallback {
				t.Errorf("style %q rendered identically to the unknown-style fallback; the name is no longer recognised by go-output v2", name)
			}
		})
	}
}

// TestDemoTableStylesMatchV1List asserts the hardcoded list still covers all
// 16 style names v1 exposed via format.TableStyles, with no duplicates.
func TestDemoTableStylesMatchV1List(t *testing.T) {
	const wantCount = 16
	if len(demoTableStyles) != wantCount {
		t.Fatalf("expected %d hardcoded v1 style names, got %d", wantCount, len(demoTableStyles))
	}
	seen := make(map[string]bool, len(demoTableStyles))
	for _, name := range demoTableStyles {
		if seen[name] {
			t.Errorf("duplicate style name %q", name)
		}
		seen[name] = true
	}
}
