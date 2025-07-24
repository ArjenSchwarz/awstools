# Requirements Document

## Introduction

The profile-generator enhancement extends the existing awstools profile generation feature to intelligently handle existing AWS CLI profiles when generating new ones. Users can now specify whether to replace existing profiles with new names based on the provided pattern or skip them entirely, providing better control over profile management in existing AWS CLI configurations.

## Requirements

### Requirement 1

**User Story:** As an AWS CLI user, I want to specify how existing profiles should be handled during profile generation, so that I can control whether they are replaced or preserved.

#### Acceptance Criteria
1. WHEN a user runs the profile generator THEN the system SHALL provide a flag to control existing profile behavior
2. IF the user specifies `--replace-existing` flag THEN the system SHALL replace existing profiles with new names based on the pattern
3. IF the user specifies `--skip-existing` flag THEN the system SHALL skip generating profiles for roles that already have profiles
4. WHEN no existing profile handling flag is provided THEN the system SHALL default to prompting the user for each conflict
5. IF both flags are provided THEN the system SHALL reject the command with a validation error

### Requirement 2

**User Story:** As an AWS CLI user, I want the system to detect existing profiles that correspond to the same AWS role, so that I can avoid duplicate profile configurations.

#### Acceptance Criteria
1. WHEN the system discovers a role THEN it SHALL check if a profile already exists for that role
2. IF a profile exists with the same SSO account ID and role name THEN the system SHALL identify it as an existing profile
3. WHEN checking for existing profiles THEN the system SHALL match based on SSO configuration rather than profile name
4. IF multiple profiles exist for the same role THEN the system SHALL identify all matching profiles
5. WHEN a role has existing profiles THEN the system SHALL log the existing profile names for user reference

### Requirement 3

**User Story:** As an AWS CLI user, I want to replace existing profiles with new names based on my naming pattern, so that I can standardize my profile naming conventions.

#### Acceptance Criteria
1. WHEN `--replace-existing` is specified AND an existing profile is found THEN the system SHALL remove the old profile
2. IF the new profile name matches the existing profile name THEN the system SHALL update the profile in place
3. WHEN replacing a profile THEN the system SHALL preserve any custom configuration not related to SSO authentication
4. IF profile replacement fails THEN the system SHALL restore the original profile and report the error
5. WHEN replacing profiles THEN the system SHALL create a backup of the original configuration file

### Requirement 4

**User Story:** As an AWS CLI user, I want to skip generating profiles for roles that already have profiles, so that I can preserve my existing profile configurations.

#### Acceptance Criteria
1. WHEN `--skip-existing` is specified AND an existing profile is found THEN the system SHALL not generate a new profile for that role
2. IF skipping existing profiles THEN the system SHALL report which roles were skipped
3. WHEN skipping profiles THEN the system SHALL continue processing other roles normally
4. IF all discovered roles have existing profiles THEN the system SHALL inform the user that no new profiles were generated
5. WHEN skipping profiles THEN the system SHALL provide a summary of skipped vs generated profiles

### Requirement 5

**User Story:** As an AWS CLI user, I want to be prompted for each profile conflict when no handling preference is specified, so that I can make individual decisions about each existing profile.

#### Acceptance Criteria
1. WHEN no existing profile handling flag is provided AND a conflict is detected THEN the system SHALL prompt the user for action
2. IF the user chooses to replace THEN the system SHALL replace that specific profile
3. IF the user chooses to skip THEN the system SHALL skip generating a profile for that role
4. WHEN prompting for conflicts THEN the system SHALL show the existing profile name and the proposed new profile name
5. IF the user cancels the prompt THEN the system SHALL exit without making any changes

### Requirement 6

**User Story:** As an AWS CLI user, I want to see a detailed report of profile generation actions, so that I understand what changes were made to my configuration.

#### Acceptance Criteria
1. WHEN profile generation completes THEN the system SHALL display a summary of all actions taken
2. IF profiles were replaced THEN the system SHALL list the old and new profile names
3. IF profiles were skipped THEN the system SHALL list the skipped roles and their existing profile names
4. WHEN new profiles are created THEN the system SHALL list the newly created profile names
5. IF no changes were made THEN the system SHALL clearly indicate that no profiles were modified

### Requirement 7

**User Story:** As an AWS CLI user, I want the system to handle edge cases gracefully during existing profile detection, so that the profile generation process is robust and reliable.

#### Acceptance Criteria
1. WHEN the AWS config file is malformed THEN the system SHALL report parsing errors and exit gracefully
2. IF the config file is read-only THEN the system SHALL report permission errors before attempting modifications
3. WHEN profile matching fails due to incomplete SSO configuration THEN the system SHALL log warnings and continue processing
4. IF backup creation fails THEN the system SHALL abort the operation and report the error
5. WHEN profile replacement partially fails THEN the system SHALL restore the backup and report which profiles failed

### Requirement 8

**User Story:** As an AWS CLI user, I want the existing profile detection to work with both legacy and modern SSO profile formats, so that the feature works regardless of my AWS CLI version.

#### Acceptance Criteria
1. WHEN detecting existing profiles THEN the system SHALL recognize both legacy SSO format and SSO session format
2. IF a profile uses legacy format (`sso_start_url`, `sso_region`, etc.) THEN the system SHALL match based on those fields
3. IF a profile uses session format (`sso_session`) THEN the system SHALL resolve the session and match based on the resolved configuration
4. WHEN matching profiles THEN the system SHALL normalize both formats for comparison
5. IF profile format cannot be determined THEN the system SHALL log a warning and skip that profile for matching