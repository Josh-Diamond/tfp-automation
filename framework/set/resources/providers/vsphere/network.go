package vsphere

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/rancher/tfp-automation/config"
	"github.com/rancher/tfp-automation/framework/set/defaults"
	"github.com/zclconf/go-cty/cty"
)

// CreateVsphereNetwork is a function that will set the vSphere network configuration in the main.tf file.
func CreateVsphereNetwork(rootBody *hclwrite.Body, terraformConfig *config.TerraformConfig, dataCenterValue hclwrite.Tokens) {
	networkBlock := rootBody.AppendNewBlock(defaults.Data, []string{defaults.VsphereNetwork, defaults.VsphereNetwork})
	networkBlockBody := networkBlock.Body()

	networkBlockBody.SetAttributeValue(defaults.ResourceName, cty.StringVal(terraformConfig.VsphereConfig.StandaloneNetwork))
	networkBlockBody.SetAttributeRaw(datacenterID, dataCenterValue)
}
