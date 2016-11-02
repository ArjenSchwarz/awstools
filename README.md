# awstools

awstools is a Go application that allows me to more easily do complex actions with AWS data. It doesn't try to replicate things that are already easy to do with the CLI, but instead focuses on more complex actions. This usually involves things that would take multiple CLI calls and a fair bit of scripting or manual work.

# Functionalities

Currently built in features:

 * Get a list of all IAM users, their groups, and the policies active upon them
 * Get a list of all the resources in a CloudFormation stack, including those from nested stacks

Run the application without any commands to get an overview, as shown below. For more details you can then run `awstools [command] --help`, which also works with subcommands such as `awstools cfn resources --help`.

```bash
$ awstools
awstools is designed to be used for more complex tasks that would take a lot of work using just the CLI.

This usually involves tasks that would require multiple calls.

Usage:
  awstools [command]

Available Commands:
  cfn         CloudFormation commands
  iam         IAM commands

Flags:
  -f, --file string     Optional file to save the output to
  -h, --help            help for awstools
  -o, --output string   Format for the output, currently supported are csv and json (default "json")
  -v, --verbose         Give verbose output

Use "awstools [command] --help" for more information about a command.
```

Output options at the moment are either csv or json, with json being the default so you can easily pass it to a tool like [jq](https://stedolan.github.io/jq/). It's also possible to directly save the output into a file. Most commands will have a verbose option that will show some additional information that you often won't need.

# Installation and configuration

Simply download the [latest release][latest] for your platform, and you can use it. You can place it somewhere in your $PATH to ensure you can run it from anywhere.

The AWS configuration is read from the standard locations:

* Your environment variables (`AWS_ACCESS_KEY`, `AWS_SECRET_ACCESS_KEY`, etc.).
* The values in your `~/.aws/credentials` file.
* Permissions from the IAM role the application has access to (when running on AWS)

[latest]: https://github.com/ArjenSchwarz/awstools/releases

# Development

awstools uses the Cobra framework for ease of development. While I will usually only build the functionalities that I need at a certain time, feel free to request or add features.

If you wish to contribute you can always create Issues or Pull Requests. For Pull Request, just follow the standard pattern.

1. Fork the repository
2. Make your changes
3. Make a pull request that explains what it does
