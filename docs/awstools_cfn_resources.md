---
date: 2022-04-16T20:43:03+10:00
title: "awstools cfn resources"
slug: awstools_cfn_resources
url: /awstools/awstools_cfn_resources/
---
## awstools cfn resources

List all the resources in a stack and any nested stacks

### Synopsis

This command will list the resources attached to the provided stack and any possible nested stacks.

Return values are the ResourceID, Type, and Stack of the resource. You can use the --namefile flag to show names instead of resource ids.

--verbose will add the status and logicalname (the nme within the stack) to the output

```
awstools cfn resources [flags]
```

### Options

```
  -h, --help   help for resources
```

### Options inherited from parent commands

```
  -a, --append            Add to the provided output file instead of replacing it
      --config string     config file (default is .awstools.yaml in current directory, or $HOME/.awstools.yaml)
      --emoji             Use emoji in the output
  -f, --file string       Optional file to save the output to
  -n, --namefile string   Use this file to provide names
  -o, --output string     Format for the output, currently supported are csv, table, json, html, dot, and drawio (default "json")
      --profile string    Use a specific profile
      --region string     Use a specific region
  -s, --stack string      The name of the stack
  -v, --verbose           Give verbose output
```

### SEE ALSO

* [awstools cfn](#awstools-cfn)	 - CloudFormation commands

