package framework

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	framework "github.com/rancher/shepherd/pkg/config"
	"github.com/rancher/tfp-automation/config"
	"github.com/rancher/tfp-automation/defaults/configs"
	set "github.com/rancher/tfp-automation/framework/set/provisioning"
	"github.com/stretchr/testify/require"
)

// Setup is a function that will set the Terraform configuration and return the Terraform options.
func Setup(t *testing.T) *terraform.Options {
	clusterConfig := new(config.TerratestConfig)
	framework.LoadConfig(configs.Terratest, clusterConfig)

	keyPath := set.SetKeyPath()

	err := set.SetConfigTF(clusterConfig, "", "")
	require.NoError(t, err)

	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: keyPath,
		NoColor:      true,
	})

	return terraformOptions
}
