package storagecluster

import (
	"fmt"
	"strconv"
	"strings"

	ocsv1 "github.com/red-hat-storage/ocs-operator/api/v1"
	"github.com/red-hat-storage/ocs-operator/controllers/util"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
)

func generateNameForCephCluster(initData *ocsv1.StorageCluster) string {
	return generateNameForCephClusterFromString(initData.Name)
}

func generateNameForCephClusterFromString(name string) string {
	return fmt.Sprintf("%s-cephcluster", name)
}

func generateNameForCephFilesystem(initData *ocsv1.StorageCluster) string {
	return fmt.Sprintf("%s-cephfilesystem", initData.Name)
}

func generateNameForCephNFS(initData *ocsv1.StorageCluster) string {
	return fmt.Sprintf("%s-cephnfs", initData.Name)
}

func generateNameForNFSService(initData *ocsv1.StorageCluster) string {
	return fmt.Sprintf("%s-service", generateNameForCephNFS(initData))
}

func generateNameForCephNFSBlockPool(initData *ocsv1.StorageCluster) string {
	return fmt.Sprintf("%s-builtin-pool", generateNameForCephNFS(initData))
}

func generateNameForCephObjectStoreUser(initData *ocsv1.StorageCluster) string {
	return fmt.Sprintf("%s-cephobjectstoreuser", initData.Name)
}

func generateNameForCephBlockPool(initData *ocsv1.StorageCluster) string {
	return fmt.Sprintf("%s-cephblockpool", initData.Name)
}

func generateNameForCephObjectStore(initData *ocsv1.StorageCluster) string {
	return fmt.Sprintf("%s-%s", initData.Name, "cephobjectstore")
}

func generateNameForCephRgwSC(initData *ocsv1.StorageCluster) string {
	return fmt.Sprintf("%s-ceph-rgw", initData.Name)
}

func generateNameForCephFilesystemSC(initData *ocsv1.StorageCluster) string {
	return fmt.Sprintf("%s-cephfs", initData.Name)
}

func generateNameForCephBlockPoolSC(initData *ocsv1.StorageCluster) string {
	return fmt.Sprintf("%s-ceph-rbd", initData.Name)
}

func generateNameForEncryptedCephBlockPoolSC(initData *ocsv1.StorageCluster) string {
	return fmt.Sprintf("%s-ceph-rbd-encrypted", initData.Name)
}

func generateNameForCephNetworkFilesystemSC(initData *ocsv1.StorageCluster) string {
	return fmt.Sprintf("%s-ceph-nfs", initData.Name)
}

func generateNameForNFSCSIProvisioner(initData *ocsv1.StorageCluster) string {
	return fmt.Sprintf("%s.nfs.csi.ceph.com", initData.Namespace)
}

// generateNameForSnapshotClass function generates 'SnapshotClass' name.
// 'snapshotType' can be: 'rbdSnapshotter' or 'cephfsSnapshotter'
func generateNameForSnapshotClass(initData *ocsv1.StorageCluster, snapshotType SnapshotterType) string {
	return fmt.Sprintf("%s-%splugin-snapclass", initData.Name, snapshotType)
}

func generateNameForSnapshotClassDriver(initData *ocsv1.StorageCluster, snapshotType SnapshotterType) string {
	return fmt.Sprintf("%s.%s.csi.ceph.com", initData.Namespace, snapshotType)
}

func generateNameForSnapshotClassSecret(instance *ocsv1.StorageCluster, snapshotType SnapshotterType) string {
	if instance.Spec.ExternalStorage.Enable {
		data, ok := externalOCSResources[instance.UID]
		if !ok {
			log.Error(fmt.Errorf("Unable to retrieve external resource from externalOCSResources"),
				"unable to generate name for snapshot class secret for external mode")
		}
		// print the Secret name which contains the prefix as the rook-csi-rbd/cephfs-provisioner default secret name
		// for example if the secret name is rook-csi-rbd-node-rookStorage-replicapool it will check the prefix with rook-csi-rbd-node if it matches it will return that name
		for _, d := range data {
			if d.Kind == "Secret" {
				if strings.Contains(d.Name, fmt.Sprintf("rook-csi-%s-provisioner", snapshotType)) {
					return d.Name
				}
			}
		}
	}
	return fmt.Sprintf("rook-csi-%s-provisioner", snapshotType)
}

func generateNameForCephRbdMirror(initData *ocsv1.StorageCluster) string {
	return fmt.Sprintf("%s-cephrbdmirror", initData.Name)
}

// generateCephReplicatedSpec returns the ReplicatedSpec for the cephCluster
// based on the StorageCluster configuration
func generateCephReplicatedSpec(initData *ocsv1.StorageCluster, poolType string) cephv1.ReplicatedSpec {
	crs := cephv1.ReplicatedSpec{}

	crs.Size = getCephPoolReplicatedSize(initData)
	crs.ReplicasPerFailureDomain = uint(getReplicasPerFailureDomain(initData))
	if "data" == poolType {
		crs.TargetSizeRatio = .49
	}

	return crs
}

// generateStorageQuotaName function generates a name for ClusterResourceQuota
func generateStorageQuotaName(storageClassName, quotaName string) string {
	return fmt.Sprintf("%s-%s", storageClassName, quotaName)
}

// GenerateCephFSProviderParameters function generates extra parameters required for provider storage clusters
func GenerateCephFSProviderParameters(initData *ocsv1.StorageCluster) (map[string]string, error) {
	deviceSetList := initData.Spec.StorageDeviceSets
	var deviceSet *ocsv1.StorageDeviceSet
	for i := range deviceSetList {
		ds := &deviceSetList[i]
		if ds.Name == "default" {
			deviceSet = ds
			break
		}
	}
	if deviceSet != nil {
		deviceCount := deviceSet.Count
		pgUnitSize := util.GetPGBaseUnitSize(deviceCount)
		pgNumValue := pgUnitSize * 4
		providerParameters := map[string]string{
			"pg_autoscale_mode": "off",
			"pg_num":            strconv.Itoa(pgNumValue),
			"pgp_num":           strconv.Itoa(pgNumValue),
		}
		return providerParameters, nil
	}
	return nil, fmt.Errorf("Could not find  device set named default in Storage cluster")

}
