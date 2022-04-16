---
date: 2022-04-16T20:43:03+10:00
title: "awstools names"
slug: awstools_names
url: /awstools/awstools_names/
---
## awstools names

Get the names for the resources in the account

### Synopsis

These names can be stored in a file and then used by other functionalities.
	This is especially useful for commands that deal with multiple accounts.

	Only outputs as JSON.

```
awstools names [flags]
```

### Options

```
  -h, --help   help for names
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

* [awstools](#awstools)	 - Various tools for dealing with complex AWS comments

