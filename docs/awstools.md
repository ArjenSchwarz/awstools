---
date: 2022-04-16T20:43:03+10:00
title: "awstools"
slug: awstools
url: /awstools/awstools/
---
## awstools

Various tools for dealing with complex AWS comments

### Synopsis

awstools is designed to be used for more complex tasks that would take a lot of work using just the CLI.

This usually involves tasks that would require multiple calls.

Full documentation for all commands can be accessed using the --help flag or by reading it on https://github.com/ArjenSchwarz/awstools/blob/main/docs/awstools.md


### Options

```
  -a, --append            Add to the provided output file instead of replacing it
      --config string     config file (default is .awstools.yaml in current directory, or $HOME/.awstools.yaml)
      --emoji             Use emoji in the output
  -f, --file string       Optional file to save the output to
  -h, --help              help for awstools
  -n, --namefile string   Use this file to provide names
  -o, --output string     Format for the output, currently supported are csv, table, json, html, dot, and drawio (default "json")
      --profile string    Use a specific profile
      --region string     Use a specific region
  -v, --verbose           Give verbose output
```

### SEE ALSO

* [awstools appmesh](#awstools-appmesh)	 - App Mesh commands
* [awstools cfn](#awstools-cfn)	 - CloudFormation commands
* [awstools completion](#awstools-completion)	 - Generate the autocompletion script for the specified shell
* [awstools demo](#awstools-demo)	 - See demos for awstools
* [awstools gen](#awstools-gen)	 - Generate various useful things for awstools
* [awstools iam](#awstools-iam)	 - IAM commands
* [awstools names](#awstools-names)	 - Get the names for the resources in the account
* [awstools organizations](#awstools-organizations)	 - AWS Organizations commands
* [awstools s3](#awstools-s3)	 - S3 commands
* [awstools sso](#awstools-sso)	 - AWS Single Sign-On commands
* [awstools tgw](#awstools-tgw)	 - Transit Gateway commands
* [awstools version](#awstools-version)	 - Show the version number
* [awstools vpc](#awstools-vpc)	 - VPC commands

