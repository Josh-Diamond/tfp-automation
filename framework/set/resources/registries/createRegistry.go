package registries

import (
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/rancher/tfp-automation/config"
	"github.com/rancher/tfp-automation/defaults/keypath"
	"github.com/rancher/tfp-automation/framework/set/defaults"
	"github.com/rancher/tfp-automation/framework/set/resources/rancher2"
	"github.com/rancher/tfp-automation/framework/set/resources/rke2"
	"github.com/sirupsen/logrus"
	"github.com/zclconf/go-cty/cty"
)

// CreateAuthenticatedRegistry is a helper function that will create an authenticated registry.
func CreateAuthenticatedRegistry(file *os.File, newFile *hclwrite.File, rootBody *hclwrite.Body, terraformConfig *config.TerraformConfig,
	rke2AuthRegistryPublicDNS string) (*os.File, error) {
	userDir, _ := rancher2.SetKeyPath(keypath.RegistryKeyPath, terraformConfig.Provider)

	registryScriptPath := filepath.Join(userDir, "src/github.com/rancher/tfp-automation/framework/set/resources/registries/auth-registry.sh")

	registryScriptContent, err := os.ReadFile(registryScriptPath)
	if err != nil {
		return nil, err
	}

	_, provisionerBlockBody := rke2.SSHNullResource(rootBody, terraformConfig, rke2AuthRegistryPublicDNS, authRegistry)

	command := "bash -c '/tmp/auth-registry.sh " + terraformConfig.StandaloneRegistry.RegistryUsername + " " +
		terraformConfig.StandaloneRegistry.RegistryPassword + " " + terraformConfig.StandaloneRegistry.RegistryName + " " +
		rke2AuthRegistryPublicDNS + " " + terraformConfig.Standalone.RancherTagVersion + " " +
		terraformConfig.StandaloneRegistry.AssetsPath + " " + terraformConfig.Standalone.OSUser + " " +
		terraformConfig.Standalone.RancherImage

	if terraformConfig.Standalone.RancherAgentImage != "" {
		command += " " + terraformConfig.Standalone.RancherAgentImage
	}

	command += " || true'"

	provisionerBlockBody.SetAttributeValue(defaults.Inline, cty.ListVal([]cty.Value{
		cty.StringVal("echo '" + string(registryScriptContent) + "' > /tmp/auth-registry.sh"),
		cty.StringVal("chmod +x /tmp/auth-registry.sh"),
		cty.StringVal(command),
	}))

	_, err = file.Write(newFile.Bytes())
	if err != nil {
		logrus.Infof("Failed to append configurations to main.tf file. Error: %v", err)
		return nil, err
	}

	return file, nil
}

// CreateNonAuthenticatedRegistry is a helper function that will create a non-authenticated registry.
func CreateNonAuthenticatedRegistry(file *os.File, newFile *hclwrite.File, rootBody *hclwrite.Body, terraformConfig *config.TerraformConfig,
	rke2NonAuthRegistryPublicDNS, registryType string) (*os.File, error) {
	userDir, _ := rancher2.SetKeyPath(keypath.RegistryKeyPath, terraformConfig.Provider)

	registryScriptPath := filepath.Join(userDir, "src/github.com/rancher/tfp-automation/framework/set/resources/registries/non-auth-registry.sh")

	registryScriptContent, err := os.ReadFile(registryScriptPath)
	if err != nil {
		return nil, err
	}

	_, provisionerBlockBody := rke2.SSHNullResource(rootBody, terraformConfig, rke2NonAuthRegistryPublicDNS, registryType)

	command := "bash -c '/tmp/non-auth-registry.sh " + terraformConfig.StandaloneRegistry.RegistryName + " " +
		rke2NonAuthRegistryPublicDNS + " " + terraformConfig.Standalone.RancherTagVersion + " " +
		terraformConfig.StandaloneRegistry.AssetsPath + " " + terraformConfig.Standalone.OSUser + " " +
		terraformConfig.Standalone.RancherImage

	if terraformConfig.Standalone.RancherAgentImage != "" {
		command += " " + terraformConfig.Standalone.RancherAgentImage
	}

	command += " || true'"

	provisionerBlockBody.SetAttributeValue(defaults.Inline, cty.ListVal([]cty.Value{
		cty.StringVal("echo '" + string(registryScriptContent) + "' > /tmp/non-auth-registry.sh"),
		cty.StringVal("chmod +x /tmp/non-auth-registry.sh"),
		cty.StringVal(command),
	}))

	_, err = file.Write(newFile.Bytes())
	if err != nil {
		logrus.Infof("Failed to append configurations to main.tf file. Error: %v", err)
		return nil, err
	}

	return file, nil
}
