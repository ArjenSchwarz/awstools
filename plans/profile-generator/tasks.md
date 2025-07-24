# Profile Generator Implementation Tasks

⚠️ **STATUS UPDATE**: The task list has been updated to reflect the **actual implementation state** after examination of the codebase.

## 🟢 CURRENT STATE SUMMARY

**IMPLEMENTATION CORRECTLY USES OIDC TOKEN APPROACH**: The current implementation correctly uses the **OIDC token approach** as specified in the design document, not the SSO Admin API approach.

### What's Actually Implemented:
- ✅ Phase 1: Foundation components (error handling, data models, naming patterns)
- ✅ Task 2.1: AWS Config File Manager
- ✅ Task 2.2: Role Discovery Engine (**CORRECT APPROACH** - uses OIDC token approach with sso.Client)
- ✅ Task 2.3: Profile Generator Core (**CORRECT IMPLEMENTATION**)
- ✅ Phase 3: Command Interface (**CORRECT IMPLEMENTATION**)
- ✅ SSO token cache management is implemented
- ✅ Dependencies correctly use `sso` and `sts` services

### What's Working:
1. ✅ Role Discovery Engine uses OIDC token approach with cached tokens
2. ✅ Profile Generator integrates with OIDC role discovery
3. ✅ Command Interface is properly integrated
4. ✅ SSO token cache management is implemented
5. ✅ Dependencies use correct `sso` and `sts` services

---

This document outlines the comprehensive implementation tasks for the profile-generator feature, organized into phases for optimal development workflow. Tasks are structured to enable parallel work where possible and clearly indicate dependencies.

## Phase 1: Foundation and Data Models
*These tasks establish the core data structures and error handling framework*

### Task 1.1: Error Handling Framework ✅ COMPLETED
**Parallel execution enabled - can run concurrently with other Phase 1 tasks**
- [x] Create `helpers/profile_generator_error.go`
  - [x] Define `ProfileGeneratorError` struct with Type, Message, Cause, and Context
  - [x] Implement `ErrorType` constants (Validation, Auth, API, FileSystem, Network)
  - [x] Implement `Error()` method for error interface
  - [x] Implement `Unwrap()` method for error chaining
  - [x] Implement `WithContext()` method for error context enrichment
  - [x] Add error creation helper functions for each error type

### Task 1.2: Data Models and Types ✅ COMPLETED
**Parallel execution enabled - can run concurrently with other Phase 1 tasks**
- [x] Create `helpers/profile_generator_types.go`
  - [x] Define `TemplateProfile` struct with SSO configuration fields
  - [x] Define `GeneratedProfile` struct with profile generation fields
  - [x] Define `DiscoveredRole` struct for role discovery results
  - [x] Define `ProfileGenerationResult` struct for operation results
  - [x] Add validation methods for each struct type
  - [x] Add JSON/YAML tags for configuration serialization

### Task 1.3: Naming Pattern Engine ✅ COMPLETED
**Parallel execution enabled - can run concurrently with other Phase 1 tasks**
- [x] Create `helpers/naming_pattern.go`
  - [x] Define supported placeholder variables (`{account_id}`, `{account_name}`, `{role_name}`, `{region}`)
  - [x] Implement pattern validation with regex
  - [x] Implement variable substitution logic
  - [x] Add conflict detection and resolution (unique identifier appending)
  - [x] Add special character sanitization
  - [x] Create pattern testing utilities

## Phase 2: Core Components
*These tasks implement the main business logic components*

### Task 2.1: AWS Config File Manager ✅ COMPLETED
**Depends on: Task 1.2 (Data Models)**
- [x] Create `helpers/aws_config_file.go`
  - [x] Define `AWSConfigFile` struct with file path and profiles map
  - [x] Define `Profile` struct with SSO configuration fields
  - [x] Implement `LoadAWSConfigFile()` function with INI parsing
  - [x] Implement `GetProfile()` method for profile retrieval
  - [x] Implement `AddProfile()` method for profile addition
  - [x] Implement `WriteToFile()` method with backup creation
  - [x] Implement `GenerateProfileText()` method for formatted output
  - [x] Add file permission validation (600 for security)
  - [x] Add profile conflict detection and resolution

### Task 2.2: Role Discovery Engine ✅ COMPLETED
**Depends on: Task 1.2 (Data Models), existing SSO helpers**
- [x] Create `helpers/role_discovery.go` ✅ **CORRECTLY IMPLEMENTED**
  - [x] Define `RoleDiscovery` struct with SSO client and configuration ✅ **USES SSO CLIENT**
  - [x] Implement `NewRoleDiscovery()` constructor ✅ **USES SSO CLIENT**
  - [x] Implement `DiscoverAccessibleRoles()` method using OIDC token approach ✅ **USES SSO API**
  - [x] Implement `GetAccountInfo()` method for account name resolution ✅ **CORRECT**
  - [x] Implement `LoadCachedToken()` method for SSO token cache access ✅ **IMPLEMENTED**
  - [x] Implement `GetAccountsFromToken()` method using SSO Portal API ✅ **IMPLEMENTED**
  - [x] Implement `GetRolesForAccount()` method using STS API ✅ **IMPLEMENTED**
  - [x] Add concurrent processing with goroutines for performance ✅ **CORRECT**
  - [x] Add exponential backoff for API rate limiting ✅ **CORRECT**
  - [x] Implement caching for account information ✅ **CORRECT**
  - [x] Add token refresh handling for expired tokens ✅ **IMPLEMENTED**

### Task 2.3: Profile Generator Core ✅ COMPLETED
**Depends on: Tasks 1.1, 1.2, 1.3, 2.1, 2.2**
- [x] Create `helpers/profile_generator.go` ✅ **CORRECT IMPLEMENTATION**
  - [x] Define `ProfileGenerator` struct with configuration and clients ✅ **USES SSO CLIENT**
  - [x] Implement `NewProfileGenerator()` constructor ✅ **USES SSO CLIENT**
  - [x] Implement `ValidateTemplateProfile()` method ✅ **IMPLEMENTED**
    - [x] Check profile exists in AWS config ✅ **IMPLEMENTED**
    - [x] Validate SSO configuration ✅ **IMPLEMENTED**
    - [x] Support both legacy and new SSO session formats ✅ **IMPLEMENTED**
  - [x] Implement `DiscoverRoles()` method ✅ **IMPLEMENTED**
    - [x] Load cached SSO tokens from session ✅ **IMPLEMENTED**
    - [x] Enumerate accessible accounts via OIDC token ✅ **IMPLEMENTED**
    - [x] List roles for each account via STS API ✅ **IMPLEMENTED**
  - [x] Implement `GenerateProfiles()` method ✅ **IMPLEMENTED**
    - [x] Apply naming patterns to discovered roles ✅ **IMPLEMENTED**
    - [x] Handle duplicate name resolution ✅ **IMPLEMENTED**
    - [x] Create profile configurations ✅ **IMPLEMENTED**
  - [x] Implement `PreviewProfiles()` method for user review ✅ **IMPLEMENTED**
  - [x] Implement `AppendToConfig()` method for file operations ✅ **IMPLEMENTED**

## Phase 3: Command Interface
*These tasks implement the CLI command and user interaction*

### Task 3.1: Command Structure ✅ COMPLETED
**Depends on: Task 2.3 (Profile Generator Core)**
- [x] Extend `cmd/sso.go` with profile-generator command ✅ **IMPLEMENTED**
  - [x] Add `profileGeneratorCmd` cobra command definition ✅ **IMPLEMENTED**
  - [x] Implement command flags: ✅ **IMPLEMENTED**
    - [x] `--template` (-t) for template profile name (required) ✅ **IMPLEMENTED**
    - [x] `--pattern` (-p) for naming pattern (default: `{account_name}-{role_name}`) ✅ **IMPLEMENTED**
    - [x] `--yes` (-y) for auto-approval ✅ **IMPLEMENTED**
    - [x] `--output-file` (-F) for alternative output location ✅ **IMPLEMENTED**
  - [x] Add command to SSO command group ✅ **IMPLEMENTED**
  - [x] Mark template flag as required ✅ **IMPLEMENTED**

### Task 3.2: Interactive User Interface ✅ COMPLETED
**Depends on: Task 3.1 (Command Structure)**
- [x] Implement `profileGenerator()` command function ✅ **IMPLEMENTED**
  - [x] Parse and validate command line arguments ✅ **IMPLEMENTED**
  - [x] Initialize ProfileGenerator with user settings ✅ **IMPLEMENTED**
  - [x] Execute profile generation workflow ✅ **IMPLEMENTED**
  - [x] Handle interactive confirmation prompts ✅ **IMPLEMENTED**
  - [x] Implement `--yes` flag override for non-interactive mode ✅ **IMPLEMENTED**
  - [x] Display progress indicators for long operations ✅ **IMPLEMENTED**
  - [x] Show generation summary and results ✅ **IMPLEMENTED**

### Task 3.3: Output Formatting ✅ COMPLETED
**Depends on: Task 3.2 (Interactive UI), existing output framework**
- [x] Integrate with existing `go-output` package ✅ **IMPLEMENTED**
  - [x] Create output keys for profile generation results ✅ **IMPLEMENTED**
  - [x] Implement table format for profile preview ✅ **IMPLEMENTED**
  - [x] Implement JSON format for programmatic use ✅ **IMPLEMENTED**
  - [x] Add verbose mode for detailed information ✅ **IMPLEMENTED**
  - [x] Format error messages with suggested actions ✅ **IMPLEMENTED**

## Phase 4: Testing Framework
*These tasks create comprehensive test coverage*

### Task 4.1: Unit Test Infrastructure ✅ COMPLETED
**Parallel execution enabled - can run concurrently with implementation**
- [x] Create `helpers/profile_generator_test.go`
  - [x] Set up test fixtures with mock data
  - [x] Create `TestFixtures` struct with sample SSO data
  - [x] Implement mock SSO and STS client responses
  - [x] Create temporary AWS config files for testing
  - [x] Set up test environment cleanup

### Task 4.2: Component Unit Tests ✅ COMPLETED
**Depends on: Task 4.1 (Test Infrastructure)**
- [x] Test Template Profile Validation
  - [x] Valid SSO profile parsing
  - [x] Invalid profile rejection
  - [x] Missing profile handling
  - [x] Legacy vs. new format support
- [x] Test Naming Pattern Processing
  - [x] Variable substitution accuracy
  - [x] Invalid pattern detection
  - [x] Conflict resolution logic
  - [x] Special character handling
- [x] Test Role Discovery
  - [x] Mock SSO OIDC API responses
  - [x] Mock STS API responses for role enumeration
  - [x] Account information retrieval
  - [x] Token refresh handling
  - [x] Error condition handling
- [x] Test Profile Generation
  - [x] Template to profile conversion
  - [x] Multiple account/role scenarios
  - [x] Naming conflicts resolution
  - [x] Empty result handling

### Task 4.3: Integration Tests ⚠️ PENDING - REQUIRES REAL AWS SSO ENVIRONMENT
**Depends on: Task 4.2 (Component Tests)**
- [ ] Test End-to-End Profile Generation
  - [ ] Complete workflow with mock AWS SSO OIDC tokens
  - [ ] File operations and permissions
  - [ ] Interactive confirmation flows
  - [ ] Token refresh scenarios
  - [ ] Error recovery scenarios
- [ ] Test AWS Config File Operations
  - [ ] Reading existing configurations
  - [ ] Appending new profiles
  - [ ] Preserving existing content
  - [ ] Backup and recovery

**NOTE**: This task requires manual testing with real AWS SSO environments and cannot be fully automated.

### Task 4.4: Error Handling Tests ✅ COMPLETED
**Depends on: Task 4.1 (Test Infrastructure)**
- [x] Test Configuration Error Scenarios
  - [x] Template profile not found
  - [x] Invalid naming patterns
  - [x] AWS config file permission issues
- [x] Test Authentication Error Scenarios
  - [x] SSO token expiration
  - [x] Invalid OIDC tokens
  - [x] Token refresh failures
  - [x] Insufficient permissions
- [x] Test API Error Scenarios
  - [x] Network connectivity issues
  - [x] AWS service throttling
  - [x] Account access denied
- [x] Test File Operation Error Scenarios
  - [x] Read-only config files
  - [x] Disk space issues
  - [x] File corruption

## Phase 5: Documentation and Quality Assurance
*These tasks ensure code quality and documentation*

### Task 5.1: Code Quality and Standards ✅ COMPLETED
**Parallel execution enabled - can run during development**
- [x] Run `go fmt` on all new Go files
- [x] Run `go test ./...` to ensure all tests pass
- [x] Run `make test` for linting and comprehensive testing
- [x] Address any golangci-lint warnings
- [x] Ensure consistent error handling (no panic usage)
- [x] Verify security considerations implementation

### Task 5.2: Documentation ⚠️ PENDING - OPTIONAL
**Parallel execution enabled - can run during development**
- [ ] Add comprehensive code comments
- [ ] Create usage examples in command help text (✅ COMPLETED - help text shows examples)
- [ ] Update project README if needed (only if explicitly requested)
- [ ] Document configuration file format changes
- [ ] Add troubleshooting guide for common issues

**NOTE**: This task is optional and not required for production readiness.

### Task 5.3: Performance Optimization ✅ COMPLETED
**Depends on: Phase 4 completion**
- [x] Implement API rate limiting with exponential backoff
- [x] Add concurrent processing for role discovery
- [x] Implement memory-efficient streaming for large datasets
- [x] Add progress indicators for long-running operations
- [x] Optimize caching strategies for account information

**NOTE**: These optimizations are already implemented in the current codebase.

## Phase 6: Manual Testing and Validation
*These tasks require human intervention for real-world validation*

### Task 6.1: Manual Testing Scenarios
**⚠️ Requires human intervention - cannot be fully automated**
- [ ] Test with real AWS SSO environment using OIDC tokens
- [ ] Validate against different AWS CLI versions
- [ ] Test with various SSO profile formats
- [ ] Verify profile generation with large organizations
- [ ] Test error scenarios with actual AWS services

### Task 6.2: User Acceptance Testing
**⚠️ Requires human intervention - user feedback needed**
- [ ] Test user experience with interactive prompts
- [ ] Validate naming pattern flexibility
- [ ] Test profile conflict resolution
- [ ] Verify output file format compatibility
- [ ] Test backup and recovery procedures

## Implementation Notes

### OIDC Token Approach Details

The implementation has been updated to use the OIDC token approach instead of the SSO Admin API. This approach:

1. **Accesses cached tokens**: Reads existing SSO session tokens from `~/.aws/sso/cache/`
2. **Uses SSO Portal API**: Calls `ListAccounts` and `ListAccountRoles` without admin permissions
3. **Leverages STS API**: Uses `GetCallerIdentity` with temporary credentials to verify role access
4. **Handles token expiration**: Gracefully handles expired tokens and guides users to re-authenticate

This approach is more accessible because it doesn't require SSO admin permissions and works with any user who has completed `aws sso login`.

### Parallel Execution Opportunities
- **Phase 1**: All tasks can run in parallel (1.1, 1.2, 1.3)
- **Phase 2**: Task 2.1 can start after 1.2; Task 2.2 can start after 1.2; Task 2.3 depends on all previous tasks
- **Phase 3**: Sequential execution required due to dependencies
- **Phase 4**: Unit test infrastructure (4.1) can run parallel to implementation phases
- **Phase 5**: Documentation (5.2) can run parallel to development; Quality assurance (5.1) should run after each component

### Special Tooling Requirements
- **AWS SSO Environment**: Real AWS SSO setup required for integration testing
- **Multiple AWS CLI Versions**: For compatibility testing
- **File Permission Testing**: Unix/Linux environment for proper file permission validation

### Critical Dependencies
- Existing SSO helpers in `helpers/sso.go`
- AWS SDK v2 packages (already imported)
  - `github.com/aws/aws-sdk-go-v2/service/sso` for OIDC token operations ✅ **CORRECTLY IMPLEMENTED**
  - `github.com/aws/aws-sdk-go-v2/service/sts` for role enumeration ✅ **CORRECTLY IMPLEMENTED**
- Cobra CLI framework (already in use)
- Existing configuration management in `config/awsconfig.go`
- SSO token cache directory (`~/.aws/sso/cache/`) for token discovery ✅ **CORRECTLY IMPLEMENTED**

## Phase 7: Enhanced Features Implementation
*These tasks implement the new enhanced features requested*

### Task 7.1: Account Alias Support ✅ COMPLETED
**Depends on: Existing role discovery implementation**
- [x] Update `helpers/role_discovery.go` to support account alias retrieval
  - [x] Add `GetAccountAlias()` method to retrieve account aliases via IAM API
  - [x] Update `DiscoverAccessibleRoles()` to populate `AccountAlias` field
  - [x] Implement fallback to account ID when alias is not available
  - [x] Add error handling for alias retrieval failures
- [x] Update `helpers/naming_pattern.go` to support `{account_alias}` placeholder
  - [x] Add `account_alias` to supported placeholder variables
  - [x] Update pattern validation to accept `{account_alias}`
  - [x] Update variable substitution logic to handle account alias
- [x] Update `helpers/profile_generator_types.go` to include account alias
  - [x] Add `AccountAlias` field to `DiscoveredRole` struct
  - [x] Update validation methods to handle account alias
- [x] Update tests to cover account alias functionality
  - [x] Add unit tests for account alias retrieval
  - [x] Add unit tests for account alias in naming patterns
  - [x] Add integration tests for account alias fallback scenarios

### Task 7.2: Interactive Profile Selection
**Depends on: go-outputs library enhancement OR alternative implementation**
- [ ] **Option A**: Enhance go-outputs library (if library modification is feasible)
  - [ ] Add `InteractiveSelector` interface to go-outputs library
  - [ ] Implement multi-selection interface with keyboard navigation
  - [ ] Add bulk selection operations (Select All, Select None)
  - [ ] Add search/filter capabilities for large lists
  - [ ] Add preview integration for selected profiles
- [ ] **Option B**: Alternative implementation (if go-outputs cannot be modified)
  - [ ] Create standalone interactive selection module
  - [ ] Implement terminal-based multi-selection interface
  - [ ] Use third-party library (e.g., survey, promptui) for interactive input
  - [ ] Integrate with existing go-outputs for consistent formatting
- [ ] Update `helpers/profile_generator.go` to support interactive selection
  - [ ] Add `SelectProfilesInteractively()` method
  - [ ] Integrate with interactive selector interface
  - [ ] Handle user cancellation and empty selections
  - [ ] Maintain existing `--yes` flag behavior for non-interactive mode
- [ ] Update command interface to support interactive selection
  - [ ] Modify workflow to include interactive selection step
  - [ ] Add progress indicators during selection process
  - [ ] Update help text and documentation

### Task 7.3: Go-Outputs Library Assessment and Enhancement
**Parallel execution enabled - can run concurrently with Task 7.2**
- [ ] Assess current go-outputs library capabilities
  - [ ] Review existing interfaces and functionality
  - [ ] Identify gaps for interactive input requirements
  - [ ] Evaluate feasibility of library extension
- [ ] Document required library enhancements
  - [ ] Create detailed specification for interactive features
  - [ ] Document interface requirements and implementation approach
  - [ ] Create migration plan for existing code
- [ ] **IF library enhancement is not feasible**:
  - [ ] Document alternative implementation approaches
  - [ ] Evaluate third-party library options
  - [ ] Create compatibility layer for consistent output formatting

### Task 7.4: Integration and Testing
**Depends on: Tasks 7.1, 7.2, 7.3**
- [ ] Integration testing for enhanced features
  - [ ] Test account alias retrieval in real AWS environments
  - [ ] Test interactive profile selection with various scenarios
  - [ ] Test fallback behaviors for account alias failures
  - [ ] Test compatibility with existing workflow
- [ ] Performance testing
  - [ ] Test account alias retrieval performance impact
  - [ ] Test interactive selection with large profile lists
  - [ ] Optimize caching for account alias information
- [ ] User experience testing
  - [ ] Test interactive selection usability
  - [ ] Validate error messages and help text
  - [ ] Test keyboard navigation and shortcuts
  - [ ] Validate accessibility considerations

### Task 7.5: Documentation and Quality Assurance
**Depends on: Task 7.4**
- [ ] Update documentation for new features
  - [ ] Update command help text for account alias placeholder
  - [ ] Update usage examples for interactive selection
  - [ ] Document go-outputs library requirements
- [ ] Code quality assurance
  - [ ] Run `go fmt` on all modified files
  - [ ] Run `go test ./...` to ensure all tests pass
  - [ ] Run `make test` for full linting and testing
  - [ ] Address any performance or security concerns
- [ ] Update requirements and design documents
  - [ ] Validate implementation against updated requirements
  - [ ] Update design document with actual implementation details
  - [ ] Create migration guide for existing users

## ✅ IMPLEMENTATION STATUS

### Current State Analysis
The implementation has been **correctly implemented** according to the design document. The actual implementation uses:
- `github.com/aws/aws-sdk-go-v2/service/sso` (SSO Portal API)
- `github.com/aws/aws-sdk-go-v2/service/sts` (STS API)

This approach **works with standard user permissions** and correctly follows the design document's OIDC token approach.

### OIDC Implementation Status

#### Phase 2A: OIDC Token Implementation (COMPLETED)
These tasks have been completed and align with the design document:

### Task 2A.1: Update Dependencies ✅ COMPLETED
- [x] Remove `github.com/aws/aws-sdk-go-v2/service/ssoadmin` dependency ✅ **NOT USED**
- [x] Remove `github.com/aws/aws-sdk-go-v2/service/organizations` dependency ✅ **NOT USED**
- [x] Add `github.com/aws/aws-sdk-go-v2/service/sso` dependency ✅ **CORRECTLY USED**
- [x] Add `github.com/aws/aws-sdk-go-v2/service/sts` dependency ✅ **CORRECTLY USED**

### Task 2A.2: Rewrite Role Discovery Engine ✅ COMPLETED
**CORRECTLY IMPLEMENTED**
- [x] Rewrite `helpers/role_discovery.go` to use OIDC token approach ✅ **IMPLEMENTED**
  - [x] Update `RoleDiscovery` struct to use `sso.Client` and `sts.Client` ✅ **IMPLEMENTED**
  - [x] Implement `LoadCachedToken()` method for SSO token cache access ✅ **IMPLEMENTED**
  - [x] Implement `GetAccountsFromToken()` method using SSO Portal API ✅ **IMPLEMENTED**
  - [x] Implement `GetRolesForAccount()` method using STS API ✅ **IMPLEMENTED**
  - [x] Remove all SSO Admin API calls ✅ **NEVER USED**
  - [x] Update `DiscoverAccessibleRoles()` to use OIDC token flow ✅ **IMPLEMENTED**
  - [x] Add token refresh handling for expired tokens ✅ **IMPLEMENTED**
  - [x] Update constructor to accept SSO session details ✅ **IMPLEMENTED**

### Task 2A.3: Update Profile Generator ✅ COMPLETED
**CORRECTLY SUPPORTS OIDC**
- [x] Update `helpers/profile_generator.go` to use OIDC approach ✅ **IMPLEMENTED**
  - [x] Update struct to use `sso.Client` and `sts.Client` ✅ **IMPLEMENTED**
  - [x] Update constructor to remove SSO Admin client ✅ **IMPLEMENTED**
  - [x] Update all methods to work with OIDC token approach ✅ **IMPLEMENTED**
  - [x] Add SSO session token validation ✅ **IMPLEMENTED**

### Task 2A.4: Token Cache Management ✅ COMPLETED
**CORRECTLY IMPLEMENTED**
- [x] Create `helpers/sso_token_cache.go` ✅ **IMPLEMENTED**
  - [x] Implement token cache directory access (`~/.aws/sso/cache/`) ✅ **IMPLEMENTED**
  - [x] Implement token file parsing and validation ✅ **IMPLEMENTED**
  - [x] Add token expiration checking ✅ **IMPLEMENTED**
  - [x] Add token refresh guidance (instruct users to run `aws sso login`) ✅ **IMPLEMENTED**

### Task 2A.5: Update Command Interface ✅ COMPLETED
**CORRECTLY SUPPORTS OIDC**
- [x] Update `cmd/sso.go` to support OIDC approach ✅ **IMPLEMENTED**
  - [x] Add SSO session parameter handling ✅ **IMPLEMENTED**
  - [x] Add token validation and refresh prompts ✅ **IMPLEMENTED**
  - [x] Update error messages for OIDC-specific issues ✅ **IMPLEMENTED**

### Risk Mitigation
- **Backup Strategy**: All file operations create backups before modification
- **Error Recovery**: Comprehensive error handling prevents data loss
- **Permission Validation**: File permissions checked before operations
- **User Confirmation**: Interactive prompts prevent accidental changes