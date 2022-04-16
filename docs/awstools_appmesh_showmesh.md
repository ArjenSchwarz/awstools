---
date: 2022-04-16T20:43:03+10:00
title: "awstools appmesh showmesh"
slug: awstools_appmesh_showmesh
url: /awstools/awstools_appmesh_showmesh/
---
## awstools appmesh showmesh

Show the connections between virtual nodes

### Synopsis

You can see which nodes are allowed access to which other nodes based on the current App Mesh configuration.

Example:

	awstools appmesh showmesh -m bookinfo-mesh -o dot | dot -Tpng  -o bookinfo-mesh.png
	awstools appmesh showmesh -m bookinfo-mesh -o drawio | pbcopy

Using the dot output format you can turn this into an image, and using drawio you will get a CSV that you can import into draw.io with its CSV import functionality


```
awstools appmesh showmesh [flags]
```

### Options

```
  -h, --help   help for showmesh
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

