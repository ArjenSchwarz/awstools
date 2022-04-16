---
date: 2022-04-16T20:43:03+10:00
title: "awstools completion bash"
slug: awstools_completion_bash
url: /awstools/awstools_completion_bash/
---
## awstools completion bash

Generate the autocompletion script for bash

### Synopsis

Generate the autocompletion script for the bash shell.

This script depends on the 'bash-completion' package.
If it is not installed already, you can install it via your OS's package manager.

To load completions in your current shell session:

	source <(awstools completion bash)

To load completions for every new session, execute once:

#### Linux:

	awstools completion bash > /etc/bash_completion.d/awstools

#### macOS:

	awstools completion bash > /usr/local/etc/bash_completion.d/awstools

You will need to start a new shell for this setup to take effect.


```
awstools completion bash
```

### Options

```
  -h, --help              help for bash
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

