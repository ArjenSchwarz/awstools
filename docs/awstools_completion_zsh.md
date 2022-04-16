---
date: 2022-04-16T20:43:03+10:00
title: "awstools completion zsh"
slug: awstools_completion_zsh
url: /awstools/awstools_completion_zsh/
---
## awstools completion zsh

Generate the autocompletion script for zsh

### Synopsis

Generate the autocompletion script for the zsh shell.

If shell completion is not already enabled in your environment you will need
to enable it.  You can execute the following once:

	echo "autoload -U compinit; compinit" >> ~/.zshrc

To load completions for every new session, execute once:

#### Linux:

	awstools completion zsh > "${fpath[1]}/_awstools"

#### macOS:

	awstools completion zsh > /usr/local/share/zsh/site-functions/_awstools

You will need to start a new shell for this setup to take effect.


```
awstools completion zsh [flags]
```

### Options

```
  -h, --help              help for zsh
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

