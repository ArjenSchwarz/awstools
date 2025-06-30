# Changelog

All notable changes to this project will be documented in this file.

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
