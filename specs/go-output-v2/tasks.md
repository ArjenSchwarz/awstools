---
references:
    - specs/go-output-v2/requirements.md
    - specs/go-output-v2/design.md
    - specs/go-output-v2/decision_log.md
---
# go-output v2 Migration

## Setup

- [x] 1. Add go-output v2.7.0 dependency alongside v1 <!-- id:1hi7wj4 -->
  - go get github.com/ArjenSchwarz/go-output/v2@v2.7.0; run go mod tidy
  - v1 dependency and its imports stay until all command files are migrated (removed in task 26)
  - verify build still passes with both modules present
  - Stream: 1
  - Requirements: [1.1](requirements.md#1.1)

## Config core

- [x] 2. Write tests for config render helpers (red) <!-- id:1hi7wj5 -->
  - new tests in config/config_test.go alongside existing v1 tests (old tests removed in task 26)
  - formatFor: every format name + unknown-value JSON fallback + file-format fallback symmetry
  - NeedsGraphFormat/IsDrawIO: true when stdout OR effective file format matches
  - transformer/style/width wiring: emoji only when output.use-emoji; TableWithStyleAndMaxColumnWidth from output.table.* keys (width 0 = unlimited)
  - end-to-end RenderDocuments via injectable stdout writer: output.format=json + output.file-format=csv + temp file -> stdout parses as JSON, file parses as CSV
  - formats-match case: file bytes identical to captured stdout bytes
  - guard: dot/mermaid with nil Graph flavor and drawio with nil DrawIO flavor error with v1 message 'This command doesn't currently support the X output format'
  - append wiring: WithAppendMode set when output.append true, suppressed by WithFileOverwrite option
  - Blocked-by: 1hi7wj4 (Add go-output v2.7.0 dependency alongside v1)
  - Stream: 1
  - Requirements: [3.1](requirements.md#3.1), [3.2](requirements.md#3.2), [3.3](requirements.md#3.3), [3.6](requirements.md#3.6), [4.2](requirements.md#4.2), [4.3](requirements.md#4.3), [4.4](requirements.md#4.4), [4.6](requirements.md#4.6), [8.3](requirements.md#8.3), [9.2](requirements.md#9.2)

- [x] 3. Implement config render core (green) <!-- id:1hi7wj6 -->
  - config/config.go: DocumentSet{Table,Graph,DrawIO}, RenderDocuments(ctx,docs,opts...), RenderDocument sugar, WithFileOverwrite RenderOption, unexported renderDocuments core with injectable stdout writer, formatFor, NeedsGraphFormat, extended IsDrawIO, SortOption(column)
  - two Outputs (stdout format + effective file format), each rendering the flavor matching its format; sequential stdout-then-file; errors returned never fatal
  - FileWriter: NewFileWriterWithOptions(filepath.Dir, filepath.Base) - no {ext}/{format} placeholders
  - keep NewOutputSettings/GetSeparator and v1 import for now (deleted in task 26)
  - Blocked-by: 1hi7wj5 (Write tests for config render helpers (red)), helpers
  - Stream: 1
  - Requirements: [3.1](requirements.md#3.1), [3.2](requirements.md#3.2), [3.3](requirements.md#3.3), [3.4](requirements.md#3.4), [3.5](requirements.md#3.5), [3.6](requirements.md#3.6), [4.1](requirements.md#4.1), [4.2](requirements.md#4.2), [4.3](requirements.md#4.3), [4.4](requirements.md#4.4), [4.5](requirements.md#4.5), [4.6](requirements.md#4.6), [7.1](requirements.md#7.1), [9.1](requirements.md#9.1), [9.2](requirements.md#9.2)

## Shared cmd helpers

- [x] 4. Write graphEdges unit tests (red) <!-- id:1hi7wj7 -->
  - cmd/graph_test.go: pin v1 splitFromToValues semantics - []string to-cell produces one edge per element, comma-joined string splits, empty targets skipped, from-value stringified
  - verify exact v1 toString join for []string cells against v1 source and encode the result in the tests
  - Blocked-by: 1hi7wj4 (Add go-output v2.7.0 dependency alongside v1)
  - Stream: 1
  - Requirements: [2.3](requirements.md#2.3)

- [x] 5. Implement cmd/graph.go graphEdges (green) <!-- id:1hi7wj8 -->
  - graphEdges(rows []map[string]any, fromCol, toCol string) []output.Edge
  - Blocked-by: 1hi7wj7 (Write graphEdges unit tests (red))
  - Stream: 1
  - Requirements: [2.3](requirements.md#2.3)

- [x] 6. Write drawio adapter unit tests (red) <!-- id:1hi7wj9 -->
  - cmd/drawio_test.go: awsShape returns same style as v1 for known group/title; unknown shape returns empty string + stderr warning, no panic
  - drawIOConnection returns v1 NewConnection defaults: From=Parent, To=Name, Invert=true, Style=DrawIODefaultConnectionStyle
  - drawIOBaseHeader(label,style,ignore) matches v1 NewHeader + SetHeightAndWidth(78,78) shape
  - Blocked-by: 1hi7wj4 (Add go-output v2.7.0 dependency alongside v1)
  - Stream: 1
  - Requirements: [5.3](requirements.md#5.3)

- [x] 7. Implement cmd/drawio.go adapters (green) <!-- id:1hi7wja -->
  - awsShape wraps icons.GetAWSShape swallowing error; drawIOConnection; drawIOBaseHeader
  - Blocked-by: 1hi7wj9 (Write drawio adapter unit tests (red)), adapter
  - Stream: 1
  - Requirements: [5.3](requirements.md#5.3)

## Canonical table commands

- [x] 8. Migrate appmesh commands (danglingnodes, meshroute) <!-- id:1hi7wjb -->
  - cmd/appmeshdanglingnodes.go, cmd/appmeshmeshroute.go
  - pattern per design: rows []map[string]any, output.New().Table(title, rows, WithKeys...).Build(), settings.RenderDocument(cmd.Context(), doc)
  - RunE conversion: Run -> RunE, panic/log.Fatal in command body -> return err
  - meshroute keeps verbose-conditional keys
  - Blocked-by: 1hi7wj6 (Implement config render core (green))
  - Stream: 2
  - Requirements: [2.1](requirements.md#2.1), [2.6](requirements.md#2.6), [3.5](requirements.md#3.5), [9.1](requirements.md#9.1)

- [x] 9. Migrate sso + s3 canonical commands (s3list, ssodangling, ssolistpermissionsets, sso.go) <!-- id:1hi7wjc -->
  - cmd/s3list.go, cmd/ssodangling.go, cmd/ssolistpermissionsets.go, cmd/sso.go displayEnhancedProfileResults
  - ssolistpermissionsets + sso.go use SortOption for their v1 SortKey
  - RunE conversion throughout
  - Blocked-by: 1hi7wj6 (Implement config render core (green))
  - Stream: 3
  - Requirements: [2.1](requirements.md#2.1), [3.4](requirements.md#3.4), [3.5](requirements.md#3.5), [9.1](requirements.md#9.1)

- [x] 10. Migrate cfn + tgw + vpc canonical commands (cfnresources, tgwdangling, vpcroutes) <!-- id:1hi7wjd -->
  - cmd/cfnresources.go, cmd/tgwdangling.go, cmd/vpcroutes.go
  - cfnresources keeps verbose column additions
  - vpcroutes: raw []string cells adopt v2 default rendering per Decision 7; drawio remnants stay commented
  - RunE conversion throughout
  - Blocked-by: 1hi7wj6 (Implement config render core (green))
  - Stream: 4
  - Requirements: [2.1](requirements.md#2.1), [2.6](requirements.md#2.6), [3.5](requirements.md#3.5), [9.1](requirements.md#9.1)

## Graph and drawio commands

- [x] 11. Migrate appmeshshowmesh <!-- id:1hi7wje -->
  - DocumentSet: Table always; Graph flavor when NeedsGraphFormat (Name->Endpoints via graphEdges); DrawIO flavor when IsDrawIO with createAppmeshShowmeshDrawIOHeader converted to output.DrawIOHeader public fields
  - flavor construction additive, never exclusive if/else
  - RunE conversion
  - Blocked-by: 1hi7wj6 (Implement config render core (green)), 1hi7wj8 (Implement cmd/graph.go graphEdges (green)), 1hi7wja (Implement cmd/drawio.go adapters (green))
  - Stream: 2
  - Requirements: [2.3](requirements.md#2.3), [5.1](requirements.md#5.1), [5.2](requirements.md#5.2), [9.1](requirements.md#9.1)

- [x] 12. Migrate iamrolelist + iamuserlist <!-- id:1hi7wjf -->
  - heterogeneous rows preserved: one table, union WithKeys, absent cells empty
  - graph: Name->Policies / Name->Groups; drawio headers use Identity=DrawioID, Layout=DrawIOLayoutHorizontalFlow; iamuserlist has two connections in verbose
  - rows for drawio declared []output.Record (defined type, no implicit conversion)
  - RunE conversion
  - Blocked-by: 1hi7wj6 (Implement config render core (green)), 1hi7wj8 (Implement cmd/graph.go graphEdges (green)), 1hi7wja (Implement cmd/drawio.go adapters (green))
  - Stream: 2
  - Requirements: [2.3](requirements.md#2.3), [2.6](requirements.md#2.6), [5.1](requirements.md#5.1), [5.2](requirements.md#5.2), [9.1](requirements.md#9.1)

- [x] 13. Migrate organizationsstructure <!-- id:1hi7wjg -->
  - traverseOrgStructureEntry signature: appends rows to a slice (or takes builder) instead of *format.OutputArray
  - DefaultDrawIOHeader + Layout=DrawIOLayoutVerticalTree; graph Name->Children
  - RunE conversion (existing log.Fatal at line 37 -> return err)
  - Blocked-by: 1hi7wj6 (Implement config render core (green)), 1hi7wj8 (Implement cmd/graph.go graphEdges (green)), 1hi7wja (Implement cmd/drawio.go adapters (green))
  - Stream: 3
  - Requirements: [2.3](requirements.md#2.3), [5.1](requirements.md#5.1), [5.2](requirements.md#5.2), [9.1](requirements.md#9.1)

- [x] 14. Migrate ssooverviewaccount + ssooverviewpermissionset <!-- id:1hi7wjh -->
  - v1 Keys-reassignment-mid-build becomes two explicit DocumentSet flavors: table keys vs drawio records built by createSSO*DrawIOContents helpers (now returning []output.Record)
  - drawio: DefaultDrawIOHeader + Layout=DrawIOLayoutHorizontalTree; graph DrawioID->Children
  - additive flavor construction; RunE conversion
  - Blocked-by: 1hi7wj6 (Implement config render core (green)), 1hi7wj8 (Implement cmd/graph.go graphEdges (green)), 1hi7wja (Implement cmd/drawio.go adapters (green))
  - Stream: 3
  - Requirements: [2.3](requirements.md#2.3), [3.4](requirements.md#3.4), [5.1](requirements.md#5.1), [5.2](requirements.md#5.2), [9.1](requirements.md#9.1)

- [x] 15. Migrate tgwroutes (both render paths) <!-- id:1hi7wji -->
  - main path AND simplelistOnly path must each populate the Graph flavor when NeedsGraphFormat holds (design invariant: guard must not fire on a capable command's alternate path)
  - graph Destinations->ID; drawio header has two connections
  - RunE conversion
  - Blocked-by: 1hi7wj6 (Implement config render core (green)), 1hi7wj8 (Implement cmd/graph.go graphEdges (green)), 1hi7wja (Implement cmd/drawio.go adapters (green))
  - Stream: 4
  - Requirements: [2.3](requirements.md#2.3), [3.4](requirements.md#3.4), [5.1](requirements.md#5.1), [5.2](requirements.md#5.2), [9.1](requirements.md#9.1)

- [x] 16. Migrate tgwoverview (drawio + combine + emoji-in-data) <!-- id:1hi7wjj -->
  - combine port: drawio.GetHeaderAndContentsFromFile -> output.ParseDrawIOFile keyed records; ID-keyed merge logic unchanged; pass WithFileOverwrite() when ShouldCombineAndAppend was true
  - emoji: read settings.GetBool(output.use-emoji) instead of Settings.UseEmoji for the manual checkmark/cross prefixes
  - R2.8: targetTgwMapping map iteration (tgwoverview.go:168) needs deterministic order - sort keys before ranging
  - BidirectionalConnectionStyle -> output.DrawIOBidirectionalConnectionStyle
  - RunE conversion
  - Blocked-by: 1hi7wj6 (Implement config render core (green)), 1hi7wja (Implement cmd/drawio.go adapters (green))
  - Stream: 4
  - Requirements: [2.8](requirements.md#2.8), [5.1](requirements.md#5.1), [5.4](requirements.md#5.4), [9.1](requirements.md#9.1)

- [x] 17. Migrate vpcpeerings (graph + drawio + combine) <!-- id:1hi7wjk -->
  - combine port to ParseDrawIOFile keyed records preserving unique() dedup; WithFileOverwrite() when merged
  - keeps raw connection style string curved=1;endArrow=none;endFill=1;fontSize=11; (R5.2)
  - graph ID->PeeringIDs (multi-value cells exercise graphEdges split)
  - RunE conversion
  - Blocked-by: 1hi7wj6 (Implement config render core (green)), 1hi7wj8 (Implement cmd/graph.go graphEdges (green)), 1hi7wja (Implement cmd/drawio.go adapters (green))
  - Stream: 4
  - Requirements: [2.3](requirements.md#2.3), [5.1](requirements.md#5.1), [5.2](requirements.md#5.2), [5.4](requirements.md#5.4), [9.1](requirements.md#9.1)

## Special cases

- [x] 18. Rewrite vpcenis tests against single-Document flow (red) <!-- id:1hi7wjl -->
  - cmd/vpcenis_test.go currently encodes v1 buffer semantics (T-1294 workaround)
  - preserve test intent: file output contains ALL per-subnet tables; stdout equals file when formats match (R4.6)
  - tests must fail against the unmigrated vpcenis
  - Blocked-by: 1hi7wj6 (Implement config render core (green))
  - Stream: 2
  - Requirements: [7.2](requirements.md#7.2)

- [x] 19. Migrate vpcenis (green) <!-- id:1hi7wjm -->
  - delete AddToBuffer/combined-array choreography; one builder, one .Table per subnet group with own WithKeys, single RenderDocument
  - keep enisGraphFormatError guard unchanged (fires pre-build, central guard unreachable, message preserved)
  - RunE conversion
  - Blocked-by: 1hi7wjl (Rewrite vpcenis tests against single-Document flow (red)), vpcenis, against
  - Stream: 2
  - Requirements: [7.2](requirements.md#7.2), [9.1](requirements.md#9.1), [9.2](requirements.md#9.2)

- [ ] 20. Migrate vpcoverview (multi-table single render) <!-- id:1hi7wjn -->
  - three per-loop OutputArray+Write blocks -> one Document: per-VPC subnet tables + per-subnet IP tables + summary table, each .Table with own WithKeys
  - known accepted change: one JSON document instead of v1 concatenated JSON fragments (design/D5)
  - RunE conversion
  - Blocked-by: 1hi7wj6 (Implement config render core (green))
  - Stream: 3
  - Requirements: [2.7](requirements.md#2.7), [7.3](requirements.md#7.3), [9.1](requirements.md#9.1)

- [ ] 21. Migrate demotables with style-drift test <!-- id:1hi7wjo -->
  - hardcode the 16 v1 style names; loop renders one single-table Document per style via inline Output with output.TableWithStyle(name)
  - --file support intentionally dropped (demo command, per design)
  - test: every hardcoded name accepted by TableWithStyle (drift guard)
  - RunE conversion
  - Blocked-by: 1hi7wj6 (Implement config render core (green))
  - Stream: 4
  - Requirements: [7.4](requirements.md#7.4)

- [x] 22. Migrate names + organizationsnames (drop PrintByteSlice/S3) <!-- id:1hi7wjp -->
  - keep json.Marshal + merge-on-append via GetStringMapFromJSONFile
  - replace format.PrintByteSlice with v1-exact semantics: write output.file when set, OTHERWISE print to stdout (one destination, not both)
  - S3 argument dropped entirely (dead path, D6)
  - RunE conversion (log.Fatal -> return err)
  - only needs v2 dep, not config core - can run early
  - Blocked-by: 1hi7wj4 (Add go-output v2.7.0 dependency alongside v1)
  - Stream: 4
  - Requirements: [6.1](requirements.md#6.1), [6.2](requirements.md#6.2), [6.3](requirements.md#6.3), [9.1](requirements.md#9.1)

## Verification and cleanup

- [ ] 23. Write per-format equivalence oracle tests <!-- id:1hi7wjq -->
  - fixed in-memory rows, no AWS calls; JSON representative must render through go-output, not names (R8.1)
  - json/yaml: unmarshal deep-equal + key order via json.Decoder token stream
  - csv: header order + scalar cells; multi-value/bool cells asserted against v2 defaults (D7)
  - table/markdown: column headers + row values in order, styling-agnostic
  - dot/mermaid: exact node set + directed edge set
  - drawio: ParseDrawIOCSV round-trip of nodes/shapes/connections
  - html: data present within DefaultHTMLTemplate document
  - empty-result case per format renders valid output (R9.3)
  - Blocked-by: 1hi7wjb (Migrate appmesh commands (danglingnodes, meshroute)), appmesh, 1hi7wjc (Migrate sso + s3 canonical commands (s3list, ssodangling, ssolistpermissionsets, sso.go)), 1hi7wjd (Migrate cfn + tgw + vpc canonical commands (cfnresources, tgwdangling, vpcroutes)), 1hi7wje (Migrate appmeshshowmesh), 1hi7wjf (Migrate iamrolelist + iamuserlist), 1hi7wjg (Migrate organizationsstructure), 1hi7wjh (Migrate ssooverviewaccount + ssooverviewpermissionset), 1hi7wji (Migrate tgwroutes (both render paths)), 1hi7wjj (Migrate tgwoverview (drawio + combine + emoji-in-data)), combine, 1hi7wjk (Migrate vpcpeerings (graph + drawio + combine)), 1hi7wjm (Migrate vpcenis (green)), vpcenis, 1hi7wjn (Migrate vpcoverview (multi-table single render)), 1hi7wjo (Migrate demotables with style-drift test), 1hi7wjp (Migrate names + organizationsnames (drop PrintByteSlice/S3))
  - Stream: 1
  - Requirements: [2.1](requirements.md#2.1), [2.2](requirements.md#2.2), [2.3](requirements.md#2.3), [2.4](requirements.md#2.4), [2.5](requirements.md#2.5), [2.7](requirements.md#2.7), [8.1](requirements.md#8.1), [9.3](requirements.md#9.3)

- [ ] 24. Write verbose-dimension equivalence test <!-- id:1hi7wjr -->
  - exercise one command's row/key builder verbose on and off (cfnresources or iamuserlist), assert the column/row delta
  - Blocked-by: 1hi7wjd (Migrate cfn + tgw + vpc canonical commands (cfnresources, tgwdangling, vpcroutes)), 1hi7wjf (Migrate iamrolelist + iamuserlist)
  - Stream: 1
  - Requirements: [2.6](requirements.md#2.6), [8.2](requirements.md#8.2)

- [ ] 25. Write drawio combine-and-append test <!-- id:1hi7wjs -->
  - write drawio CSV via v2, run the ported merge logic plus new records, assert combined ID set, dedup, and column order
  - Blocked-by: 1hi7wjj (Migrate tgwoverview (drawio + combine + emoji-in-data)), combine, 1hi7wjk (Migrate vpcpeerings (graph + drawio + combine))
  - Stream: 1
  - Requirements: [5.4](requirements.md#5.4), [8.4](requirements.md#8.4)

- [ ] 26. Remove v1 dependency and finish cleanup <!-- id:1hi7wjt -->
  - delete config.NewOutputSettings, dead GetSeparator, their old tests, and all remaining v1 imports
  - go.mod/go.sum: drop github.com/ArjenSchwarz/go-output v1.4.0 + delete stale commented replace ../go-output2 line
  - grep-verify zero references to ArjenSchwarz/go-output without /v2
  - go fmt, go test ./..., make test all pass
  - Blocked-by: 1hi7wjq (Write per-format equivalence oracle tests), 1hi7wjr (Write verbose-dimension equivalence test), 1hi7wjs (Write drawio combine-and-append test)
  - Stream: 1
  - Requirements: [1.2](requirements.md#1.2), [1.3](requirements.md#1.3), [1.4](requirements.md#1.4)
