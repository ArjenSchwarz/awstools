# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Regression tests for IAM access-key API failures (T-1549)
- Makefile targets for code quality: `fmt`, `vet`, `modernize`, `check`, `security-scan`
- Makefile targets for testing: `test-verbose`, `test-coverage`
- Makefile targets for dependency management: `deps-tidy`, `deps-update`
- Makefile target `install` for installing the application
- Organized help output with categorized sections
- Direct Connect Gateway support for name resolution in EC2 resources
- Support for additional Transit Gateway attachment types in DrawIO output (peering, direct-connect-gateway, connect)
- Support for multiple comma-separated resource IDs in `tgw routetables --resource-id` flag

### Changed

- `clean` target now also removes coverage artifacts
- Formatting fix in `helpers/organizations_test.go`
- Enhanced Transit Gateway route table visualization to use actual resource types from AWS API
- Updated DrawIO output to use raw resource IDs for proper connection matching in Transit Gateway diagrams
- Updated AWS SDK dependencies to latest versions (v1.39.2)

### Fixed

- `tgw routetables --resource-id vpc-...` now keeps the destination resources of route tables the VPC matches only as a source attachment, so filtered graph and text output no longer reference missing nodes; Draw.io output also omits inbound edges to route tables removed by the filter (T-1717)
- SSO `by-account` and `by-permission-set` graph outputs (drawio, dot, mermaid) filtered with `--resource-id` no longer draw root edges to accounts or permission sets that were filtered out, and permission-set root edges use name-file names to match their nodes (T-1714)
- `vpc ip-finder` now finds IPv6 addresses assigned to ENIs instead of reporting them as not found; IPv6 results are marked secondary unless AWS flags the address as the ENI's primary IPv6 address (T-1706)
- `vpc enis` now reports the Transit Gateway attachment that owns each TGW ENI when a VPC has several TGW attachments, instead of reusing the last attachment found for the VPC; ENIs whose attachment can't be determined show as unknown rather than a guessed attachment (T-1571)
- `vpc enis --split` with JSON (or YAML) output now writes one valid document of all ENI rows, including their VPC and Subnet, instead of one array per subnet back to back; table-style formats keep their per-subnet tables (T-1591)
- IAM `rolelist` no longer merges same-named attached policies (e.g. AWS-managed and customer-managed `ReadOnlyAccess`): attached policies are keyed by ARN, policy rows gain an `ARN` column and list only their own roles, and drawio/dot/mermaid link roles to attached policies by ARN. `rolelist` mermaid output (stdout or file, alongside any other file or stdout format) no longer panics on edges to missing nodes or merges roles that have several policies (T-1656)
- Profile generator now applies conflict detection (skip/replace/prompt) to naming patterns containing `{region}`, using the template profile's region, and stops with an error instead of treating roles it cannot analyse as conflict-free (T-1428)
- Profile generator no longer gives two generated profiles the same name (two replaced conflicts, or a replaced conflict and a non-conflicted role), which made one role overwrite the other in the AWS config; the later role now gets a numeric suffix that avoids existing profiles (T-1646)
- `vpc overview --file` now saves every table (subnet overviews, IP details and summary) instead of only the summary. JSON and YAML files are one document listing each table's title and rows, HTML files are one page, and `--append` adds the tables to an existing HTML report (T-1536)
- App Mesh route and provider helpers now tolerate missing nested fields without panicking (T-1634)
- S3 bucket listing now reports enumeration failures once without panicking or printing usage (T-1643)
- IAM user listing now returns contextual command errors when access-key API calls fail instead of panicking (T-1549)
- `sso profile-generator` now discovers roles through the SSO portal in the template profile's `sso_region` instead of the ambient AWS region (T-1468)
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
