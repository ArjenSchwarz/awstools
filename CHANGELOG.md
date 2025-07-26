# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Enhanced
- Profile generator with integrated conflict detection and resolution orchestration
- Conflict resolution workflow with strategy-based handling (replace, skip, prompt)
- Profile generation result reporting with detailed conflict resolution information
- Conflict detector initialization with lazy loading for improved performance
- Profile filtering and generation for non-conflicted roles
- Enhanced conflict reporting with comprehensive action tracking

### Added
- `ConflictResolutionResult` struct for structured conflict resolution outcomes
- `FilterRolesByConflicts()` method for separating conflicted and non-conflicted roles
- `GenerateProfilesForNonConflictedRoles()` method for efficient profile generation
- `initializeConflictDetector()` method with lazy initialization pattern
- Enhanced `ProfileGenerationResult` with conflict resolution fields and detailed reporting
- Comprehensive test coverage for conflict resolution orchestration (650+ test lines)

### Technical Details
- Integrated `ProfileConflictDetector` into `ProfileGenerator` workflow
- Enhanced `ResolveConflicts()` to return structured `ConflictResolutionResult`
- Added conflict action tracking with detailed operation logging
- Improved profile generation efficiency with role filtering optimization
- Extended test suite with conflict detection integration and resolution testing

### Added
- Profile conflict detection engine with comprehensive conflict analysis capabilities
- Enhanced profile generator with conflict resolution strategies (replace, skip, prompt)
- Comprehensive profile conflict detector with role-to-profile matching capabilities
- Enhanced AWS config file operations with profile replacement and backup functionality
- SSO token cache improvements with better error handling and validation
- Optimized naming pattern validation using slices.Contains for better performance
- Optimized profile lookup indices for O(1) conflict detection performance
- Enhanced AWS config file operations with profile search and matching utilities
- Comprehensive test coverage for profile conflict detection with 400+ test lines
- Performance optimizations with pre-allocated slice capacity and cached SSO configurations
- Enhanced profile generator data models with conflict resolution support
- SSO session configuration support in AWS config file handling
- Comprehensive test coverage for profile generator types and AWS config file operations
- Conflict resolution strategy enums and data structures for profile generation
- Profile conflict detection types and action tracking structures
- Kiro steering documentation for development guidelines and project structure

### Enhanced
- AWS config file parser with SSO session support for both legacy and modern formats
- Profile generator types with validation methods and conflict handling capabilities
- Data models for profile replacement tracking and conflict action recording
- Test suite with 963 lines of comprehensive coverage for new functionality

### Technical Details
- Extended `helpers/profile_generator_types.go` with conflict resolution data structures
- Added `ConflictResolutionStrategy`, `ProfileConflict`, `ConflictAction`, and `ProfileReplacement` types
- Enhanced `helpers/aws_config_file.go` with SSO session resolution and profile matching
- Added `SSOSession`, `ResolvedSSOConfig` structs for normalized SSO configuration handling
- Created comprehensive unit tests in `helpers/aws_config_file_test.go` and `helpers/profile_generator_types_test.go`
- Added `.kiro/steering/agents.md` with development guidelines for Go code quality

- Profile generator enhancement specification with conflict detection and resolution capabilities
- Account alias support in profile naming patterns with `{account_alias}` placeholder
- Comprehensive efficiency improvements documentation for AWS API optimization
- Kiro automation hooks for code efficiency checking and manual commit workflows
- Enhanced project steering documentation for technology stack, structure, and product overview

- Profile generator with account alias retrieval and fallback to account ID
- Role discovery with IAM client integration for account alias support
- Naming pattern validation to include account alias placeholder
- Test coverage for account alias functionality and naming patterns

### Enhanced
- New `sso profile-generator` command for generating AWS CLI profiles from IAM Identity Center roles
- Comprehensive profile generation with support for all assumable roles across accounts
- Profile naming patterns with customizable template support
- Token-based authentication with automatic caching and refresh
- Batch profile generation with conflict detection and resolution
- Support for both legacy and modern SSO profile formats
- Extensive test coverage for profile generation functionality
- Command-line interface with preview, approval, and output file options

### Changed
- Updated go.mod dependencies to latest versions including AWS SDK v2 updates
- Enhanced AWS config file handling with support for AWS_CONFIG_FILE environment variable
- Improved error handling with structured error types and context
- Added comprehensive documentation for profile generator feature

### Technical Details
- Added `helpers/profile_generator.go` with full profile generation workflow
- Added `helpers/profile_generator_types.go` with structured data types
- Added `helpers/profile_generator_test.go` with comprehensive test suite (1033 lines)
- Added `helpers/sso_token_cache.go` for SSO token management
- Enhanced `helpers/aws_config_file.go` with profile appending functionality
- Updated `helpers/role_discovery.go` with improved role discovery logic
- Added SSO service integration for role and account discovery

## [Previous Unreleased]

### Fixed
- Fixed failing unit tests in IAM and EC2 helpers packages
- Fixed IAM `CanBeAssumedFrom` method to use alphabetical sorting for consistent output
- Fixed EC2 `getResourceNameAndID` function to properly handle unattached ENIs
- Fixed unused parameter linting issues in ip-finder command and EC2 helper functions

### Changed
- Simplified VPC ip-finder command by removing `--include-secondary` flag (always searches both primary and secondary IPs)
- Updated help text and documentation to clarify that ip-finder searches both IP types by default
- Simplified IAM principal sorting logic to use standard alphabetical ordering
- Added constants for IAM principal types to improve code maintainability

### Removed
- Removed `--include-secondary` flag from vpc ip-finder command
- Removed complex custom sorting logic in IAM CanBeAssumedFrom method

### Added
- New `vpc ip-finder` command for locating IP addresses across AWS infrastructure
- Comprehensive IP address search functionality with support for primary and secondary IPs
- Detailed resource identification including EC2 instances, VPC endpoints, NAT gateways, and load balancers
- Enhanced output showing VPC, subnet, security group, and route table information
- Support for all existing output formats (JSON, CSV, table, HTML)
- Comprehensive error handling with helpful troubleshooting suggestions
- Integration with existing ENI lookup cache for optimal performance

### Technical Details
- Added IP address validation functions supporting both IPv4 and IPv6
- Implemented ENI search filters for efficient AWS API queries
- Created comprehensive data structures for IP finder results
- Added extensive unit tests, integration tests, and performance benchmarks
- Following established awstools patterns for error handling and output formatting
- Leveraging existing ENI analysis and caching mechanisms for performance optimization

## [2025-07-03] - Refactored VPC overview code following PR review feedback

### Changed
- Eliminated code duplication by creating centralized `GetResourceDisplayNameWithGlobalLookup` helper function
- Enhanced documentation for `analyzeSubnetIPUsage` function with comprehensive parameter and return value descriptions
- Centralized resource naming logic to ensure consistent behavior across the codebase

### Fixed
- Removed unused functions `getENIUsageType` and `getENIAttachmentDetails` to reduce code maintenance burden
- Added string constants for repeated ENI service types to fix goconst linting issues
- All linting issues resolved (0 issues remaining)

### Technical Details
- Created reusable helper supporting both global naming lookup and tag-based fallback
- Updated cmd/vpcoverview.go to use centralized helper instead of local duplicate implementation
- Added constants: `vpcEndpointType`, `interfaceType`, `lambdaFunctionType`, `awsServiceType`
- Enhanced function documentation explaining IP usage analysis process and performance optimizations

## [2025-07-03] - Fixed linting issues and updated development guidelines

### Fixed
- Fixed goconst linting issue by using existing nameColumn constant instead of hardcoded "Name" string
- Fixed gocritic linting issue by converting if-else chain to switch statement for better readability
- Reduced linting issues from 2 to 0

### Changed
- Updated CLAUDE.md with additional development guidelines about generated text in commits and pull requests
- Improved code structure in route table target identification logic

### Technical Details
- Converted if-else chain in `FormatRouteTableInfo` function to switch statement for better performance
- Used consistent naming constants across the codebase
- Enhanced development documentation with Claude-specific guidance

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
