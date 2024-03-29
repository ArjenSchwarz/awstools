---
date: 2022-04-16T20:43:03+10:00
title: "awstools gen docs"
slug: awstools_gen_docs
url: /awstools/awstools_gen_docs/
---
## awstools gen docs

Generate Markdown documentation for awstools

### Synopsis

Generate documentation for awstools in Markdown format
This is used for the documentation in the repository, but can be run separately. By default it will generate it in the docs directory from where you run the command, but you can override this with the --directory flag.

```
awstools gen docs [flags]
```

### Options

```
  -d, --directory string   The directory where the documentation will be generated (default "./docs")
  -h, --help               help for docs
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

