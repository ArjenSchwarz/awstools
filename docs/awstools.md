## awstools

Various tools for dealing with complex AWS comments

### Synopsis

awstools is designed to be used for more complex tasks that would take a lot of work using just the CLI.

This usually involves tasks that would require multiple calls.

Full documentation for all commands can be accessed using the --help flag or by reading it on https://github.com/ArjenSchwarz/awstools/blob/main/docs/awstools.md


### Options

```
  -a, --append            Add to the provided output file instead of replacing it
  -f, --file string       Optional file to save the output to
  -h, --help              help for awstools
  -n, --namefile string   Use this file to provide names
  -o, --output string     Format for the output, currently supported are csv, json, html, dot, and drawio (default "json")
      --profile string    Use a specific profile
      --region string     Use a specific region
  -v, --verbose           Give verbose output
```

### SEE ALSO

* [awstools appmesh](awstools_appmesh.md)	 - App Mesh commands
* [awstools cfn](awstools_cfn.md)	 - CloudFormation commands
* [awstools gen](awstools_gen.md)	 - Generate various useful things for awstools
* [awstools iam](awstools_iam.md)	 - IAM commands
* [awstools names](awstools_names.md)	 - Get the names for the resources in the account
* [awstools organizations](awstools_organizations.md)	 - AWS Organizations commands
* [awstools sso](awstools_sso.md)	 - AWS Single Sign-On commands
* [awstools tgw](awstools_tgw.md)	 - Transit Gateway commands
* [awstools vpc](awstools_vpc.md)	 - VPC commands

###### Auto generated by spf13/cobra on 22-Mar-2021
