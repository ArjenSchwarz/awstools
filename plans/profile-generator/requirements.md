# Requirements Document

## Introduction

The profile-generator feature enables AWS CLI users to automatically generate AWS CLI profiles for all assumable roles in AWS IAM Identity Center. By providing an existing profile configuration and a naming pattern, users can discover and configure all available roles without manual enumeration. This streamlines the setup process for multi-account AWS environments using IAM Identity Center.

## Requirements

### Requirement 1
**User Story:** As an AWS CLI user, I want to provide an existing IAM Identity Center profile as a template, so that the system can use its authentication configuration to discover available roles.

#### Acceptance Criteria
1. WHEN a user specifies an existing profile name THEN the system SHALL validate that the profile exists in the AWS CLI configuration
2. IF the specified profile uses IAM Identity Center authentication THEN the system SHALL extract the SSO configuration details
3. WHEN the profile validation fails THEN the system SHALL display a clear error message indicating the profile was not found or is invalid
4. IF the profile does not use IAM Identity Center THEN the system SHALL reject the profile with an appropriate error message

### Requirement 2
**User Story:** As an AWS CLI user, I want to specify a naming pattern for generated profiles, so that I can maintain consistent profile naming conventions across my configuration.

#### Acceptance Criteria
1. WHEN a user provides a naming pattern THEN the system SHALL accept patterns with placeholders for account ID, account name, and role name
2. IF the naming pattern contains invalid characters THEN the system SHALL reject the pattern with validation errors
3. WHEN generating profile names THEN the system SHALL substitute placeholders with actual values from discovered roles
4. IF duplicate profile names would be generated THEN the system SHALL append a unique identifier to prevent conflicts

### Requirement 3
**User Story:** As an AWS CLI user, I want the system to discover all assumable roles using my IAM Identity Center access, so that I don't have to manually identify each role.

#### Acceptance Criteria
1. WHEN the system authenticates with IAM Identity Center THEN it SHALL enumerate all accessible accounts
2. IF authentication fails THEN the system SHALL display appropriate error messages and exit gracefully
3. WHEN enumerating accounts THEN the system SHALL list all permission sets (roles) available in each account
4. IF no assumable roles are found THEN the system SHALL inform the user that no roles were discovered

### Requirement 4
**User Story:** As an AWS CLI user, I want to see a preview of all generated profiles before they are added to my configuration, so that I can review and approve the changes.

#### Acceptance Criteria
1. WHEN profile generation is complete THEN the system SHALL display all generated profiles in a readable format
2. IF the user chooses to preview THEN the system SHALL show the complete profile configuration for each generated profile
3. WHEN displaying profiles THEN the system SHALL include account information, role names, and SSO configuration details
4. IF no profiles would be generated THEN the system SHALL inform the user and exit without making changes

### Requirement 5
**User Story:** As an AWS CLI user, I want the option to append generated profiles to my existing AWS CLI configuration, so that I can easily integrate them into my current setup.

#### Acceptance Criteria
1. WHEN the user confirms to append profiles THEN the system SHALL add all generated profiles to the AWS CLI configuration file
2. IF the configuration file is read-only THEN the system SHALL display appropriate error messages about file permissions
3. WHEN appending profiles THEN the system SHALL preserve existing configuration and formatting
4. IF a profile with the same name already exists THEN the system SHALL ask for user confirmation before overwriting
5. WHEN the append operation completes successfully THEN the system SHALL display a summary of added profiles

### Requirement 6
**User Story:** As an AWS CLI user, I want the system to handle errors gracefully during the profile generation process, so that I receive clear feedback about any issues.

#### Acceptance Criteria
1. WHEN AWS API calls fail THEN the system SHALL display meaningful error messages with suggested remediation steps
2. IF network connectivity issues occur THEN the system SHALL inform the user about connection problems
3. WHEN IAM Identity Center tokens expire THEN the system SHALL prompt the user to re-authenticate
4. IF insufficient permissions are detected THEN the system SHALL specify which permissions are missing