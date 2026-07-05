package cmd

import (
	"maps"
	"slices"
	"strings"

	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/spf13/cobra"
)

// tgwoverviewCmd represents the tgwoverview command
var tgwoverviewCmd = &cobra.Command{
	Use:   "overview",
	Short: "A basic overview of the Transit Gateway",
	Long: `Provides an overview of all the route tables and routes in the Transit Gateway.
This can be improved on, but offers a simple text based overview with all relevant information

If you choose the drawio output instead, you get a simple diagram showing the Transit Gateway and all resources (VPCs, VPNs, Direct Connect) attached to it.
	`,
	RunE: tgwoverview,
}

var excludeRouteTarget string
var includeBlackhole bool

func init() {
	tgwCmd.AddCommand(tgwoverviewCmd)
	tgwoverviewCmd.Flags().StringVarP(&excludeRouteTarget, "exclude-target", "e", "", "Optional value to exclude a specific target from the output")
	tgwoverviewCmd.Flags().BoolVarP(&includeBlackhole, "blackhole-routes", "b", false, "Optional value to include blackhole routes")
}

func tgwoverview(cmd *cobra.Command, _ []string) error {
	awsConfig := config.DefaultAwsConfig(*settings)
	resultTitle := "Transit Gateway Routes in account " + getName(helpers.GetAccountID(awsConfig.StsClient()))
	gateways := helpers.GetAllTransitGateways(awsConfig.Ec2Client())
	keys := []string{"Transit Gateway Account", "Transit Gateway", routeTableColumn, cidrColumn, "Target", "Target Type", "State"}
	useEmoji := settings.UseEmoji()

	rows := []map[string]any{}
	for _, gateway := range gateways {
		// Sort the route table keys before ranging so row order is
		// deterministic between runs (R2.8).
		for _, routetableID := range slices.Sorted(maps.Keys(gateway.RouteTables)) {
			routetable := gateway.RouteTables[routetableID]
			// Track which resources appear as route targets so we can
			// identify source attachments that are otherwise invisible.
			routeTargets := make(map[string]bool)
			for _, route := range routetable.Routes {
				if excludeRouteTarget == route.Attachment.ResourceID {
					continue
				}
				if !includeBlackhole && route.State == "blackhole" {
					continue
				}
				if route.Attachment.ResourceID != "" {
					routeTargets[route.Attachment.ResourceID] = true
				}
				content := make(map[string]any)
				content["Transit Gateway Account"] = getNameWithID(gateway.AccountID)
				content["Transit Gateway"] = getNameWithID(gateway.ID)
				content[routeTableColumn] = getNameWithID(routetable.ID)
				content[cidrColumn] = route.CIDR
				if route.Attachment.ResourceID != "" {
					content["Target"] = getNameWithID(route.Attachment.ResourceID)
				} else {
					content["Target"] = ""
				}
				content["Target Type"] = helpers.TypeByResourceID(route.Attachment.ResourceID)
				state := route.State
				if useEmoji {
					if route.State == "blackhole" {
						state = "❌ " + state
					} else {
						state = "✅ " + state
					}
				}
				content["State"] = state
				rows = append(rows, content)
			}
			// Show source attachments (associations) that don't already
			// appear as route targets so non-VPC attachments are visible.
			for _, attachment := range routetable.SourceAttachments {
				if attachment.ResourceID == "" || routeTargets[attachment.ResourceID] {
					continue
				}
				content := make(map[string]any)
				content["Transit Gateway Account"] = getNameWithID(gateway.AccountID)
				content["Transit Gateway"] = getNameWithID(gateway.ID)
				content[routeTableColumn] = getNameWithID(routetable.ID)
				content[cidrColumn] = "-"
				content["Target"] = getNameWithID(attachment.ResourceID)
				content["Target Type"] = attachment.ResourceType
				content["State"] = "associated"
				rows = append(rows, content)
			}
		}
	}

	docs := config.DocumentSet{
		Table: output.New().
			Table(resultTitle, rows, output.WithKeys(keys...), config.SortOption(routeTableColumn)).
			Build(),
	}
	merged := false
	if settings.IsDrawIO() {
		records, didMerge, err := createTgwOverviewDrawIORecords(gateways)
		if err != nil {
			return err
		}
		merged = didMerge
		docs.DrawIO = output.New().
			DrawIO(resultTitle, records, createTgwOverviewDrawIOHeader()).
			Build()
	}
	var opts []config.RenderOption
	if merged {
		// The prior file contents are already merged into the document, so
		// the file must be written fresh instead of appended to.
		opts = append(opts, config.WithFileOverwrite())
	}
	return settings.RenderDocuments(cmd.Context(), docs, opts...)
}

func createTgwOverviewDrawIOHeader() output.DrawIOHeader {
	drawioheader := drawIOBaseHeader("%Name%", "%Image%", imageColumn)
	connection := drawIOConnection()
	connection.From = destinationsColumn
	connection.To = "ID"
	connection.Invert = false
	connection.Style = output.DrawIOBidirectionalConnectionStyle
	drawioheader.Connections = append(drawioheader.Connections, connection)
	return drawioheader
}

// createTgwOverviewDrawIORecords builds the drawio node records for the
// overview diagram: one node per Transit Gateway plus one per attached
// resource, with each resource's Destinations pointing at its gateway(s).
// When combine-and-append is active the prior file's records are read back
// first (keyed by ID, as v1 did positionally) so the diagram combines prior
// and new data; the returned bool reports whether that merge happened, in
// which case the caller must write the file fresh instead of appending.
func createTgwOverviewDrawIORecords(gateways []helpers.TransitGateway) ([]output.Record, bool, error) {
	type targetTgwMap struct {
		ID           string
		Name         string
		Destinations []string
		Image        string
	}
	targetTgwMapping := make(map[string]targetTgwMap)
	merged := false
	if settings.ShouldCombineAndAppend() {
		parsed, err := output.ParseDrawIOFile(settings.GetString("output.file"))
		if err != nil {
			return nil, false, err
		}
		for _, record := range parsed.Records {
			id := drawIORecordString(record, "ID")
			targetTgwMapping[id] = targetTgwMap{
				ID:           id,
				Name:         drawIORecordString(record, nameColumn),
				Destinations: strings.Split(drawIORecordString(record, destinationsColumn), ","),
				Image:        drawIORecordString(record, imageColumn),
			}
		}
		merged = true
	}
	for _, gateway := range gateways {
		targetTgwMapping[gateway.ID] = targetTgwMap{
			ID:    gateway.ID,
			Name:  gateway.Name,
			Image: awsShape("Network Content Delivery", "Transit Gateway"),
		}
		attachedresources, _ := filterGateway([]helpers.TransitGateway{gateway})
		for _, resourceid := range slices.Sorted(maps.Keys(attachedresources)) {
			destinations := []string{gateway.ID}
			if val, ok := targetTgwMapping[resourceid]; ok {
				destinations = unique(append(destinations, val.Destinations...))
			}
			image := ""
			switch helpers.TypeByResourceID(resourceid) {
			case vpcResourceType:
				image = awsShape("Network Content Delivery", vpcColumn)
			case "vpn":
				image = awsShape("Network Content Delivery", "Site-to-Site VPN")
			case "dxgw":
				image = awsShape("Network Content Delivery", "Direct Connect Gateway")
			case tgwResourceType:
				image = awsShape("Network Content Delivery", "Transit Gateway")
			}
			targetTgwMapping[resourceid] = targetTgwMap{
				ID:           resourceid,
				Name:         getName(resourceid),
				Destinations: destinations,
				Image:        image,
			}
		}
	}
	records := make([]output.Record, 0, len(targetTgwMapping))
	// Sort the map keys before ranging so record order is deterministic
	// between runs (R2.8).
	for _, id := range slices.Sorted(maps.Keys(targetTgwMapping)) {
		mapping := targetTgwMapping[id]
		records = append(records, output.Record{
			"ID":               mapping.ID,
			nameColumn:         mapping.Name,
			destinationsColumn: strings.Join(mapping.Destinations, ","),
			imageColumn:        mapping.Image,
		})
	}
	return records, merged, nil
}
