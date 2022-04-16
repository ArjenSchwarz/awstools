---
date: 2022-04-16T20:43:03+10:00
title: "awstools appmesh danglingnodes"
slug: awstools_appmesh_danglingnodes
url: /awstools/awstools_appmesh_danglingnodes/
---
## awstools appmesh danglingnodes

Get all dangling nodes

### Synopsis

Get an overview of all nodes without a route or service attached to them

```
awstools appmesh danglingnodes [flags]
```

### Options

```
  -h, --help   help for danglingnodes
```

### Options inherited from parent commands

```
  -a, --append            Add to the provided output file instead of replacing it
      --config string     config file (default is .awstools.yaml in current directory, or $HOME/.awstools.yaml)
      --emoji             Use emoji in the output
  -f, --file string       Optional file to save the output to
  -m, --meshname string   The name of the mesh
  -n, --namefile string   Use this file to provide names
  -o, --output string     Format for the output, currently supported are csv, table, json, html, dot, and drawio (default "json")
      --profile string    Use a specific profile
      --region string     Use a specific region
  -v, --verbose           Give verbose output
```

### SEE ALSO

* [awstools appmesh](#awstools-appmesh)	 - App Mesh commands

