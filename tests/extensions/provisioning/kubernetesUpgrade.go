package provisioning

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/rancher/shepherd/clients/rancher"
	"github.com/rancher/tfp-automation/config"
	framework "github.com/rancher/tfp-automation/framework/set"
	"github.com/stretchr/testify/require"
)

// KubernetesUpgrade is a function that will run terraform apply and uprade the
// Kubernetes version of the provisioned cluster.
func KubernetesUpgrade(t *testing.T, client *rancher.Client, rancherConfig *rancher.Config, terraformConfig *config.TerraformConfig,
	clusterConfig *config.TerratestConfig, testUser, testPassword, clusterName, poolName string, terraformOptions *terraform.Options) {
	DefaultUpgradedK8sVersion(t, client, clusterConfig, terraformConfig)

	err := framework.ConfigTF(nil, rancherConfig, terraformConfig, clusterConfig, testUser, testPassword, clusterName, poolName, "")
	require.NoError(t, err)

	terraform.Apply(t, terraformOptions)
}
