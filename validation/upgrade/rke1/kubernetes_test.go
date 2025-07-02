//go:build validation

package rke1

import (
	"testing"

	"github.com/rancher/shepherd/clients/rancher"
	extClusters "github.com/rancher/shepherd/extensions/clusters"
	"github.com/rancher/shepherd/extensions/clusters/kubernetesversions"
	"github.com/rancher/shepherd/pkg/config"
	"github.com/rancher/shepherd/pkg/session"
	"github.com/rancher/tests/actions/clusters"
	"github.com/rancher/tests/actions/provisioninginput"
	resources "github.com/rancher/tests/validation/provisioning/resources/provisioncluster"
	standard "github.com/rancher/tests/validation/provisioning/resources/standarduser"
	"github.com/rancher/tests/validation/upgrade"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type UpgradeRKE1KubernetesTestSuite struct {
	suite.Suite
	session            *session.Session
	client             *rancher.Client
	standardUserClient *rancher.Client
	provisioningConfig *provisioninginput.Config
	rke1ClusterID      string
}

func (u *UpgradeRKE1KubernetesTestSuite) TearDownSuite() {
	u.session.Cleanup()
}

func (u *UpgradeRKE1KubernetesTestSuite) SetupSuite() {
	testSession := session.NewSession()
	u.session = testSession

	u.provisioningConfig = new(provisioninginput.Config)
	config.LoadConfig(provisioninginput.ConfigurationFileKey, u.provisioningConfig)

	client, err := rancher.NewClient("", testSession)
	require.NoError(u.T(), err)

	u.client = client

	u.standardUserClient, err = standard.CreateStandardUser(u.client)
	require.NoError(u.T(), err)

	nodeRolesStandard := []provisioninginput.NodePools{
		provisioninginput.EtcdNodePool,
		provisioninginput.ControlPlaneNodePool,
		provisioninginput.WorkerNodePool,
	}

	nodeRolesStandard[0].NodeRoles.Quantity = 3
	nodeRolesStandard[1].NodeRoles.Quantity = 2
	nodeRolesStandard[2].NodeRoles.Quantity = 3

	u.provisioningConfig.NodePools = nodeRolesStandard

	u.rke1ClusterID, err = resources.ProvisionRKE1Cluster(u.T(), u.standardUserClient, u.provisioningConfig, false, false)
	require.NoError(u.T(), err)
}

func (u *UpgradeRKE1KubernetesTestSuite) TestUpgradeRKE1Kubernetes() {
	tests := []struct {
		name        string
		client      *rancher.Client
		clusterType string
	}{
		{"Upgrading ", u.client, extClusters.RKE1ClusterType.String()},
	}

	for _, tt := range tests {
		version, err := kubernetesversions.Default(u.client, tt.clusterType, nil)
		require.NoError(u.T(), err)

		clusterResp, err := u.client.Management.Cluster.ByID(u.rke1ClusterID)
		require.NoError(u.T(), err)

		testConfig := clusters.ConvertConfigToClusterConfig(u.provisioningConfig)
		testConfig.KubernetesVersion = version[0]

		tt.name += "RKE1 cluster from " + clusterResp.RancherKubernetesEngineConfig.Version + " to " + testConfig.KubernetesVersion

		u.Run(tt.name, func() {
			upgrade.DownstreamCluster(&u.Suite, tt.name, u.client, clusterResp.Name, testConfig, u.rke1ClusterID, testConfig.KubernetesVersion, true)
		})
	}
}

func TestUpgradeRKE1KubernetesTestSuite(t *testing.T) {
	suite.Run(t, new(UpgradeRKE1KubernetesTestSuite))
}
