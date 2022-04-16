---
date: 2022-04-16T20:43:03+10:00
title: "awstools sso dangling"
slug: awstools_sso_dangling
url: /awstools/awstools_sso_dangling/
---
## awstools sso dangling

An overview of unassigned permission sets

### Synopsis

Lists all permission sets that aren't assigned to an account

Includes full details on the managed and inline policies.

```
awstools sso dangling [flags]
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

* [awstools sso](#awstools-sso)	 - AWS Single Sign-On commands

