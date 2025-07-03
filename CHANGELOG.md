# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [2025-07-03] - Added VPC overview command and IP usage analysis

### Added
- New `vpc overview` command providing comprehensive VPC resource utilization analysis
- Detailed subnet IP address allocation and usage tracking
- VPC usage summary statistics with filtering capabilities
- Enhanced Claude Code configuration with design and task generation commands

### Technical Details
- VPC overview command supports filtering by specific VPC ID using `--vpc` flag
- IP address analysis includes AWS reserved IPs, service IPs, and availability tracking
- Tiered resource naming using both global naming and Name tags
- Route table analysis to distinguish public vs private subnets
- Support for multiple output formats (JSON, CSV, table, HTML, etc.)

### Files Added
- `cmd/vpcoverview.go` - VPC overview command implementation (227 lines)
- `plans/vpc-overview/design.md` - Comprehensive design documentation
- `plans/vpc-overview/requirements.md` - Feature requirements specification
- `plans/vpc-overview/tasks.md` - Implementation task breakdown
- `plans/vpc-overview/eni-analyser.js` - ENI analysis utility script
- `.claude/commands/design.md` - Design document generation command
- `.claude/commands/tasks.md` - Task list generation command

### Files Modified
- `helpers/ec2.go` - Added VPC usage analysis functions (618 lines of additions)
- `.claude/settings.local.json` - Enhanced tool permissions for development
- `CHANGELOG.md` - Updated format to follow Keep a Changelog standard

## [2025-07-02 14:42:05] - Added comprehensive unit tests for config package

### Added
- Created comprehensive unit test suite covering all functions in the config package
- Tests for Config struct methods (GetLCString, GetOutputFormat, GetString, GetBool, GetInt, etc.)
- Tests for output formatting and separator logic
- Tests for AWS client creation methods in AWSConfig struct
- Tests for AWS configuration handling with profile and region support

### Technical Details
- 397 lines of comprehensive test coverage across both config.go and awsconfig.go
- Table-driven tests with subtests for different scenarios
- Mock-friendly tests that handle AWS credential requirements gracefully
- All tests pass with proper Go formatting standards
- Added testify/assert dependency for improved test assertions

### Files Added
- `config/config_test.go` - Unit tests for Config struct methods and output settings
- `config/awsconfig_test.go` - Unit tests for AWSConfig struct and AWS client creation

### Dependencies Updated
- `go.mod` - Added github.com/stretchr/testify v1.10.0 for enhanced testing capabilities
- `go.sum` - Updated dependency checksums and removed unused dependencies

## [2025-01-01 00:00:00] - Added comprehensive unit tests for helpers package

### Added
- Created unit test suite covering all 11 helper modules in the helpers package
- Tests for AWS service integrations (EC2, S3, IAM, RDS, SSO, Organizations, CloudFormation, App Mesh)
- Tests for utility functions and data structure validation
- Edge case handling and error condition testing

### Technical Details
- 2,266 lines of comprehensive test coverage across all helper functions
- Table-driven tests for complex logic validation
- Mock structures prepared for future integration testing
- All tests pass with proper Go formatting standards

### Files Added
- `helpers/appmesh_test.go` - App Mesh structure and route testing
- `helpers/cfn_test.go` - CloudFormation stack resource testing
- `helpers/ec2_test.go` - EC2, VPC, Transit Gateway testing
- `helpers/iam_test.go` - IAM user, group, and policy testing
- `helpers/iamroles_test.go` - IAM role and policy document testing
- `helpers/organizations_test.go` - AWS Organizations structure testing
- `helpers/rds_test.go` - RDS instance and tag processing testing
- `helpers/s3_test.go` - S3 bucket configuration and replication testing
- `helpers/sso_test.go` - SSO instance and permission set testing
- `helpers/sts_test.go` - STS account identity testing
- `helpers/utils_test.go` - Utility function testing

## [2025-06-30] - Fixed linting issues and updated build infrastructure

### Added
- Created `.golangci.yml` configuration file for golangci-lint v2.2.1
- Added `cmd/constants.go` with shared constants for command column names and resource types
- Added `CLAUDE.md` with development guidance for Claude Code
- Added `cmd/vpcenis.go` command file
- Created `.claude/` directory for Claude Code configuration

### Fixed
- **Critical linting issues resolved (106 → 63 issues)**:
  - Fixed all 8 errcheck issues by adding proper error handling for `viper.BindPFlag` calls
  - Fixed all 9 staticcheck issues including deprecated `io/ioutil` imports, inefficient slice operations, and AWS SDK v2 compatibility
  - Fixed all 4 goconst issues by creating constants for repeated strings (`nameColumn`, `childrenColumn`, `permissionSetColumn`, `vpcResourceType`)
  - Reduced gocritic issues from 30 to 9 by fixing `strings.Replace` → `strings.ReplaceAll`, optimizing regex compilation
  - Fixed 1 unused variable issue
- **AWS SDK v2 compatibility**:
  - Updated S3 helper functions to handle new pointer types (`*bool` → `bool` conversions)
  - Fixed `ReplicationRuleFilter` handling for new struct-based approach
  - Replaced deprecated `rule.Prefix` usage
- **Build system improvements**:
  - Upgraded golangci-lint from v1.54.2 to v2.2.1 for Go 1.24.1 compatibility
  - Fixed import statements to use `os.ReadFile` instead of deprecated `io/ioutil.ReadFile`

### Technical Details
- Centralized repeated string constants in `cmd/constants.go` to reduce code duplication
- Implemented proper error handling for configuration binding operations
- Optimized regular expression compilation using `regexp.MustCompile` for constant patterns
- Updated AWS SDK v2 usage patterns for better type safety

### Files Modified
- `Makefile` - Reorganized structure with all .PHONY declarations at top, added help target
- `cmd/appmeshshowmesh.go` - Optimized slice append operations
- `cmd/demosettings.go` - Fixed YAML import alias
- `cmd/docsghpages.go` - Updated to use `strings.ReplaceAll`
- `cmd/iamrolelist.go` - Added constants for repeated strings
- `cmd/iamuserlist.go` - Fixed regex compilation and constants usage
- `cmd/names.go` - Removed unused variable
- `cmd/organizationsstructure.go` - Added constants for repeated strings
- `cmd/root.go` - Added error handling for viper operations, fixed deprecated imports
- `cmd/s3list.go` - Fixed AWS SDK v2 boolean pointer handling
- `cmd/ssolistpermissionsets.go` - Added constants for repeated strings
- `cmd/ssooverviewaccount.go` - Added constants for repeated strings
- `cmd/tgwoverview.go` - Added constants for repeated strings
- `cmd/tgwroutes.go` - Added constants for repeated strings
- `config/awsconfig.go` - Auto-formatted
- `go.mod` - Updated dependencies
- `go.sum` - Updated dependency checksums
- `helpers/ec2.go` - Auto-formatted
- `helpers/rds.go` - Auto-formatted
- `helpers/s3.go` - Fixed AWS SDK v2 compatibility, optimized slice operations
- `helpers/sso.go` - Auto-formatted
- `helpers/utils.go` - Fixed deprecated `io/ioutil` import

### Files Added
- `.golangci.yml` - Golangci-lint v2 configuration with essential linters
- `cmd/constants.go` - Shared constants for command implementations
- `CLAUDE.md` - Development commands and architecture documentation
- `cmd/vpcenis.go` - VPC ENI command implementation
- `.claude/` - Claude Code configuration directory
