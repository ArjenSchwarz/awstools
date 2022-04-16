---
date: 2022-04-16T20:43:03+10:00
title: "awstools gen completion"
slug: awstools_gen_completion
url: /awstools/awstools_gen_completion/
---
## awstools gen completion

Generate completion script

### Synopsis

To load completions:

Bash:

	$ source <(awstools gen completion bash)

To load completions for each session, execute once:
Linux:

	$ awstools gen completion bash > /etc/bash_completion.d/awstools

macOS:

	$ awstools gen completion bash > /usr/local/etc/bash_completion.d/awstools

Zsh:

If shell completion is not already enabled in your environment, you will need to enable it.  You can execute the following once:

	$ echo "autoload -U compinit; compinit" >> ~/.zshrc

To load completions for each session, execute once:

	$ awstools gen completion zsh > "${fpath[1]}/_awstools"

You will need to start a new shell for this setup to take effect.

fish:

	$ awstools gen completion fish | source

To load completions for each session, execute once:

	$ awstools gen completion fish > ~/.config/fish/completions/awstools.fish

PowerShell:

	PS> awstools gen completion powershell | Out-String | Invoke-Expression

To load completions for every new session, run:

	PS> awstools gen completion powershell > awstools.ps1

and source this file from your PowerShell profile.


```
awstools gen completion [bash|zsh|fish|powershell]
```

### Options

```
  -h, --help   help for completion
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

