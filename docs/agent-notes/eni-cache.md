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
- No pagination is used for VPC endpoint and NAT gateway API calls. If a VPC has more resources than the default page size, results may be truncated.
