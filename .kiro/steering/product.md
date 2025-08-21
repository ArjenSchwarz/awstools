# Product Overview

awstools is a Go CLI application designed for complex AWS operations that would require multiple AWS CLI calls and manual scripting. It focuses on multi-service analysis, cross-account data aggregation, and visualization of AWS infrastructure relationships.

## Core Purpose
- Simplify complex AWS tasks that are cumbersome with standard AWS CLI
- Provide comprehensive analysis across multiple AWS services
- Generate visual diagrams and reports for infrastructure understanding
- Support multi-account and cross-region data collection

## Key Features
- **Multi-format output**: JSON, CSV, table, HTML, markdown, dot (graphviz), drawio
- **Cross-account aggregation**: Collect and combine data from multiple AWS accounts
- **Visual diagrams**: Generate infrastructure diagrams for App Mesh, Organizations, SSO, Transit Gateway
- **Resource naming**: Human-readable names via naming files with fallback to resource IDs
- **Comprehensive analysis**: Deep-dive into VPC usage, IAM relationships, SSO permissions, etc.

## Target Users
AWS engineers and architects who need to understand complex infrastructure relationships and perform analysis that spans multiple AWS services and accounts.