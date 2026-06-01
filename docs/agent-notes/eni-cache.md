# ENI Lookup Cache

## Architecture

`ENILookupCache` (`helpers/ec2.go`) is a pre-populated cache that avoids repeated AWS API calls when processing ENI listings. It's created by `NewENILookupCache` which collects unique VPC IDs and instance IDs from a set of ENIs, then batch-fetches related resources.

### Cache Maps

- `EndpointsByENI` — maps ENI ID to `*types.VpcEndpoint`
- `NATGatewaysByENI` — maps ENI ID to `*types.NatGateway`
- `InstanceNames` — maps Instance ID to name string
- `TransitGateways` — maps VPC ID to TGW attachment ID string
- `VPCEndpoints` / `NATGateways` — maps VPC ID to resource pointer

### Consumers

Three functions use the cache for ENI detail resolution:
- `getENIUsageTypeOptimized` — checks `EndpointsByENI` for type classification
- `getENIAttachmentDetailsOptimized` — extracts service name from endpoints, NAT gateway name/ID
- `getResourceNameAndID` — returns endpoint/NAT gateway IDs for resource identification

## Gotchas

- Pointer storage pattern: When storing pointers from range loops into maps, use `&slice[i]` (index-based) rather than `&loopVar`. The range value variable is a copy; while Go 1.22+ creates per-iteration copies, the index-based pattern is clearer and version-independent.
- `batchFetchVPCEndpoints` and `batchFetchNATGateways` use `panic(err)` on API failure — these should eventually be converted to return errors.
- Output write path (T-1294): `cmd/vpcenis.go` no longer uses the `AddToBuffer()` + empty-outer-`Write()` pattern. That pattern wrote ENI rows to go-output's shared package `buffer` but called `Write()` on an empty outer `OutputArray`; in go-output v1.4.0 `Write()` prints the buffer to stdout, calls `buffer.Reset()`, then runs the `--file` branch — which saw an empty buffer and serialized the empty array, so `--file` got `[]`. Fixed by `buildENIOutput` (testable pure builder taking an `attachmentFor func(types.NetworkInterface) string` resolver to avoid AWS clients), `writeENIs` (non-split: `Write()` directly on the populated array), and `writeENIsBySubnet` (split: buffer per-subnet tables for stdout, but call the final `Write()` on a `combined` populated array so the `--file` branch has real data). Note `PrintByteSlice` uses `os.Create` (truncates), so per-group `Write()` calls cannot be used for split file output. Regression tests: `TestEnisFileOutput_NonSplit_NotEmpty_T1294`, `TestEnisFileOutput_Split_NotEmpty_T1294`.
- Pagination: both the batch cache fetchers (`batchFetchVPCEndpoints`, `batchFetchNATGateways`, `batchFetchTransitGateways`) and the per-ENI helpers (`GetVPCEndpointFromNetworkInterface`, `GetNatGatewayFromNetworkInterface`, `GetTransitGatewayFromNetworkInterface`) walk every page via `NewDescribe*Paginator`. T-657 fixed the per-ENI helpers — a matching resource on page 2+ previously looked unattached. T-705 then widened the exported helpers from `*ec2.Client` to the narrower `ec2.Describe*APIClient` interface and added a command-side regression test in `cmd/vpcenis_test.go` (using the composite `eniAttachmentLookupClient` in `cmd/vpcenis.go`) so a future refactor of `getAttachment` cannot silently bypass pagination.
