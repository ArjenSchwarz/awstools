---
date: 2022-04-16T20:43:03+10:00
title: "awstools sso list-permission-sets"
slug: awstools_sso_list-permission-sets
url: /awstools/awstools_sso_list-permission-sets/
---
## awstools sso list-permission-sets

A list of the SSO Permission Sets

### Synopsis

Provides an overview of all the permission sets and their attached policies and deployed accounts

By default this command gives an output showing the number of managed policies attached and whether it has an inline policy. To expand this and see the details, use the --verbose (-v) flag.
	

```
awstools sso list-permission-sets [flags]
```

### Options

```
  -h, --help   help for list-permission-sets
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

