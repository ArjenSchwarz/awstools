package cmd

import (
	"bytes"
	"io"
	"testing"

	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/stretchr/testify/assert"
)

// TestAWSShape_KnownShape pins the style string awsShape must return for a
// known group/title. The expected value is v1's drawio/shapes/aws.json entry
// for Containers/Container, verified byte-identical in go-output v2.7.0's
// icons/aws.json, so the migration keeps emitting v1 shape styles (R5.3).
func TestAWSShape_KnownShape(t *testing.T) {
	want := "outlineConnect=0;fontColor=#232F3E;gradientColor=none;fillColor=#D05C17;strokeColor=none;dashed=0;verticalLabelPosition=bottom;verticalAlign=top;align=center;html=1;fontSize=12;fontStyle=0;aspect=fixed;pointerEvents=1;shape=mxgraph.aws4.container_3;"
	assert.Equal(t, want, awsShape("Containers", "Container"))
}

// TestAWSShape_AllCommandShapesResolve guards every group/title pair the
// commands request: each must resolve to a non-empty style in v2's shape set
// (verified byte-identical to v1's shapes/aws.json), so no command silently
// loses its icon in the migration.
//
// One command pair is deliberately absent: "Network Content Delivery"/"Direct
// Connect Gateway" (tgw routes) is missing from v1's shape set too, so v1
// already rendered that node without an icon — pinned separately below.
func TestAWSShape_AllCommandShapesResolve(t *testing.T) {
	pairs := [][2]string{
		{"Containers", "Container"},
		{"General Resources", "General"},
		{"General Resources", "User"},
		{"General Resources", "Users"},
		{"Management Governance", "Organizational Unit"},
		{"Management Governance", "Organizations"},
		{"Network Content Delivery", "Direct Connect"},
		{"Network Content Delivery", "Peering Connection"},
		{"Network Content Delivery", "Site-to-Site VPN"},
		{"Network Content Delivery", "Transit Gateway"},
		{"Security Identity Compliance", "Organizations Account"},
		{"Security Identity Compliance", "Permissions"},
		{"Security Identity Compliance", "Role"},
		{"Security Identity Compliance", "Single Sign-On"},
	}
	for _, pair := range pairs {
		assert.NotEmpty(t, awsShape(pair[0], pair[1]), "shape %s/%s must resolve", pair[0], pair[1])
	}
}

// TestAWSShape_DirectConnectGatewayStaysEmpty pins v1 parity for the one
// shape a command requests that exists in neither v1's nor v2's shape set:
// v1 drawio.AWSShape returned "" for it, and awsShape must keep doing so.
func TestAWSShape_DirectConnectGatewayStaysEmpty(t *testing.T) {
	original := shapeWarningWriter
	shapeWarningWriter = io.Discard
	t.Cleanup(func() { shapeWarningWriter = original })

	assert.Empty(t, awsShape("Network Content Delivery", "Direct Connect Gateway"))
}

// TestAWSShape_UnknownShape asserts v1 drawio.AWSShape's lenient behavior: an
// unknown group/title returns an empty style without panicking, and a warning
// naming the shape goes to the stderr seam (R5.3).
func TestAWSShape_UnknownShape(t *testing.T) {
	tests := map[string]struct {
		group string
		title string
	}{
		"unknown group":                  {group: "No Such Group", title: "Container"},
		"unknown title in a known group": {group: "Containers", title: "No Such Shape"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			original := shapeWarningWriter
			shapeWarningWriter = &buf
			t.Cleanup(func() { shapeWarningWriter = original })

			got := awsShape(tc.group, tc.title)

			assert.Empty(t, got)
			assert.Contains(t, buf.String(), tc.group)
			assert.Contains(t, buf.String(), tc.title)
		})
	}
}

// TestDrawIOConnection_V1Defaults pins v1 drawio.NewConnection's defaults:
// {From: "Parent", To: "Name", Invert: true, Style: DefaultConnectionStyle}.
func TestDrawIOConnection_V1Defaults(t *testing.T) {
	want := output.DrawIOConnection{
		From:   "Parent",
		To:     "Name",
		Invert: true,
		Label:  "",
		Style:  output.DrawIODefaultConnectionStyle,
	}
	assert.Equal(t, want, drawIOConnection())

	// Pin the style bytes to v1's DefaultConnectionStyle verbatim, so a drift
	// in v2's constant cannot silently change the emitted diagrams.
	assert.Equal(t, "curved=1;endArrow=blockThin;endFill=1;fontSize=11;", drawIOConnection().Style)
}

// TestDrawIOBaseHeader_V1Shape pins the shared header shape every command's
// create*DrawIOHeader builds on: v1 drawio.NewHeader(label, style, ignore)
// (layout auto, spacing 40/100/40, namespace csvimport-) followed by
// SetHeightAndWidth("78", "78").
func TestDrawIOBaseHeader_V1Shape(t *testing.T) {
	got := drawIOBaseHeader("%Name%<br>%Type%", "%Image%", "Image,Type")
	want := output.DrawIOHeader{
		Label:        "%Name%<br>%Type%",
		Style:        "%Image%",
		Ignore:       "Image,Type",
		Layout:       output.DrawIOLayoutAuto,
		NodeSpacing:  40,
		LevelSpacing: 100,
		EdgeSpacing:  40,
		Height:       "78",
		Width:        "78",
		Namespace:    "csvimport-",
	}
	assert.Equal(t, want, got)
}
