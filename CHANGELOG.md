# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Shared cmd helpers for the go-output v2 migration (Shared cmd helpers phase): `graphEdges` replicating v1 edge-splitting semantics (with a documented nil-cell cleanup, Decision 10) and drawio adapters (`awsShape`, `drawIOConnection`, `drawIOBaseHeader`) matching v1 defaults, all with pinning tests
- Central go-output v2 render helper in the config package (go-output-v2 migration, Config core phase): `DocumentSet`, `RenderDocuments`/`RenderDocument` with per-destination format dispatch, file append/overwrite wiring, emoji transformer, and v1-compatible unsupported-format guards, with a full test suite
- go-output v2.7.0 dependency alongside v1 (go-output-v2 migration, Setup phase); pulls minor AWS SDK updates (aws-sdk-go-v2 v1.39.3, service/s3 v1.88.5, smithy-go v1.23.1)
- Specification for the go-output v2.7.0 migration (`specs/go-output-v2/`): requirements, design, decision log, and task list covering the switch from go-output v1.4.0 to v2 across all commands
- Makefile targets for code quality: `fmt`, `vet`, `modernize`, `check`, `security-scan`
- Makefile targets for testing: `test-verbose`, `test-coverage`
- Makefile targets for dependency management: `deps-tidy`, `deps-update`
- Makefile target `install` for installing the application
- Organized help output with categorized sections
- Direct Connect Gateway support for name resolution in EC2 resources
- Support for additional Transit Gateway attachment types in DrawIO output (peering, direct-connect-gateway, connect)
- Support for multiple comma-separated resource IDs in `tgw routetables --resource-id` flag

### Changed

- Special-case commands migrated to go-output v2 (Special cases phase): vpc enis (single-document flow with rewritten tests), vpc overview (all tables rendered in one pass), demo tables (16 styles with a style-drift guard test); plus verbose-dimension and drawio combine-and-append equivalence tests
- Graph and drawio commands migrated to go-output v2 (Graph and drawio commands phase): appmesh showmesh, iam rolelist/userlist, organizations structure, sso overview (by account and by permission set), tgw routes/overview, vpc peerings — per-format document flavors built additively, drawio combine/append flows ported to v2's CSV read-back, and `tgw routes --list` now supports graph/drawio output instead of erroring
- Canonical table commands migrated to go-output v2 (Canonical table commands phase): appmesh danglingnodes/meshroute, s3 list, sso dangling/list-permission-sets, sso profile generator, cfn resources, tgw dangling, vpc routes — same keys, titles, verbose columns, and sort order, with errors now returned instead of exiting
- `names` and `organizations names` no longer use go-output: direct JSON write to file or stdout with identical semantics, dead S3 output path removed, errors returned instead of exiting (go-output-v2 migration, task 22)
- `clean` target now also removes coverage artifacts
- Formatting fix in `helpers/organizations_test.go`
- Enhanced Transit Gateway route table visualization to use actual resource types from AWS API
- Updated DrawIO output to use raw resource IDs for proper connection matching in Transit Gateway diagrams
- Updated AWS SDK dependencies to latest versions (v1.39.2)

### Fixed

- Profile generator now reads from `--output-file` for conflict detection, template validation, and profile generation instead of always reading the default AWS config file (T-538)
- Role discovery account alias lookup now uses SSO-provided account names instead of IAM ListAccountAliases, which incorrectly returned the template profile's alias for all accounts (T-481)
- ENI cache pointer reuse in `batchFetchVPCEndpoints` and `batchFetchNATGateways` — use index-based iteration to store pointers to slice elements instead of loop variables (T-456)
- Transit Gateway route processing now skips blackhole routes without attachments
- Fixed duplicate destinations in Transit Gateway route tables output
- Fixed Transit Gateway and route table name resolution to use ID as fallback when Name tag is missing

## [1.2.0] - 2025-01-16

### Added

#### SSO Profile Generator
- New `sso profile-generator` command for generating AWS CLI profiles from IAM Identity Center roles
- Automatic discovery and generation of profiles for all assumable roles across accounts
- Conflict detection and resolution with multiple strategies (replace, skip, prompt)

#### VPC IP Finder
- New `vpc ip-finder` command for locating IP addresses across AWS infrastructure
- Comprehensive search across EC2 instances, VPC endpoints, NAT gateways, and load balancers
- Support for both primary and secondary IP addresses

#### VPC Overview
- New `vpc overview` command providing comprehensive VPC resource utilization analysis
- Detailed subnet IP address allocation and usage tracking
- Route table analysis to distinguish public vs private subnets

### Changed
- Updated to latest Go version and modernized all dependencies
- Applied Go modernization patterns throughout the codebase
- Updated GitHub Actions workflows to use latest versions
- Migrated to golangci-lint v2 for improved linting
- Enhanced AWS SDK v2 compatibility and error handling
- Improved code quality with consistent use of constants and switch statements

### Fixed
- Fixed failing unit tests in IAM and EC2 helpers packages
- Fixed AWS SDK v2 boolean pointer handling in S3 operations
- Resolved all critical linting issues (106 → 0 issues)

### Infrastructure
- Added comprehensive unit test coverage for config and helpers packages
- Enhanced development tooling with improved Makefile targets
- Added CLAUDE.md for development guidance

## [1.1.0] - Previous Release