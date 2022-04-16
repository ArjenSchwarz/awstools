---
date: 2022-04-16T20:43:03+10:00
title: "awstools completion fish"
slug: awstools_completion_fish
url: /awstools/awstools_completion_fish/
---
## awstools completion fish

Generate the autocompletion script for fish

### Synopsis

Generate the autocompletion script for the fish shell.

To load completions in your current shell session:

	awstools completion fish | source

To load completions for every new session, execute once:

	awstools completion fish > ~/.config/fish/completions/awstools.fish

You will need to start a new shell for this setup to take effect.


```
awstools completion fish [flags]
```

### Options

```
  -h, --help              help for fish
      --no-descriptions   disable completion descriptions
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

* [awstools completion](#awstools-completion)	 - Generate the autocompletion script for the specified shell

