//go:build (validation || extended || infra.any || cluster.any) && !sanity && !stress

package rke1

import (
	"testing"

	"github.com/rancher/shepherd/clients/rancher"
	"github.com/rancher/shepherd/pkg/config"
	"github.com/rancher/shepherd/pkg/session"
	"github.com/rancher/tests/actions/etcdsnapshot"
	"github.com/rancher/tests/actions/provisioninginput"
	resources "github.com/rancher/tests/validation/provisioning/resources/provisioncluster"
	standard "github.com/rancher/tests/validation/provisioning/resources/standarduser"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type RKE1S3SnapshotRestoreTestSuite struct {
	suite.Suite
	session            *session.Session
	client             *rancher.Client
	standardUserClient *rancher.Client
	provisioningConfig *provisioninginput.Config
	rke1ClusterID      string
}

func (s *RKE1S3SnapshotRestoreTestSuite) TearDownSuite() {
	s.session.Cleanup()
}

func (s *RKE1S3SnapshotRestoreTestSuite) SetupSuite() {
	testSession := session.NewSession()
	s.session = testSession

	s.provisioningConfig = new(provisioninginput.Config)
	config.LoadConfig(provisioninginput.ConfigurationFileKey, s.provisioningConfig)

	client, err := rancher.NewClient("", testSession)
	require.NoError(s.T(), err)

	s.client = client

	s.standardUserClient, err = standard.CreateStandardUser(s.client)
	require.NoError(s.T(), err)

	nodeRolesStandard := []provisioninginput.NodePools{
		provisioninginput.EtcdNodePool,
		provisioninginput.ControlPlaneNodePool,
		provisioninginput.WorkerNodePool,
	}

	nodeRolesStandard[0].NodeRoles.Quantity = 3
	nodeRolesStandard[1].NodeRoles.Quantity = 2
	nodeRolesStandard[2].NodeRoles.Quantity = 3

	s.provisioningConfig.NodePools = nodeRolesStandard

	s.rke1ClusterID, err = resources.ProvisionRKE1Cluster(s.T(), s.standardUserClient, s.provisioningConfig, true, false)
	require.NoError(s.T(), err)
}

func (s *RKE1S3SnapshotRestoreTestSuite) TestRKE1S3SnapshotRestore() {
	snapshotRestoreNone := &etcdsnapshot.Config{
		UpgradeKubernetesVersion: "",
		SnapshotRestore:          "none",
		RecurringRestores:        1,
	}

	tests := []struct {
		name         string
		etcdSnapshot *etcdsnapshot.Config
		clusterID    string
	}{
		{"RKE1 S3 Restore", snapshotRestoreNone, s.rke1ClusterID},
	}

	for _, tt := range tests {
		cluster, err := s.client.Management.Cluster.ByID(tt.clusterID)
		require.NoError(s.T(), err)

		s.Run(tt.name, func() {
			err := etcdsnapshot.CreateAndValidateSnapshotRestore(s.client, cluster.Name, tt.etcdSnapshot, containerImage)
			require.NoError(s.T(), err)
		})
	}
}

func TestRKE1S3SnapshotRestoreTestSuite(t *testing.T) {
	suite.Run(t, new(RKE1S3SnapshotRestoreTestSuite))
}
