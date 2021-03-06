## awstools vpc peerings

Get VPC Peerings

### Synopsis

Get an overview of Peerings. For a graphical option consider using
	the dot or drawio output formats.

	awstools vpc peerings -o dot | dot -Tpng  -o peerings.png
	awstools vpc peerings -o drawio | pbcopy

```
awstools vpc peerings [flags]
```

### Options

```
  -h, --help   help for peerings
```

### Options inherited from parent commands

```
  -a, --append            Add to the provided output file instead of replacing it
  -f, --file string       Optional file to save the output to
  -n, --namefile string   Use this file to provide names
  -o, --output string     Format for the output, currently supported are csv, json, html, dot, and drawio (default "json")
      --profile string    Use a specific profile
      --region string     Use a specific region
  -v, --verbose           Give verbose output
```

### SEE ALSO

* [awstools vpc](awstools_vpc.md)	 - VPC commands

###### Auto generated by spf13/cobra on 22-Mar-2021
