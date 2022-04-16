---
date: 2022-04-16T20:43:03+10:00
title: "awstools gen ghpages"
slug: awstools_gen_ghpages
url: /awstools/awstools_gen_ghpages/
---
## awstools gen ghpages

Generate documentation for awstools' gh pages site

### Synopsis

Generate documentation for awstools in Markdown format
This is used for the documentation in the GitHub Pages site, but can be run separately.

```
awstools gen ghpages [flags]
```

### Options

```
  -h, --help   help for ghpages
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

* [awstools gen](#awstools-gen)	 - Generate various useful things for awstools

