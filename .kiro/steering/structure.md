# Project Structure

## Directory Organization

### Core Application
- **`main.go`** - Entry point, delegates to cmd package
- **`cmd/`** - CLI command definitions using Cobra framework
  - `root.go` - Base command with global flags and configuration
  - Service-specific command files (e.g., `vpc.go`, `iam.go`, `s3.go`)
  - Individual command implementations (e.g., `vpcroutes.go`, `iamrolelist.go`)
- **`helpers/`** - Core business logic and AWS SDK interactions
  - Service-specific helper files (e.g., `ec2.go`, `iam.go`, `organizations.go`)
  - Corresponding test files (`*_test.go`)
- **`config/`** - Configuration management using Viper
  - `config.go` - Configuration struct and methods
  - `awsconfig.go` - AWS-specific configuration handling

### Documentation & Build
- **`docs/`** - Auto-generated documentation (Hugo-based)
  - `content/commands/` - Generated command documentation
  - `images/` - Diagram examples and screenshots
- **`Makefile`** - Build automation and development commands
- **`.golangci.yml`** - Linting configuration

### Development & Planning
- **`plans/`** - Feature planning and design documents
  - Organized by feature (e.g., `ip-usage-finder/`, `profile-generator/`)
  - Contains `requirements.md`, `design.md`, `tasks.md` per feature

## Architecture Patterns

### Command Structure
- Each AWS service has a parent command (e.g., `vpc`, `iam`, `sso`)
- Subcommands implement specific functionality (e.g., `vpc routes`, `iam rolelist`)
- Commands follow pattern: `awstools [service] [action] [flags]`

### Helper Organization
- Business logic separated from CLI concerns
- Each AWS service has dedicated helper file
- Helpers handle AWS SDK interactions and data processing
- Return structured data that commands format for output

### Configuration Hierarchy
1. CLI flags (highest priority)
2. Config file settings
3. Environment variables
4. Default values (lowest priority)

### Error Handling
- Panic on AWS API errors (fail-fast approach)
- Graceful handling of missing resources
- Comprehensive error context in helper functions

### Testing Strategy
- Unit tests for helper functions using testify
- Test files alongside source files (`*_test.go`)
- Focus on business logic testing over CLI integration

### Output Consistency
- All commands use shared output formatting via go-output library
- Consistent flag handling across all commands
- Support for multiple output formats with same data structures

## Naming Conventions
- **Files**: lowercase with underscores (e.g., `vpc_routes.go`)
- **Functions**: CamelCase with service prefix (e.g., `GetAllVPCRouteTables`)
- **Structs**: CamelCase descriptive names (e.g., `VPCRouteTable`, `TransitGateway`)
- **Constants**: camelCase for private, CamelCase for public