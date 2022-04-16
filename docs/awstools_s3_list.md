---
date: 2022-04-16T20:43:03+10:00
title: "awstools s3 list"
slug: awstools_s3_list
url: /awstools/awstools_s3_list/
---
## awstools s3 list

An overview of S3 buckets

### Synopsis

Lists all S3 buckets.

```
awstools s3 list [flags]
```

### Options

```
  -h, --help                  help for list
  -t, --include-tags string   Optional tag values to show in output
      --public-only           Only show public buckets
      --unencrypted-only      Only show unencrypted buckets
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

* [awstools s3](#awstools-s3)	 - S3 commands

