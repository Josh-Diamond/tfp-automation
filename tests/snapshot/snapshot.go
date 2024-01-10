package snapshot

import (
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/rancher/shepherd/clients/rancher"
	management "github.com/rancher/shepherd/clients/rancher/generated/management/v3"
	"github.com/rancher/shepherd/extensions/clusters"
	"github.com/rancher/shepherd/extensions/clusters/kubernetesversions"
	extdefault "github.com/rancher/shepherd/extensions/defaults"
	"github.com/rancher/shepherd/extensions/etcdsnapshot"
	"github.com/rancher/shepherd/extensions/ingresses"
	nodestat "github.com/rancher/shepherd/extensions/nodes"
	"github.com/rancher/shepherd/extensions/provisioning"
	"github.com/rancher/shepherd/extensions/workloads"
	"github.com/rancher/shepherd/extensions/workloads/pods"
	"github.com/rancher/tfp-automation/config"
	set "github.com/rancher/tfp-automation/framework/set/provisioning"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	all                          = "all"
	containerImage               = "nginx"
	containerName                = "nginx"
	defaultNamespace             = "default"
	DeploymentSteveType          = "apps.deployment"
	isCattleLabeled              = true
	IngressSteveType             = "networking.k8s.io.ingress"
	ingressPath                  = "/index.html"
	initialIngressName           = "ingress-before-restore"
	initialWorkloadName          = "wload-before-restore"
	localClusterName             = "local"
	K3S                          = "k3s"
	kubernetesVersion            = "kubernetesVersion"
	namespace                    = "fleet-default"
	port                         = "port"
	ProvisioningSteveResouceType = "provisioning.cattle.io.cluster"
	RKE1                         = "rke1"
	RKE2                         = "rke2"
	serviceAppendName            = "service-"
	ServiceType                  = "service"
	WorkloadNamePostBackup       = "wload-after-backup"
)

func snapshotRestore(t *testing.T, client *rancher.Client, clusterName string, clusterConfig *config.TerratestConfig, terraformOptions *terraform.Options) {
	clusterID, err := clusters.GetClusterIDByName(client, clusterName)
	require.NoError(t, err)

	steveclient, err := client.Steve.ProxyDownstream(clusterID)
	require.NoError(t, err)

	localClusterID, err := clusters.GetClusterIDByName(client, localClusterName)
	require.NoError(t, err)

	var isRKE1 bool

	clusterObject, _, _ := clusters.GetProvisioningClusterByName(client, clusterName, namespace)
	if clusterObject == nil {
		_, err := client.Management.Cluster.ByID(clusterID)
		require.NoError(t, err)

		isRKE1 = true
	} else {
		isRKE1 = false
	}

	containerTemplate := workloads.NewContainer(containerName, containerImage, corev1.PullAlways, []corev1.VolumeMount{}, []corev1.EnvFromSource{}, nil, nil, nil)
	podTemplate := workloads.NewPodTemplate([]corev1.Container{containerTemplate}, []corev1.Volume{}, []corev1.LocalObjectReference{}, nil)
	deployment := workloads.NewDeploymentTemplate(initialWorkloadName, defaultNamespace, podTemplate, isCattleLabeled, nil)

	service := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceAppendName + initialWorkloadName,
			Namespace: defaultNamespace,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name: port,
					Port: 80,
				},
			},
			Selector: deployment.Spec.Template.Labels,
		},
	}

	deploymentResp, serviceResp, err := workloads.CreateDeploymentWithService(steveclient, initialWorkloadName, deployment, service)
	require.NoError(t, err)

	err = workloads.VerifyDeployment(steveclient, deploymentResp)
	require.NoError(t, err)
	require.Equal(t, initialWorkloadName, deploymentResp.ObjectMeta.Name)

	path := ingresses.NewIngressPathTemplate(networking.PathTypeExact, ingressPath, serviceAppendName+initialWorkloadName, 80)
	ingressTemplate := ingresses.NewIngressTemplate(initialIngressName, defaultNamespace, "", []networking.HTTPIngressPath{path})

	ingressResp, err := ingresses.CreateIngress(steveclient, initialIngressName, ingressTemplate)
	require.NoError(t, err)
	require.Equal(t, initialIngressName, ingressResp.ObjectMeta.Name)

	if isRKE1 {
		snapshotRestoreRKE1(t, client, podTemplate, deployment, clusterName, clusterID, localClusterID, clusterConfig, isRKE1, terraformOptions)
	} else {
		snapshotRestoreRKE2K3S(t, client, podTemplate, deployment, clusterName, clusterID, localClusterID, clusterConfig, isRKE1, terraformOptions)
	}

	logrus.Infof("Deleting created workloads...")
	err = steveclient.SteveType(DeploymentSteveType).Delete(deploymentResp)
	require.NoError(t, err)

	err = steveclient.SteveType(ServiceType).Delete(serviceResp)
	require.NoError(t, err)

	err = steveclient.SteveType(IngressSteveType).Delete(ingressResp)
	require.NoError(t, err)
}

func snapshotRestoreRKE1(t *testing.T, client *rancher.Client, podTemplate corev1.PodTemplateSpec, deployment *v1.Deployment, clusterName, clusterID, localClusterID string, clusterConfig *config.TerratestConfig, isRKE1 bool, terraformOptions *terraform.Options) {
	existingSnapshots, err := etcdsnapshot.GetRKE1Snapshots(client, clusterID)
	require.NoError(t, err)

	err = etcdsnapshot.CreateRKE1Snapshot(client, clusterName)
	require.NoError(t, err)

	clusterResp, err := client.Management.Cluster.ByID(clusterID)
	require.NoError(t, err)

	podErrors := pods.StatusPods(client, clusterID)
	assert.Empty(t, podErrors)

	initialKubernetesVersion := clusterResp.RancherKubernetesEngineConfig.Version
	require.Equal(t, initialKubernetesVersion, clusterResp.RancherKubernetesEngineConfig.Version)

	initialControlPlaneUnavailable := clusterResp.RancherKubernetesEngineConfig.UpgradeStrategy.MaxUnavailableControlplane
	require.Equal(t, initialControlPlaneUnavailable, clusterResp.RancherKubernetesEngineConfig.UpgradeStrategy.MaxUnavailableControlplane)

	initialWorkerUnavailableValue := clusterResp.RancherKubernetesEngineConfig.UpgradeStrategy.MaxUnavailableWorker
	require.Equal(t, initialWorkerUnavailableValue, clusterResp.RancherKubernetesEngineConfig.UpgradeStrategy.MaxUnavailableWorker)

	createPostBackupWorkloads(t, client, clusterID, podTemplate, deployment)

	etcdNodeCount, _ := etcdsnapshot.MatchNodeToAnyEtcdRole(client, clusterID)
	snapshotToRestore, err := provisioning.VerifySnapshots(client, localClusterID, clusterName, etcdNodeCount+len(existingSnapshots), isRKE1)
	require.NoError(t, err)

	if clusterConfig.SnapshotInput.SnapshotRestore == kubernetesVersion || clusterConfig.SnapshotInput.SnapshotRestore == all {
		clusterID, err := clusters.GetClusterIDByName(client, clusterName)
		require.NoError(t, err)

		clusterResp, err := client.Management.Cluster.ByID(clusterID)
		require.NoError(t, err)

		if clusterConfig.SnapshotInput.UpgradeKubernetesVersion == "" {
			defaultVersion, err := kubernetesversions.Default(client, clusters.RKE1ClusterType.String(), nil)
			clusterConfig.SnapshotInput.UpgradeKubernetesVersion = defaultVersion[0]
			require.NoError(t, err)
		}

		clusterResp.RancherKubernetesEngineConfig.Version = clusterConfig.SnapshotInput.UpgradeKubernetesVersion

		if clusterConfig.SnapshotInput.ControlPlaneUnavailableValue != "" && clusterConfig.SnapshotInput.WorkerUnavailableValue != "" {
			clusterResp.RancherKubernetesEngineConfig.UpgradeStrategy.MaxUnavailableControlplane = clusterConfig.SnapshotInput.ControlPlaneUnavailableValue
			clusterResp.RancherKubernetesEngineConfig.UpgradeStrategy.MaxUnavailableWorker = clusterConfig.SnapshotInput.WorkerUnavailableValue
		}

		clusterConfig.KubernetesVersion = clusterResp.RancherKubernetesEngineConfig.Version

		err = set.SetConfigTF(clusterConfig, clusterName)
		require.NoError(t, err)

		terraform.Apply(t, terraformOptions)

		logrus.Infof("Cluster version is upgraded to: %s", clusterResp.RancherKubernetesEngineConfig.Version)

		nodestat.AllManagementNodeReady(client, clusterResp.ID, extdefault.ThirtyMinuteTimeout)

		podErrors := pods.StatusPods(client, clusterID)
		assert.Empty(t, podErrors)
		require.Equal(t, clusterConfig.SnapshotInput.UpgradeKubernetesVersion, clusterResp.RancherKubernetesEngineConfig.Version)

		if clusterConfig.SnapshotInput.ControlPlaneUnavailableValue != "" && clusterConfig.SnapshotInput.WorkerUnavailableValue != "" {
			logrus.Infof("Control plane unavailable value is set to: %s", clusterResp.RancherKubernetesEngineConfig.UpgradeStrategy.MaxUnavailableControlplane)
			logrus.Infof("Worker unavailable value is set to: %s", clusterResp.RancherKubernetesEngineConfig.UpgradeStrategy.MaxUnavailableWorker)

			require.Equal(t, clusterConfig.SnapshotInput.ControlPlaneUnavailableValue, clusterResp.RancherKubernetesEngineConfig.UpgradeStrategy.MaxUnavailableControlplane)
			require.Equal(t, clusterConfig.SnapshotInput.WorkerUnavailableValue, clusterResp.RancherKubernetesEngineConfig.UpgradeStrategy.MaxUnavailableWorker)
		}
	}

	// Give the option to restore the same snapshot multiple times. By default, it is set to 1.
	for i := 0; i < clusterConfig.SnapshotInput.RecurringRestores; i++ {
		snapshotRKE1Restore := &management.RestoreFromEtcdBackupInput{
			EtcdBackupID:     snapshotToRestore,
			RestoreRkeConfig: clusterConfig.SnapshotInput.SnapshotRestore,
		}

		err = etcdsnapshot.RestoreRKE1Snapshot(client, clusterName, snapshotRKE1Restore)
		require.NoError(t, err)

		err = clusters.WaitClusterToBeUpgraded(client, clusterID)
		require.NoError(t, err)

		clusterResp, err = client.Management.Cluster.ByID(clusterID)
		require.NoError(t, err)

		logrus.Infof("Cluster version is restored to: %s", clusterResp.RancherKubernetesEngineConfig.Version)

		nodestat.AllManagementNodeReady(client, clusterResp.ID, extdefault.ThirtyMinuteTimeout)

		podErrors = pods.StatusPods(client, clusterID)
		assert.Empty(t, podErrors)
		require.Equal(t, initialKubernetesVersion, clusterResp.RancherKubernetesEngineConfig.Version)

		if clusterConfig.SnapshotInput.SnapshotRestore == kubernetesVersion || clusterConfig.SnapshotInput.SnapshotRestore == all {
			clusterResp, err = client.Management.Cluster.ByID(clusterID)
			require.NoError(t, err)
			require.Equal(t, initialKubernetesVersion, clusterResp.RancherKubernetesEngineConfig.Version)

			if clusterConfig.SnapshotInput.ControlPlaneUnavailableValue != "" && clusterConfig.SnapshotInput.WorkerUnavailableValue != "" {
				logrus.Infof("Control plane unavailable value is restored to: %s", clusterResp.RancherKubernetesEngineConfig.UpgradeStrategy.MaxUnavailableControlplane)
				logrus.Infof("Worker unavailable value is restored to: %s", clusterResp.RancherKubernetesEngineConfig.UpgradeStrategy.MaxUnavailableWorker)

				require.Equal(t, initialControlPlaneUnavailable, clusterResp.RancherKubernetesEngineConfig.UpgradeStrategy.MaxUnavailableControlplane)
				require.Equal(t, initialWorkerUnavailableValue, clusterResp.RancherKubernetesEngineConfig.UpgradeStrategy.MaxUnavailableWorker)
			}
		}
	}
}

func snapshotRestoreRKE2K3S(t *testing.T, client *rancher.Client, podTemplate corev1.PodTemplateSpec, deployment *v1.Deployment, clusterName, clusterID, localClusterID string, clusterConfig *config.TerratestConfig, isRKE1 bool, terraformOptions *terraform.Options) {
	existingSnapshots, err := etcdsnapshot.GetRKE2K3SSnapshots(client, localClusterID, clusterName)
	require.NoError(t, err)

	clusterConfig.SnapshotInput.CreateSnapshot = true

	err = set.SetConfigTF(clusterConfig, clusterName)
	require.NoError(t, err)

	terraform.Apply(t, terraformOptions)

	err = clusters.WaitClusterToBeUpgraded(client, clusterID)
	require.NoError(t, err)

	clusterObject, _, err := clusters.GetProvisioningClusterByName(client, clusterName, namespace)
	require.NoError(t, err)

	podErrors := pods.StatusPods(client, clusterID)
	assert.Empty(t, podErrors)

	initialKubernetesVersion := clusterObject.Spec.KubernetesVersion
	require.Equal(t, initialKubernetesVersion, clusterObject.Spec.KubernetesVersion)

	initialControlPlaneConcurrencyValue := clusterObject.Spec.RKEConfig.UpgradeStrategy.ControlPlaneConcurrency
	require.Equal(t, initialControlPlaneConcurrencyValue, clusterObject.Spec.RKEConfig.UpgradeStrategy.ControlPlaneConcurrency)

	initialWorkerConcurrencyValue := clusterObject.Spec.RKEConfig.UpgradeStrategy.WorkerConcurrency
	require.Equal(t, initialWorkerConcurrencyValue, clusterObject.Spec.RKEConfig.UpgradeStrategy.WorkerConcurrency)

	createPostBackupWorkloads(t, client, clusterID, podTemplate, deployment)

	etcdNodeCount, _ := etcdsnapshot.MatchNodeToAnyEtcdRole(client, clusterID)
	snapshotToRestore, err := provisioning.VerifySnapshots(client, localClusterID, clusterName, etcdNodeCount+len(existingSnapshots), isRKE1)
	require.NoError(t, err)

	if clusterConfig.SnapshotInput.SnapshotRestore == kubernetesVersion || clusterConfig.SnapshotInput.SnapshotRestore == all {
		clusterObject, _, err := clusters.GetProvisioningClusterByName(client, clusterName, namespace)
		require.NoError(t, err)

		initialKubernetesVersion := clusterObject.Spec.KubernetesVersion

		if clusterConfig.SnapshotInput.UpgradeKubernetesVersion == "" {
			if strings.Contains(initialKubernetesVersion, RKE2) {
				defaultVersion, err := kubernetesversions.Default(client, clusters.RKE2ClusterType.String(), nil)
				clusterConfig.SnapshotInput.UpgradeKubernetesVersion = defaultVersion[0]
				require.NoError(t, err)
			} else if strings.Contains(initialKubernetesVersion, K3S) {
				defaultVersion, err := kubernetesversions.Default(client, clusters.K3SClusterType.String(), nil)
				clusterConfig.SnapshotInput.UpgradeKubernetesVersion = defaultVersion[0]
				require.NoError(t, err)
			}
		}

		clusterObject.Spec.KubernetesVersion = clusterConfig.SnapshotInput.UpgradeKubernetesVersion

		if clusterConfig.SnapshotInput.ControlPlaneConcurrencyValue != "" && clusterConfig.SnapshotInput.WorkerConcurrencyValue != "" {
			clusterObject.Spec.RKEConfig.UpgradeStrategy.ControlPlaneConcurrency = clusterConfig.SnapshotInput.ControlPlaneConcurrencyValue
			clusterObject.Spec.RKEConfig.UpgradeStrategy.WorkerConcurrency = clusterConfig.SnapshotInput.WorkerConcurrencyValue
		}

		clusterConfig.KubernetesVersion = clusterObject.Spec.KubernetesVersion
		clusterConfig.SnapshotInput.CreateSnapshot = false

		err = set.SetConfigTF(clusterConfig, clusterName)
		require.NoError(t, err)

		terraform.Apply(t, terraformOptions)

		err = clusters.WaitClusterToBeUpgraded(client, clusterID)
		require.NoError(t, err)

		logrus.Infof("Cluster version is upgraded to: %s", clusterObject.Spec.KubernetesVersion)

		podErrors := pods.StatusPods(client, clusterID)
		assert.Empty(t, podErrors)
		require.Equal(t, clusterConfig.SnapshotInput.UpgradeKubernetesVersion, clusterObject.Spec.KubernetesVersion)

		if clusterConfig.SnapshotInput.ControlPlaneConcurrencyValue != "" && clusterConfig.SnapshotInput.WorkerConcurrencyValue != "" {
			logrus.Infof("Control plane concurrency value is set to: %s", clusterObject.Spec.RKEConfig.UpgradeStrategy.ControlPlaneConcurrency)
			logrus.Infof("Worker concurrency value is set to: %s", clusterObject.Spec.RKEConfig.UpgradeStrategy.WorkerConcurrency)

			require.Equal(t, clusterConfig.SnapshotInput.ControlPlaneConcurrencyValue, clusterObject.Spec.RKEConfig.UpgradeStrategy.ControlPlaneConcurrency)
			require.Equal(t, clusterConfig.SnapshotInput.WorkerConcurrencyValue, clusterObject.Spec.RKEConfig.UpgradeStrategy.WorkerConcurrency)
		}
	}

	// Give the option to restore the same snapshot multiple times. By default, it is set to 1.
	for i := 0; i < clusterConfig.SnapshotInput.RecurringRestores; i++ {
		clusterConfig.SnapshotInput.CreateSnapshot = false
		clusterConfig.SnapshotInput.RestoreSnapshot = true
		clusterConfig.SnapshotInput.SnapshotName = snapshotToRestore

		err = set.SetConfigTF(clusterConfig, clusterName)
		require.NoError(t, err)

		terraform.Apply(t, terraformOptions)

		err = clusters.WaitClusterToBeUpgraded(client, clusterID)
		require.NoError(t, err)

		clusterObject, _, err = clusters.GetProvisioningClusterByName(client, clusterName, namespace)
		require.NoError(t, err)

		logrus.Infof("Cluster version is restored to: %s", clusterObject.Spec.KubernetesVersion)

		podErrors = pods.StatusPods(client, clusterID)
		assert.Empty(t, podErrors)
		require.Equal(t, initialKubernetesVersion, clusterObject.Spec.KubernetesVersion)

		steveclient, err := client.Steve.ProxyDownstream(clusterID)
		require.NoError(t, err)

		deploymentList, err := steveclient.SteveType(workloads.DeploymentSteveType).NamespacedSteveClient(defaultNamespace).List(nil)
		require.NoError(t, err)
		require.Equal(t, 1, len(deploymentList.Data))
		require.Equal(t, initialWorkloadName, deploymentList.Data[0].ObjectMeta.Name)

		if clusterConfig.SnapshotInput.SnapshotRestore == kubernetesVersion || clusterConfig.SnapshotInput.SnapshotRestore == all {
			clusterObject, _, err := clusters.GetProvisioningClusterByName(client, clusterName, namespace)
			require.NoError(t, err)
			require.Equal(t, initialKubernetesVersion, clusterObject.Spec.KubernetesVersion)

			if clusterConfig.SnapshotInput.ControlPlaneConcurrencyValue != "" && clusterConfig.SnapshotInput.WorkerConcurrencyValue != "" {
				logrus.Infof("Control plane concurrency value is restored to: %s", clusterObject.Spec.RKEConfig.UpgradeStrategy.ControlPlaneConcurrency)
				logrus.Infof("Worker concurrency value is restored to: %s", clusterObject.Spec.RKEConfig.UpgradeStrategy.WorkerConcurrency)

				require.Equal(t, initialControlPlaneConcurrencyValue, clusterObject.Spec.RKEConfig.UpgradeStrategy.ControlPlaneConcurrency)
				require.Equal(t, initialWorkerConcurrencyValue, clusterObject.Spec.RKEConfig.UpgradeStrategy.WorkerConcurrency)
			}
		}
	}
}

func createPostBackupWorkloads(t *testing.T, client *rancher.Client, clusterID string, podTemplate corev1.PodTemplateSpec, deployment *v1.Deployment) {
	postBackupDeployment := workloads.NewDeploymentTemplate(WorkloadNamePostBackup, defaultNamespace, podTemplate, isCattleLabeled, nil)
	postBackupService := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceAppendName + WorkloadNamePostBackup,
			Namespace: defaultNamespace,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name: port,
					Port: 80,
				},
			},
			Selector: deployment.Spec.Template.Labels,
		},
	}

	steveclient, err := client.Steve.ProxyDownstream(clusterID)
	require.NoError(t, err)

	postDeploymentResp, _, err := workloads.CreateDeploymentWithService(steveclient, WorkloadNamePostBackup, postBackupDeployment, postBackupService)
	require.NoError(t, err)

	err = workloads.VerifyDeployment(steveclient, postDeploymentResp)
	require.NoError(t, err)
	require.Equal(t, WorkloadNamePostBackup, postDeploymentResp.ObjectMeta.Name)
}
