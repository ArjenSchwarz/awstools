# Technology Stack

## Core Technologies
- **Language**: Go 1.24.1+
- **CLI Framework**: Cobra (github.com/spf13/cobra)
- **Configuration**: Viper (github.com/spf13/viper) - supports YAML, JSON, TOML
- **AWS SDK**: aws-sdk-go-v2 (latest v2 SDK with SSO support)
- **Output Formatting**: github.com/ArjenSchwarz/go-output (custom library)
- **Testing**: testify (github.com/stretchr/testify)

## Build System & Commands

### Makefile Targets
```bash
# Primary development workflow
make build          # Run tests, clean, and compile
make test           # Run tests only  
make lint           # Run golangci-lint
make compile        # Build binary only
make clean          # Clean build artifacts

# Manual commands
go build            # Build the application
go test ./...       # Run all tests
golangci-lint run   # Run linting (requires golangci-lint v2.2.1+)
go fmt ./...        # Format code (auto-run after .go file changes)
```

### Code Quality
- **Linter**: golangci-lint v2.2.1+ with comprehensive rule set
- **Enabled linters**: errcheck, govet, ineffassign, misspell, staticcheck, unused, revive, goconst, gocritic, unconvert
- **Testing**: Unit tests with testify assertions
- **Formatting**: Standard Go formatting with `go fmt`

## Configuration Management
- **Config files**: `.awstools.yaml` (or .json/.toml) in current directory or $HOME
- **Environment variables**: Standard AWS env vars (AWS_PROFILE, AWS_REGION, etc.)
- **CLI flags**: Override config values at runtime
- **AWS credentials**: Supports AWS CLI v2 SSO sessions, standard credential chain

## Output Formats
- **Base formats**: json (default), csv, table, html, markdown
- **Graphical formats**: dot (graphviz), drawio (diagrams.net), mermaid (planned)
- **Table styles**: Configurable via demo command, default similar to AWS CLI

## Dependencies
- AWS SDK v2 services: ec2, iam, organizations, s3, sso, ssoadmin, sts, cloudformation, appmesh, rds
- CLI utilities: Cobra for commands, Viper for config
- Output formatting: Custom go-output library for consistent formatting
- Utilities: mitchellh/go-homedir for cross-platform home directory