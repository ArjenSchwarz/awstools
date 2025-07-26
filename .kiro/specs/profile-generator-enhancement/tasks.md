# Profile Generator Enhancement Implementation Plan

This implementation plan converts the profile generator enhancement design into a series of discrete coding tasks that build incrementally on the existing profile generator implementation. Each task focuses on writing, modifying, or testing specific code components to add conflict detection and resolution capabilities.

- [x] Task 1: Enhance Data Models and Types

    - [x] 1.1 Extend profile generator types for conflict handling
      - Modify `helpers/profile_generator_types.go` to add conflict resolution data structures
      - Add `ConflictResolutionStrategy` enum with Prompt, Replace, Skip options
      - Add `ProfileConflict` struct with discovered role, existing profiles, and conflict type
      - Add `ConflictAction` struct to track resolution actions taken
      - Add `ProfileReplacement` struct for tracking replaced profiles
      - Write unit tests for new data structures and their validation methods
      - _Requirements: 1.1, 2.1, 3.1, 4.1, 5.1_

  - [x] 1.2 Enhance AWS config file data models
    - Modify `helpers/aws_config_file.go` to add SSO session support
    - Add `SSOSession` struct for SSO session configurations
    - Add `ResolvedSSOConfig` struct for normalized SSO configuration matching
    - Extend `Profile` struct with `ResolvedSSOConfig` field for conflict detection
    - Extend `AWSConfigFile` struct with `Sessions` map for SSO session storage
    - Write unit tests for enhanced data models and SSO session resolution
    - _Requirements: 8.1, 8.2, 8.3, 8.4_

- [x] 2: Implement Profile Conflict Detection Engine

  - [x] 2.1 Create profile conflict detector component
    - Create `helpers/profile_conflict_detector.go` with conflict detection logic
    - Implement `ProfileConflictDetector` struct with config file and naming pattern dependencies
    - Implement `DetectConflicts()` method to analyze all discovered roles for conflicts
    - Implement `AnalyzeRole()` method to check individual roles against existing profiles
    - Implement `ClassifyConflict()` method to determine conflict types (same role vs same name)
    - Write unit tests for conflict detection with various profile formats and scenarios
    - _Requirements: 2.1, 2.2, 2.3, 2.4_

  - [x] 2.2 Implement SSO configuration resolution and matching
    - Extend `helpers/aws_config_file.go` with SSO session resolution methods
    - Implement `LoadSSOSessions()` method to parse SSO session configurations from config file
    - Implement `ResolveSSOSession()` method to resolve session references to actual configurations
    - Implement `ResolveProfileSSOConfig()` method to normalize both legacy and session-based SSO formats
    - Implement `MatchesRole()` method to compare profiles against discovered roles using normalized SSO config
    - Write unit tests for SSO resolution with legacy format, session format, and mixed environments
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5_

  - [x] 2.3 Implement profile search and matching utilities
    - Add `FindProfilesForRole()` method to `AWSConfigFile` to find existing profiles for specific roles
    - Implement efficient profile lookup using hash maps for performance
    - Add profile name conflict detection for proposed new profile names
    - Implement duplicate profile detection for same SSO configuration
    - Write unit tests for profile search with various conflict scenarios and edge cases
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5_

- [x] 3: Implement Conflict Resolution Strategies

  - [x] 3.1 Implement profile replacement functionality
    - Add `ReplaceProfile()` method to `AWSConfigFile` for atomic profile replacement
    - Add `RemoveProfile()` method to safely remove existing profiles
    - Implement profile name change handling while preserving custom configuration properties
    - Add validation to ensure replacement operations maintain config file integrity
    - Write unit tests for profile replacement with various profile formats and custom properties
    - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5_

  - [x] 3.2 Implement backup and recovery system
    - Add `CreateBackup()` method to `AWSConfigFile` to create timestamped backups before modifications
    - Add `RestoreFromBackup()` method to recover from failed operations
    - Implement atomic file operations to prevent corruption during concurrent access
    - Add file permission preservation during backup and restore operations
    - Write unit tests for backup creation, restoration, and error recovery scenarios
    - _Requirements: 3.5, 7.4, 7.5_

  - [x] 3.3 Implement interactive conflict resolution
    - Add `PromptForConflictResolution()` method to `ProfileGenerator` for user interaction
    - Implement clear conflict presentation showing existing vs proposed profile names
    - Add user input validation for conflict resolution choices (replace/skip/cancel)
    - Implement cancellation handling that exits without making any changes
    - Write unit tests for interactive prompts using mock input/output streams
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5_

- [x] 4: Enhance Profile Generator Core Logic

  - [x] 4.1 Integrate conflict detection into profile generation workflow
    - Modify `helpers/profile_generator.go` to add conflict resolution strategy parameter
    - Add `conflictStrategy` field to `ProfileGenerator` struct
    - Update `NewProfileGenerator()` constructor to accept conflict resolution strategy
    - Integrate `ProfileConflictDetector` into the profile generation workflow
    - Add `DetectProfileConflicts()` method to identify conflicts before profile generation
    - Write unit tests for enhanced constructor and conflict detection integration
    - _Requirements: 1.1, 2.1, 2.2_

  - [x] 4.2 Implement conflict resolution orchestration
    - Add `ResolveConflicts()` method to `ProfileGenerator` to orchestrate conflict resolution
    - Implement strategy-based conflict resolution (replace, skip, prompt) logic
    - Add conflict action tracking to record all resolution decisions made
    - Implement role filtering to separate conflicted roles from non-conflicted ones
    - Write unit tests for each conflict resolution strategy with various conflict scenarios
    - _Requirements: 1.1, 3.1, 4.1, 5.1_

  - [x] 4.3 Enhance profile generation result reporting
    - Modify `ProfileGenerationResult` struct to include conflict resolution information
    - Add fields for detected conflicts, resolution actions, replaced profiles, and skipped roles
    - Implement `GenerateConflictReport()` method to create detailed operation summaries
    - Add backup path tracking in generation results for recovery reference
    - Write unit tests for enhanced result reporting with various conflict resolution outcomes
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5_

- [ ] 5: Update Command Interface

  - [ ] 5.1 Add new command line flags for conflict resolution
    - Modify `cmd/sso.go` to add `--replace-existing` and `--skip-existing` flags
    - Implement mutual exclusion validation between the two new flags
    - Add flag descriptions and help text explaining conflict resolution options
    - Update command examples to demonstrate new conflict resolution capabilities
    - Write unit tests for flag parsing and validation logic
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5_

  - [ ] 5.2 Integrate conflict resolution into command execution
    - Modify `profileGenerator()` command function to parse conflict resolution flags
    - Add conflict strategy determination logic based on provided flags
    - Update profile generator initialization to pass conflict resolution strategy
    - Integrate conflict resolution workflow into existing command execution flow
    - Write unit tests for command execution with various flag combinations
    - _Requirements: 1.1, 3.1, 4.1, 5.1_

  - [ ] 5.3 Enhance command output and reporting
    - Update profile generation output to include conflict resolution summary
    - Add detailed reporting of replaced profiles, skipped roles, and new profiles created
    - Implement progress indicators for conflict detection and resolution phases
    - Add error reporting for conflict resolution failures with recovery guidance
    - Write unit tests for enhanced command output formatting and error handling
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5_

- [ ] 6: Implement Error Handling and Edge Cases

  - [ ] 6.1 Add enhanced error types for conflict resolution
    - Create `ConflictResolutionError` type in `helpers/profile_generator_error.go`
    - Create `BackupError` type for backup and recovery operation failures
    - Add error context enrichment for conflict-specific error information
    - Implement error recovery guidance for common conflict resolution failures
    - Write unit tests for new error types and error context handling
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5_

  - [ ] 6.2 Implement robust config file parsing and validation
    - Add malformed config file detection and graceful error handling
    - Implement partial parsing recovery for configs with some invalid profiles
    - Add file permission validation before attempting modifications
    - Implement concurrent access protection using file locking mechanisms
    - Write unit tests for config file parsing edge cases and error recovery
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5_

  - [ ] 6.3 Add comprehensive operation rollback capabilities
    - Implement automatic rollback on partial failure during profile replacement
    - Add transaction-like behavior for multi-profile operations
    - Implement cleanup of temporary files and incomplete operations
    - Add detailed error reporting for rollback operations and recovery steps
    - Write unit tests for rollback scenarios and partial failure recovery
    - _Requirements: 3.4, 7.4, 7.5_

- [ ] 7: Comprehensive Testing and Validation

  - [ ] 7.1 Create comprehensive unit test suite for conflict detection
    - Test conflict detection with legacy SSO profile format
    - Test conflict detection with SSO session profile format  
    - Test conflict detection in mixed format environments
    - Test edge cases like missing SSO sessions and malformed profiles
    - Test performance with large numbers of existing profiles
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 8.1, 8.2, 8.3, 8.4, 8.5_

  - [ ] 7.2 Create integration tests for end-to-end conflict resolution workflows
    - Test complete replace strategy workflow with various conflict types
    - Test complete skip strategy workflow with multiple conflicts
    - Test interactive prompt workflow with mixed resolution choices
    - Test backup and recovery scenarios with operation failures
    - Test command line interface with all flag combinations
    - _Requirements: 1.1, 3.1, 4.1, 5.1, 6.1, 6.2, 6.3, 6.4, 6.5_

  - [ ] 7.3 Create performance and stress tests
    - Test conflict detection performance with hundreds of existing profiles
    - Test memory usage during large-scale profile operations
    - Test concurrent access scenarios with file locking
    - Test backup and restore performance with large config files
    - Benchmark conflict resolution strategies for optimization opportunities
    - _Requirements: Performance considerations from design document_

- [ ] 8: Documentation and Code Quality

  - [ ] 8.1 Add comprehensive code documentation
    - Add detailed comments to all new functions and methods
    - Document conflict resolution algorithms and design decisions
    - Add usage examples for new conflict resolution capabilities
    - Document error handling patterns and recovery procedures
    - Update existing function documentation to reflect enhanced capabilities
    - _Requirements: All requirements - documentation support_

  - [ ] 8.2 Ensure code quality and standards compliance
    - Run `go fmt` on all modified Go files
    - Run `go test ./...` to ensure all tests pass including new test suites
    - Run `make test` for comprehensive linting and testing
    - Address any golangci-lint warnings in new and modified code
    - Verify consistent error handling patterns without panic usage
    - _Requirements: Code quality standards from design document_

  - [ ] 8.3 Update command help and usage documentation
    - Update command help text to include new conflict resolution flags
    - Add practical examples demonstrating conflict resolution scenarios
    - Document best practices for profile name standardization
    - Add troubleshooting guide for common conflict resolution issues
    - Update any existing documentation that references profile generation behavior
    - _Requirements: 1.1, 6.1, 6.2, 6.3, 6.4, 6.5_