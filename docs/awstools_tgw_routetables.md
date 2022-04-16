---
date: 2022-04-16T20:43:03+10:00
title: "awstools tgw routetables"
slug: awstools_tgw_routetables
url: /awstools/awstools_tgw_routetables/
---
## awstools tgw routetables

Get an overview of connections between Transit Gateway Route Tables and attached resources

### Synopsis

Get an overview of connections between Transit Gateway Route Tables and attached resources
	This is currently limited to showing VPC attachments only, but that will be fixed soon.

	Using the --resource-id (-r) flag, you can limit the output to the provided resource.
	For a route table that means all the VPCs it connects to,
	while for a VPC that means all the route tables it connects
	to and through them what other VPCs can reach it or it can reach.

	Supports a Draw.io output

```
awstools tgw routetables [flags]
```

### Options

```
  -h, --help                 help for routetables
  -r, --resource-id string   The id of the resource you want to limit to
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
  -v, --verbose           Give verbose output
```

### SEE ALSO

* [awstools tgw](#awstools-tgw)	 - Transit Gateway commands

