package cmd

// Common column names used across commands
const (
	nameColumn          = "Name"
	childrenColumn      = "Children"
	permissionSetColumn = "PermissionSet"
	accountColumn       = "Account"
	accountIDColumn     = "AccountID"
	attachmentColumn    = "Attachment"
	cidrColumn          = "CIDR"
	descriptionColumn   = "Description"
	destinationsColumn  = "Destinations"
	drawIOIDColumn      = "DrawIOID"
	fieldColumn         = "Field"
	imageColumn         = "Image"
	routeTableColumn    = "Route Table"
	routesColumn        = "Routes"
	subnetColumn        = "Subnet"
	typeColumn          = "Type"
	vpcColumn           = "VPC"
	stackColumn         = "Stack"
	exportColumn        = "Export"
	valueColumn         = "Value"
	targetGatewayColumn = "TargetGateway"
	importedColumn      = "Imported"
)

// Common literal values used across commands
const (
	s3BucketARNDescription = "ARN of the S3 bucket"
)

// Resource type constants
const (
	vpcResourceType = "vpc"
	tgwResourceType = "tgw"
)
