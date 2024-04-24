package provisioning

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/rancher/tfp-automation/config"
	"github.com/zclconf/go-cty/cty"
)

const (
	EtcdSnapshotCreate  = "etcd_snapshot_create"
	EtcdSnapshotRestore = "etcd_snapshot_restore"

	Generation       = "generation"
	RestoreRKEConfig = "restore_rke_config"
)

// setCreateRKE2K3SSnapshot is a function that will set the etcd_snapshot_create resource
// block in the main.tf file for a RKE2/K3S cluster.
func setCreateRKE2K3SSnapshot(terraformConfig *config.TerraformConfig, rkeConfigBlockBody *hclwrite.Body) {
	createSnapshotBlock := rkeConfigBlockBody.AppendNewBlock(EtcdSnapshotCreate, nil)
	createSnapshotBlockBody := createSnapshotBlock.Body()

	generation := int64(1)

	if createSnapshotBlockBody.GetAttribute(Generation) == nil {
		createSnapshotBlockBody.SetAttributeValue(Generation, cty.NumberIntVal(generation))
	} else {
		createSnapshotBlockBody.SetAttributeValue(Generation, cty.NumberIntVal(generation+1))
	}
}

// setRestoreRKE2K3SSnapshot is a function that will set the etcd_snapshot_restore
// resource block in the main.tf file for a RKE2/K3S cluster.
func setRestoreRKE2K3SSnapshot(terraformConfig *config.TerraformConfig, rkeConfigBlockBody *hclwrite.Body, snapshots config.Snapshots) {
	restoreSnapshotBlock := rkeConfigBlockBody.AppendNewBlock(EtcdSnapshotRestore, nil)
	restoreSnapshotBlockBody := restoreSnapshotBlock.Body()

	generation := int64(1)

	if restoreSnapshotBlockBody.GetAttribute(Generation) == nil {
		restoreSnapshotBlockBody.SetAttributeValue(Generation, cty.NumberIntVal(generation))
	} else {
		restoreSnapshotBlockBody.SetAttributeValue(Generation, cty.NumberIntVal(generation+1))
	}

	restoreSnapshotBlockBody.SetAttributeValue((resourceName), cty.StringVal(snapshots.SnapshotName))
	restoreSnapshotBlockBody.SetAttributeValue((RestoreRKEConfig), cty.StringVal(snapshots.SnapshotRestore))
}
