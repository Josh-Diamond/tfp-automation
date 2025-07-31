package aws

import (
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/rancher/tfp-automation/config"
	"github.com/rancher/tfp-automation/defaults/resourceblocks/nodeproviders/amazon"
	"github.com/rancher/tfp-automation/framework/format"
	"github.com/rancher/tfp-automation/framework/set/defaults"
	"github.com/zclconf/go-cty/cty"
)

// SetAWSRKE1Provider is a helper function that will set the AWS RKE1
// Terraform configurations in the main.tf file.
func SetAWSRKE1Provider(nodeTemplateBlockBody *hclwrite.Body, terraformConfig *config.TerraformConfig) {
	awsConfigBlock := nodeTemplateBlockBody.AppendNewBlock(amazon.EC2Config, nil)
	awsConfigBlockBody := awsConfigBlock.Body()

	awsConfigBlockBody.SetAttributeValue(defaults.AccessKey, cty.StringVal(terraformConfig.AWSCredentials.AWSAccessKey))
	awsConfigBlockBody.SetAttributeValue(defaults.SecretKey, cty.StringVal(terraformConfig.AWSCredentials.AWSSecretKey))
	awsConfigBlockBody.SetAttributeValue(defaults.Region, cty.StringVal(terraformConfig.AWSConfig.Region))

	awsConfigBlockBody.SetAttributeValue(amazon.AMI, cty.StringVal(terraformConfig.AWSConfig.AMI))
	awsConfigBlockBody.SetAttributeValue(amazon.InstanceType, cty.StringVal(terraformConfig.AWSConfig.AWSInstanceType))
	awsConfigBlockBody.SetAttributeValue(amazon.SSHUser, cty.StringVal(terraformConfig.AWSConfig.AWSUser))
	awsConfigBlockBody.SetAttributeValue(amazon.VolumeType, cty.StringVal(terraformConfig.AWSConfig.AWSVolumeType))
	awsConfigBlockBody.SetAttributeValue(amazon.RootSize, cty.NumberIntVal(terraformConfig.AWSConfig.AWSRootSize))

	securityGroups := format.ListOfStrings(terraformConfig.AWSConfig.AWSSecurityGroupNames)
	awsConfigBlockBody.SetAttributeRaw(amazon.SecurityGroup, securityGroups)

	awsConfigBlockBody.SetAttributeValue(amazon.SubnetID, cty.StringVal(terraformConfig.AWSConfig.AWSSubnetID))
	awsConfigBlockBody.SetAttributeValue(amazon.VPCID, cty.StringVal(terraformConfig.AWSConfig.AWSVpcID))
	awsConfigBlockBody.SetAttributeValue(amazon.Zone, cty.StringVal(terraformConfig.AWSConfig.AWSZoneLetter))
}

// SetAWSRKE2K3SProvider is a helper function that will set the AWS RKE2/K3S
// Terraform provider details in the main.tf file.
func SetAWSRKE2K3SProvider(rootBody *hclwrite.Body, terraformConfig *config.TerraformConfig) {
	cloudCredBlock := rootBody.AppendNewBlock(defaults.Resource, []string{defaults.CloudCredential, terraformConfig.ResourcePrefix})
	cloudCredBlockBody := cloudCredBlock.Body()

	provider := hclwrite.Tokens{
		{Type: hclsyntax.TokenIdent, Bytes: []byte(defaults.Rancher2 + "." + defaults.StandardUser)},
	}

	cloudCredBlockBody.SetAttributeRaw(defaults.Provider, provider)
	cloudCredBlockBody.SetAttributeValue(defaults.ResourceName, cty.StringVal(terraformConfig.ResourcePrefix))

	awsCredBlock := cloudCredBlockBody.AppendNewBlock(amazon.EC2CredentialConfig, nil)
	awsCredBlockBody := awsCredBlock.Body()

	awsCredBlockBody.SetAttributeValue(defaults.AccessKey, cty.StringVal(terraformConfig.AWSCredentials.AWSAccessKey))
	awsCredBlockBody.SetAttributeValue(defaults.SecretKey, cty.StringVal(terraformConfig.AWSCredentials.AWSSecretKey))
}
