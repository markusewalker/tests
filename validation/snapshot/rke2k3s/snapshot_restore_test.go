//go:build (validation || extended || infra.any || cluster.any) && !sanity && !stress

package rke2k3s

import (
	"os"
	"testing"

	"github.com/rancher/shepherd/clients/rancher"
	extClusters "github.com/rancher/shepherd/extensions/clusters"
	"github.com/rancher/shepherd/pkg/config"
	"github.com/rancher/shepherd/pkg/config/operations"
	"github.com/rancher/shepherd/pkg/session"
	"github.com/rancher/tests/actions/clusters"
	"github.com/rancher/tests/actions/config/defaults"
	"github.com/rancher/tests/actions/etcdsnapshot"
	"github.com/rancher/tests/actions/provisioninginput"
	resources "github.com/rancher/tests/validation/provisioning/resources/provisioncluster"
	standard "github.com/rancher/tests/validation/provisioning/resources/standarduser"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	containerImage                = "nginx"
	namespace                     = "fleet-default"
	provisioningSteveResourceType = "provisioning.cattle.io.cluster"
	windowsContainerImage         = "mcr.microsoft.com/windows/servercore/iis"
)

type SnapshotRestoreTestSuite struct {
	suite.Suite
	session            *session.Session
	client             *rancher.Client
	standardUserClient *rancher.Client
	cattleConfig       map[string]any
	rke2ClusterConfig  *clusters.ClusterConfig
	k3sClusterConfig   *clusters.ClusterConfig
	rke2ClusterID      string
	k3sClusterID       string
}

func (s *SnapshotRestoreTestSuite) TearDownSuite() {
	s.session.Cleanup()
}

func (s *SnapshotRestoreTestSuite) SetupSuite() {
	testSession := session.NewSession()
	s.session = testSession

	client, err := rancher.NewClient("", s.session)
	require.NoError(s.T(), err)

	s.client = client

	s.standardUserClient, err = standard.CreateStandardUser(s.client)
	require.NoError(s.T(), err)

	s.cattleConfig = config.LoadConfigFromFile(os.Getenv(config.ConfigEnvironmentKey))

	s.rke2ClusterConfig = new(clusters.ClusterConfig)
	operations.LoadObjectFromMap(defaults.ClusterConfigKey, s.cattleConfig, s.rke2ClusterConfig)

	s.k3sClusterConfig = new(clusters.ClusterConfig)
	operations.LoadObjectFromMap(defaults.ClusterConfigKey, s.cattleConfig, s.k3sClusterConfig)

	nodeRolesStandard := []provisioninginput.MachinePools{
		provisioninginput.EtcdMachinePool,
		provisioninginput.ControlPlaneMachinePool,
		provisioninginput.WorkerMachinePool,
	}

	nodeRolesStandard[0].MachinePoolConfig.Quantity = 3
	nodeRolesStandard[1].MachinePoolConfig.Quantity = 2
	nodeRolesStandard[2].MachinePoolConfig.Quantity = 3

	s.rke2ClusterConfig.MachinePools = nodeRolesStandard
	s.k3sClusterConfig.MachinePools = nodeRolesStandard

	s.rke2ClusterID, err = resources.ProvisionRKE2K3SCluster(s.T(), s.standardUserClient, extClusters.RKE2ClusterType.String(), s.rke2ClusterConfig, false, false)
	require.NoError(s.T(), err)

	s.k3sClusterID, err = resources.ProvisionRKE2K3SCluster(s.T(), s.standardUserClient, extClusters.K3SClusterType.String(), s.k3sClusterConfig, false, false)
	require.NoError(s.T(), err)
}

func (s *SnapshotRestoreTestSuite) TestSnapshotRestore() {
	snapshotRestoreNone := &etcdsnapshot.Config{
		UpgradeKubernetesVersion: "",
		SnapshotRestore:          "none",
		RecurringRestores:        1,
	}

	snapshotRestoreK8sVersion := &etcdsnapshot.Config{
		UpgradeKubernetesVersion: "",
		SnapshotRestore:          "kubernetesVersion",
		RecurringRestores:        1,
	}

	snapshotRestoreAll := &etcdsnapshot.Config{
		UpgradeKubernetesVersion:     "",
		SnapshotRestore:              "all",
		ControlPlaneConcurrencyValue: "15%",
		WorkerConcurrencyValue:       "20%",
		RecurringRestores:            1,
	}

	tests := []struct {
		name         string
		etcdSnapshot *etcdsnapshot.Config
		clusterID    string
	}{
		{"RKE2 restore etcd only", snapshotRestoreNone, s.rke2ClusterID},
		{"RKE2 restore Kubernetes version and etcd", snapshotRestoreK8sVersion, s.rke2ClusterID},
		{"RKE2 restore cluster config, Kubernetes version and etcd", snapshotRestoreAll, s.rke2ClusterID},
		{"K3S restore etcd only", snapshotRestoreNone, s.k3sClusterID},
		{"K3S restore Kubernetes version and etcd", snapshotRestoreK8sVersion, s.k3sClusterID},
		{"K3S restore cluster config, Kubernetes version and etcd", snapshotRestoreAll, s.k3sClusterID},
	}

	for _, tt := range tests {
		cluster, err := s.client.Steve.SteveType(provisioningSteveResourceType).ByID(tt.clusterID)
		require.NoError(s.T(), err)

		s.Run(tt.name, func() {
			err := etcdsnapshot.CreateAndValidateSnapshotRestore(s.client, cluster.Name, tt.etcdSnapshot, containerImage)
			require.NoError(s.T(), err)
		})
	}
}

func TestSnapshotRestoreTestSuite(t *testing.T) {
	suite.Run(t, new(SnapshotRestoreTestSuite))
}
