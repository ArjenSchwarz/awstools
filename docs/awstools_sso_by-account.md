---
date: 2022-04-16T20:43:03+10:00
title: "awstools sso by-account"
slug: awstools_sso_by-account
url: /awstools/awstools_sso_by-account/
---
## awstools sso by-account

A basic overview of the SSO Config Permission Sets by account

### Synopsis

Provides an overview of all the permission sets and assignments attached to an account,
	grouped by account.

You can filter the output to a single account by supplying the --resource-id (-r) flag with the account ID or, if you use a name file, the account alias from the name file.

Verbose mode will add the policies for the permissionsets in the textual output formats drawio output will generate a graph that goes SSO Instance -> Accounts -> Permission Sets -> User/Group You may notice the same permission sets shown multiple times, this is to improve readability not a bug. dot output is currently limited as it shows internal names only
	

```
awstools sso by-account [flags]
```

### Options

```
  -h, --help                 help for by-account
  -r, --resource-id string   The account id (or account alias) you want to limit to
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

