//go:build (validation || infra.rke2k3s || cluster.nodedriver || extended) && !infra.any && !infra.aks && !infra.eks && !infra.gke && !infra.rke1 && !cluster.any && !cluster.custom && !sanity && !stress

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
	"github.com/rancher/tests/actions/machinepools"
	"github.com/rancher/tests/actions/provisioninginput"
	"github.com/rancher/tests/actions/scalinginput"
	resources "github.com/rancher/tests/validation/provisioning/resources/provisioncluster"
	standard "github.com/rancher/tests/validation/provisioning/resources/standarduser"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	provisioningSteveResourceType = "provisioning.cattle.io.cluster"
)

type NodeReplacingTestSuite struct {
	suite.Suite
	client             *rancher.Client
	standardUserClient *rancher.Client
	session            *session.Session
	cattleConfig       map[string]any
	rke2ClusterConfig  *clusters.ClusterConfig
	k3sClusterConfig   *clusters.ClusterConfig
	rke2ClusterID      string
	k3sClusterID       string
}

func (s *NodeReplacingTestSuite) TearDownSuite() {
	s.session.Cleanup()
}

func (s *NodeReplacingTestSuite) SetupSuite() {
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

	s.rke2ClusterID, err = resources.ProvisionRKE2K3SCluster(s.T(), s.standardUserClient, extClusters.RKE2ClusterType.String(), s.rke2ClusterConfig, true, false)
	require.NoError(s.T(), err)

	s.k3sClusterID, err = resources.ProvisionRKE2K3SCluster(s.T(), s.standardUserClient, extClusters.K3SClusterType.String(), s.k3sClusterConfig, true, false)
	require.NoError(s.T(), err)
}

func (s *NodeReplacingTestSuite) TestReplacingNodes() {
	nodeRolesEtcd := machinepools.NodeRoles{
		Etcd: true,
	}

	nodeRolesControlPlane := machinepools.NodeRoles{
		ControlPlane: true,
	}

	nodeRolesWorker := machinepools.NodeRoles{
		Worker: true,
	}

	tests := []struct {
		name      string
		nodeRoles machinepools.NodeRoles
		clusterID string
	}{
		{"Replacing RKE2 control plane nodes", nodeRolesControlPlane, s.rke2ClusterID},
		{"Replacing RKE2 etcd nodes", nodeRolesEtcd, s.rke2ClusterID},
		{"Replacing RKE2 worker nodes", nodeRolesWorker, s.rke2ClusterID},
		{"Replacing K3S control plane nodes", nodeRolesControlPlane, s.k3sClusterID},
		{"Replacing K3S etcd nodes", nodeRolesEtcd, s.k3sClusterID},
		{"Replacing K3S worker nodes", nodeRolesWorker, s.k3sClusterID},
	}

	for _, tt := range tests {
		cluster, err := s.client.Steve.SteveType(provisioningSteveResourceType).ByID(tt.clusterID)
		require.NoError(s.T(), err)

		s.Run(tt.name, func() {
			err := scalinginput.ReplaceNodes(s.client, cluster.Name, tt.nodeRoles.Etcd, tt.nodeRoles.ControlPlane, tt.nodeRoles.Worker)
			require.NoError(s.T(), err)
		})
	}
}

func TestNodeReplacingTestSuite(t *testing.T) {
	suite.Run(t, new(NodeReplacingTestSuite))
}
