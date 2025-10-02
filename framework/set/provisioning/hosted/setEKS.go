package hosted

import (
	"os"
	"strconv"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/rancher/tfp-automation/config"
	"github.com/rancher/tfp-automation/defaults/resourceblocks/nodeproviders/amazon"
	format "github.com/rancher/tfp-automation/framework/format"
	"github.com/rancher/tfp-automation/framework/set/defaults"
	resources "github.com/rancher/tfp-automation/framework/set/resources/rancher2"
	"github.com/zclconf/go-cty/cty"
)

// SetEKS is a function that will set the EKS configurations in the main.tf file.
func SetEKS(terraformConfig *config.TerraformConfig, terratestConfig *config.TerratestConfig, newFile *hclwrite.File, rootBody *hclwrite.Body,
	file *os.File) (*hclwrite.File, *os.File, error) {
	cloudCredBlock := rootBody.AppendNewBlock(defaults.Resource, []string{defaults.CloudCredential, defaults.CloudCredential})
	cloudCredBlockBody := cloudCredBlock.Body()

	cloudCredBlockBody.SetAttributeValue(defaults.ResourceName, cty.StringVal(terraformConfig.ResourcePrefix))

	ec2CredConfigBlock := cloudCredBlockBody.AppendNewBlock(amazon.EC2CredentialConfig, nil)
	ec2CredConfigBlockBody := ec2CredConfigBlock.Body()

	ec2CredConfigBlockBody.SetAttributeValue(defaults.AccessKey, cty.StringVal(terraformConfig.AWSCredentials.AWSAccessKey))
	ec2CredConfigBlockBody.SetAttributeValue(defaults.SecretKey, cty.StringVal(terraformConfig.AWSCredentials.AWSSecretKey))

	rootBody.AppendNewline()

	clusterBlock := rootBody.AppendNewBlock(defaults.Resource, []string{defaults.Cluster, defaults.Cluster})
	clusterBlockBody := clusterBlock.Body()

	clusterBlockBody.SetAttributeValue(defaults.ResourceName, cty.StringVal(terraformConfig.ResourcePrefix))

	eksConfigBlock := clusterBlockBody.AppendNewBlock(amazon.EKSConfig, nil)
	eksConfigBlockBody := eksConfigBlock.Body()

	cloudCredID := hclwrite.Tokens{
		{Type: hclsyntax.TokenIdent, Bytes: []byte(defaults.CloudCredential + "." + defaults.CloudCredential + ".id")},
	}

	eksConfigBlockBody.SetAttributeRaw(defaults.CloudCredentialID, cloudCredID)
	eksConfigBlockBody.SetAttributeValue(defaults.Region, cty.StringVal(terraformConfig.AWSConfig.Region))
	eksConfigBlockBody.SetAttributeValue(defaults.KubernetesVersion, cty.StringVal(terratestConfig.KubernetesVersion))
	awsSubnetsList := format.ListOfStrings(terraformConfig.AWSConfig.AWSSubnets)
	eksConfigBlockBody.SetAttributeRaw(amazon.Subnets, awsSubnetsList)
	awsSecGroupsList := format.ListOfStrings(terraformConfig.AWSConfig.AWSSecurityGroups)
	eksConfigBlockBody.SetAttributeRaw(amazon.SecurityGroups, awsSecGroupsList)
	eksConfigBlockBody.SetAttributeValue(amazon.PrivateAccess, cty.BoolVal(terraformConfig.AWSConfig.PrivateAccess))
	eksConfigBlockBody.SetAttributeValue(amazon.PublicAccess, cty.BoolVal(terraformConfig.AWSConfig.PublicAccess))

	for count, pool := range terratestConfig.Nodepools {
		poolNum := strconv.Itoa(count)

		_, err := resources.SetResourceNodepoolValidation(terraformConfig, pool, poolNum)
		if err != nil {
			return nil, nil, err
		}

		nodePoolsBlock := eksConfigBlockBody.AppendNewBlock(amazon.NodeGroups, nil)
		nodePoolsBlockBody := nodePoolsBlock.Body()

		nodePoolsBlockBody.SetAttributeValue(defaults.ResourceName, cty.StringVal(terraformConfig.ResourcePrefix+`-pool`+poolNum))
		nodePoolsBlockBody.SetAttributeValue(amazon.DiskSize, cty.NumberIntVal(pool.DiskSize))
		nodePoolsBlockBody.SetAttributeValue(amazon.InstanceType, cty.StringVal(pool.InstanceType))
		nodePoolsBlockBody.SetAttributeValue(amazon.DesiredSize, cty.NumberIntVal(pool.DesiredSize))
		nodePoolsBlockBody.SetAttributeValue(amazon.MaxSize, cty.NumberIntVal(pool.MaxSize))
		nodePoolsBlockBody.SetAttributeValue(amazon.MinSize, cty.NumberIntVal(pool.MinSize))
	}

	return newFile, file, nil
}
