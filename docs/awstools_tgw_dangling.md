---
date: 2022-04-16T20:43:03+10:00
title: "awstools tgw dangling"
slug: awstools_tgw_dangling
url: /awstools/awstools_tgw_dangling/
---
## awstools tgw dangling

Check for incomplete routes

### Synopsis

Check for incomplete routes.

	An incomplete route is defined as one that goes in only a single
	direction. e.g. while VPC1 connects to VPC2, there is no returning
	connection.

```
awstools tgw dangling [flags]
```

### Options

```
  -h, --help   help for dangling
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

