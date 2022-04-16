---
date: 2022-04-16T20:43:03+10:00
title: "awstools iam userlist"
slug: awstools_iam_userlist
url: /awstools/awstools_iam_userlist/
---
## awstools iam userlist

Get an overview of the IAM users in the account

### Synopsis

Retrieves a list of all IAM users in the account and the groups they are in.
It also shows the policies they have through either the group or directly. The groups themselves are shown separately, as are policies when using the verbose flag.

The drawio output format links the users to groups and (in verbose mode) both of those to the policies.

```
awstools iam userlist [flags]
```

### Options

```
  -h, --help   help for userlist
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

* [awstools iam](#awstools-iam)	 - IAM commands

