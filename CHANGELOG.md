# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Makefile targets for code quality: `fmt`, `vet`, `modernize`, `check`, `security-scan`
- Makefile targets for testing: `test-verbose`, `test-coverage`
- Makefile targets for dependency management: `deps-tidy`, `deps-update`
- Makefile target `install` for installing the application
- Organized help output with categorized sections

### Fixed

- ENI cache pointer reuse in `batchFetchVPCEndpoints` and `batchFetchNATGateways` — use index-based iteration to store pointers to slice elements instead of loop variables (T-456)

### Changed

- `clean` target now also removes coverage artifacts
- Formatting fix in `helpers/organizations_test.go`

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