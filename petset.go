package controllers

import (
	"fmt"

	api "kubedb.dev/apimachinery/apis/kubedb/v1alpha2"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	kutil "kmodules.xyz/client-go"
	clientutil "kmodules.xyz/client-go/client"
	coreutil "kmodules.xyz/client-go/core/v1"
	"kmodules.xyz/go-containerregistry/authn"
	ofst "kmodules.xyz/offshoot-api/api/v1"
	psapi "kubeops.dev/petset/apis/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (c *ClickHouseReconciler) ensurePetSet() error {
	volumes := c.getVolumes()
	pvc := c.getPVC()
	dbVolumeMounts := c.getDBVolumeMounts()
	initVolumeMounts := c.getInitVolumeMounts()
	podTemplate := c.db.Spec.PodTemplate
	dbContainerPorts := []core.ContainerPort{
		{
			Name:          "http",
			ContainerPort: 8123,
		},
		{
			Name:          "tcp",
			ContainerPort: 9000,
		},
		{
			Name:          "https",
			ContainerPort: 8443,
		},
		{
			Name:          "tls",
			ContainerPort: 9440,
		},
		{
			Name:          "prom",
			ContainerPort: 9363,
		},
	}
	image, err := authn.ImageWithDigest(c.Client, c.version.Spec.DB.Image, K8sChainOpts(c.db))
	if err != nil {
		c.Log.Error(err, "Failed to get image with digest")
		return nil
	}

	initImage, err := authn.ImageWithDigest(c.Client, c.version.Spec.InitContainer.Image, K8sChainOpts(c.db))
	if err != nil {
		c.Log.Error(err, "Failed to get image with digest")
		return nil
	}

	var envList []core.EnvVar

	envList = coreutil.UpsertEnvVars(envList, []core.EnvVar{
		{
			Name: "CLICKHOUSE_POD_NAME",
			ValueFrom: &core.EnvVarSource{
				FieldRef: &core.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
		{
			Name: "CLICKHOUSE_POD_NAMESPACE",
			ValueFrom: &core.EnvVarSource{
				FieldRef: &core.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
		{
			Name:  "CLICKHOUSE_SERVICE",
			Value: c.db.GoverningServiceName(),
		},
	}...)
	if !c.db.Spec.DisableSecurity {
		envList = coreutil.UpsertEnvVars(envList, []core.EnvVar{
			{
				Name: "CLICKHOUSE_USER",
				ValueFrom: &core.EnvVarSource{
					SecretKeyRef: &core.SecretKeySelector{
						LocalObjectReference: core.LocalObjectReference{
							Name: c.db.DefaultUserCredSecretName("admin"),
						},
						Key: core.BasicAuthUsernameKey,
					},
				},
			},
			{
				Name: "CLICKHOUSE_PASSWORD",
				ValueFrom: &core.EnvVarSource{
					SecretKeyRef: &core.SecretKeySelector{
						LocalObjectReference: core.LocalObjectReference{
							Name: c.db.DefaultUserCredSecretName("admin"),
						},
						Key: core.BasicAuthPasswordKey,
					},
				},
			},
			{
				Name:  "CLICKHOUSE_DEFAULT_ACCESS_MANAGEMENT",
				Value: "1",
			},
		}...)
	}

	dbContainer := core.Container{
		Name:         api.ClickHouseContainerName,
		Image:        image,
		Ports:        dbContainerPorts,
		Env:          envList,
		VolumeMounts: dbVolumeMounts,
	}

	containerTemplate := coreutil.GetContainerByName(c.db.Spec.PodTemplate.Spec.Containers, api.ClickHouseContainerName)
	if containerTemplate != nil {
		dbContainer = coreutil.MergeContainer(dbContainer, *containerTemplate)
	}
	containers := coreutil.UpsertContainer(c.db.Spec.PodTemplate.Spec.Containers, dbContainer)

	initContainer := core.Container{
		Name:         api.ClickHouseInitContainerName,
		Image:        initImage,
		Env:          envList,
		VolumeMounts: initVolumeMounts,
	}

	initContainerTemplate := coreutil.GetContainerByName(c.db.Spec.PodTemplate.Spec.InitContainers, api.ClickHouseInitContainerName)
	if initContainerTemplate != nil {
		initContainer = coreutil.MergeContainer(initContainer, *initContainerTemplate)
	}

	// Create of Patch the petset with given opts
	ps := &psapi.PetSet{
		ObjectMeta: meta.ObjectMeta{
			Name:      c.db.PetSetName(),
			Namespace: c.db.Namespace,
		},
	}

	cop, err := clientutil.CreateOrPatch(c.ctx, c.KBClient, ps, func(obj client.Object, createOp bool) client.Object {
		in := obj.(*psapi.PetSet)
		in.Labels = c.db.OffshootLabels()
		in.Annotations = podTemplate.Controller.Annotations
		in.Spec.Template.Labels = c.db.PodLabels()
		in.Spec.Selector = &meta.LabelSelector{
			MatchLabels: c.db.OffshootLabels(),
		}
		if createOp {
			coreutil.EnsureOwnerReference(&in.ObjectMeta, c.db.Owner())
		}
		replica := 0
		if c.db.Spec.Topology == "Cluster" {
			clusters := c.db.Spec.Cluster
			for _, cluster := range clusters {
				rep := ConvertInt32PtrToInt(cluster.Replicas)
				shard := ConvertInt32PtrToInt(cluster.Shards)
				replica += (rep * shard)
			}
		} else {
			replica = ConvertInt32PtrToInt(c.db.Spec.Replicas)
		}
		in.Spec.Replicas = ConvertIntToInt32Ptr(replica)
		in.Spec.ServiceName = c.db.GoverningServiceName()

		in.Spec.Template.Spec.InitContainers = coreutil.UpsertContainer(in.Spec.Template.Spec.InitContainers, initContainer)
		in.Spec.Template.Spec.Containers = coreutil.UpsertContainers(in.Spec.Template.Spec.Containers, containers)
		in.Spec.Template.Spec.Volumes = coreutil.UpsertVolume(in.Spec.Template.Spec.Volumes, volumes...)
		if pvc != nil {
			in.Spec.VolumeClaimTemplates = coreutil.UpsertVolumeClaim(in.Spec.VolumeClaimTemplates, *pvc)
		}
		in.Spec.Template.Spec.NodeSelector = c.db.Spec.PodTemplate.Spec.NodeSelector
		if c.db.Spec.PodTemplate.Spec.SchedulerName != "" {
			in.Spec.Template.Spec.SchedulerName = podTemplate.Spec.SchedulerName
		}
		in.Spec.Template.Spec.Tolerations = podTemplate.Spec.Tolerations
		in.Spec.Template.Spec.ImagePullSecrets = podTemplate.Spec.ImagePullSecrets
		in.Spec.Template.Spec.PriorityClassName = podTemplate.Spec.PriorityClassName
		in.Spec.Template.Spec.Priority = podTemplate.Spec.Priority
		in.Spec.Template.Spec.HostNetwork = podTemplate.Spec.HostNetwork
		in.Spec.Template.Spec.HostPID = podTemplate.Spec.HostPID
		in.Spec.Template.Spec.HostIPC = podTemplate.Spec.HostIPC
		in.Spec.Template.Spec.SecurityContext = podTemplate.Spec.SecurityContext
		if c.db.Spec.PodTemplate.Spec.DNSPolicy != "" {
			in.Spec.Template.Spec.DNSPolicy = podTemplate.Spec.DNSPolicy
		}
		if c.db.Spec.PodTemplate.Spec.TerminationGracePeriodSeconds != nil {
			in.Spec.Template.Spec.TerminationGracePeriodSeconds = podTemplate.Spec.TerminationGracePeriodSeconds
		}
		if c.db.Spec.PodTemplate.Spec.RuntimeClassName != nil {
			in.Spec.Template.Spec.RuntimeClassName = podTemplate.Spec.RuntimeClassName
		}
		if c.db.Spec.PodTemplate.Spec.EnableServiceLinks != nil {
			in.Spec.Template.Spec.EnableServiceLinks = podTemplate.Spec.EnableServiceLinks
		}
		// PetSet update strategy is set default to "OnDelete"
		in.Spec.UpdateStrategy = apps.StatefulSetUpdateStrategy{
			Type: apps.OnDeleteStatefulSetStrategyType,
		}
		// in.Spec.PodPlacementPolicy = r.db.Spec.PodPlacementPolicy

		return in
	})
	if err != nil {
		return err
	}
	if cop == kutil.VerbCreated {
		c.Log.Info(fmt.Sprintf("PetSet %s/%s created", ps.Namespace, ps.Name))
	}

	// ensure pdb
	if err := c.SyncPetSetPodDisruptionBudget(ps); err != nil {
		c.Log.Error(err, "Failed to create/patch PodDisruptionBudget")
		return err
	}
	return nil
}

func (c *ClickHouseReconciler) getPVC() *core.PersistentVolumeClaim {
	pvc := &core.PersistentVolumeClaim{
		ObjectMeta: meta.ObjectMeta{
			Name: c.db.PVCName(api.ClickHouseVolumeData),
		},
	}
	if len(c.db.Spec.Storage.AccessModes) == 0 {
		pvc.Spec.AccessModes = []core.PersistentVolumeAccessMode{
			core.ReadWriteOnce,
		}
	} else {
		pvc.Spec.AccessModes = c.db.Spec.Storage.AccessModes
	}
	if c.db.Spec.Storage.StorageClassName != nil {
		pvc.Spec.StorageClassName = c.db.Spec.Storage.StorageClassName
	}

	if c.db.Spec.Storage.Resources.Requests != nil {
		pvc.Spec.Resources.Requests = c.db.Spec.Storage.Resources.Requests
	}
	return pvc
}

func (c *ClickHouseReconciler) getVolumes() []core.Volume {
	// User provided custom volume (if any)
	volumes := ofst.ConvertVolumes(c.db.Spec.PodTemplate.Spec.Volumes)
	// Upsert Volume for configuration directory
	volumes = coreutil.UpsertVolume(volumes, []core.Volume{
		{
			Name: api.ClickHouseTempClusterConfigVolName,
			VolumeSource: core.VolumeSource{
				Secret: &core.SecretVolumeSource{
					SecretName: c.db.ConfigSecretName(),
					Items: []core.KeyToPath{
						{
							Key:  api.ClickHouseClusterConfigFileName,
							Path: api.ClickHouseClusterConfigFileName,
						},
						{
							Key:  api.ClickHouseClusterConfigFileName,
							Path: api.ClickHouseClusterConfigFileName,
						},
					},
				},
			},
		},
		{
			Name: api.ClickHouseTempConfigVolName,
			VolumeSource: core.VolumeSource{
				EmptyDir: &core.EmptyDirVolumeSource{},
			},
		},
	}...)
	return volumes
}

func (c *ClickHouseReconciler) getDBVolumeMounts() []core.VolumeMount {
	var volumeMounts []core.VolumeMount

	container := coreutil.GetContainerByName(c.db.Spec.PodTemplate.Spec.Containers, api.ClickHouseContainerName)
	if container != nil {
		volumeMounts = container.VolumeMounts
	}

	volumeMounts = coreutil.UpsertVolumeMount(volumeMounts, []core.VolumeMount{
		{
			Name:      c.db.PVCName(api.ClickHouseVolumeData),
			MountPath: api.ClickHouseDataDir,
			ReadOnly:  false,
		},
		{
			Name:      api.ClickHouseTempClusterConfigVolName,
			MountPath: api.ClickHouseTempConfigDir,
			ReadOnly:  false,
		},
		{
			Name:      api.ClickHouseTempConfigVolName,
			MountPath: "/etc/clickhouse-server/config.d/",
		},
	}...)

	return volumeMounts
}

func (c *ClickHouseReconciler) getInitVolumeMounts() []core.VolumeMount {
	var volumeMounts []core.VolumeMount

	container := coreutil.GetContainerByName(c.db.Spec.PodTemplate.Spec.Containers, api.ClickHouseContainerName)
	if container != nil {
		volumeMounts = container.VolumeMounts
	}

	volumeMounts = coreutil.UpsertVolumeMount(volumeMounts, []core.VolumeMount{
		{
			Name:      api.ClickHouseTempClusterConfigVolName,
			MountPath: api.ClickHouseTempConfigDir,
			ReadOnly:  false,
		},
		{
			Name:      api.ClickHouseTempConfigVolName,
			MountPath: api.ClickHouseTempDir,
		},
	}...)

	return volumeMounts
}
