---
date: 2022-04-16T20:43:03+10:00
title: "awstools tgw overview"
slug: awstools_tgw_overview
url: /awstools/awstools_tgw_overview/
---
## awstools tgw overview

A basic overview of the Transit Gateway

### Synopsis

Provides an overview of all the route tables and routes in the Transit Gateway.
This can be improved on, but offers a simple text based overview with all relevant information

If you choose the drawio output instead, you get a simple diagram showing the Transit Gateway and all resources (VPCs, VPNs, Direct Connect) attached to it.
	

```
awstools tgw overview [flags]
```

### Options

```
  -h, --help   help for overview
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

