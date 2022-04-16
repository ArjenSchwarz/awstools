---
date: 2022-04-16T20:43:03+10:00
title: "awstools iam rolelist"
slug: awstools_iam_rolelist
url: /awstools/awstools_iam_rolelist/
---
## awstools iam rolelist

Get an overview of the roles and their policies

### Synopsis

Retrieves a list of all IAM roles in the account and their policies.
The policies themselves are also shown separately.

The drawio output format links the users to policies.

```
awstools iam rolelist [flags]
```

### Options

```
  -h, --help   help for rolelist
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

