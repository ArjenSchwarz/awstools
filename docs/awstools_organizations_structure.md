---
date: 2022-04-16T20:43:03+10:00
title: "awstools organizations structure"
slug: awstools_organizations_structure
url: /awstools/awstools_organizations_structure/
---
## awstools organizations structure

Get a graphical overview of the Organization's structure

### Synopsis

This command provides a graphical overview of how the accounts are connected.

Examples:

	awstools organizations structure -o dot | dot -Tpng -o structure.png
	awstools organizations structure -o drawio | pbcopy

Using the dot output format you can turn this into an image, and using drawio you will get a CSV that you can import into draw.io with its CSV import functionality. 

```
awstools organizations structure [flags]
```

### Options

```
  -h, --help   help for structure
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

* [awstools organizations](#awstools-organizations)	 - AWS Organizations commands

