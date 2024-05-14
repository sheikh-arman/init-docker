/*
Copyright AppsCode Inc. and Contributors

Licensed under the AppsCode Community License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Community-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"fmt"
	"log"

	api "kubedb.dev/apimachinery/apis/kubedb/v1alpha2"

	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	kutil "kmodules.xyz/client-go"
	clientutil "kmodules.xyz/client-go/client"
	coreutil "kmodules.xyz/client-go/core/v1"
	meta_util "kmodules.xyz/client-go/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (c *ClickHouseReconciler) ensureConfigSecrets() error {
	if c.db.Spec.Topology != "Cluster" {
		return nil
	}
	log.Println("\n\n\n\nIs it calling !??????????? 1")
	defaultConf := map[string]string{}
	_ = defaultConf

	// ensure secret with configurations
	configSecret := &core.Secret{
		ObjectMeta: meta.ObjectMeta{
			Name:      c.db.ConfigSecretName(),
			Namespace: c.db.Namespace,
		},
	}
	podConfig := ""
	remoteServer := fmt.Sprintf("" +
		"      remote_servers:\n" +
		"        \"@replace\": replace\n ")
	clusters := c.db.Spec.Cluster
	pod := 0
	for _, cluster := range clusters {
		clusterName := cluster.Name
		replica := ConvertInt32PtrToInt(cluster.Replicas)
		shard := ConvertInt32PtrToInt(cluster.Shards)
		remoteServer += fmt.Sprintf("       %s:\n", clusterName)
		for shardNo := 0; shardNo < shard; shardNo += 1 {
			remoteServer += fmt.Sprintf("          shard:\n")
			remoteServer += fmt.Sprintf("            internal_replication: true\n")
			for replicaNo := 0; replicaNo < replica; replicaNo += 1 {
				remoteServer += fmt.Sprintf("            replica:\n"+
					"              host: %s-%d.%s\n"+
					"              port: 9000\n", c.db.OffshootName(), pod, c.db.PrimaryServiceDNS())
				podConfig += fmt.Sprintf("%s-%d:\n"+
					"	CLUSTER: %s\n"+
					"	SHARD: %d\n"+
					"	REPLICA: %d\n"+
					"	INSTALLATION: %s\n", c.db.OffshootName(), pod, clusterName, shard+1, replicaNo+1, c.db.OffshootName())
				pod += 1
			}
		}
	}

	config := map[string][]byte{
		api.ClickHouseClusterConfigFileName:    []byte(remoteServer),
		api.ClickHouseClusterPodConfigFileName: []byte(podConfig),
	}

	v, err := clientutil.CreateOrPatch(c.ctx, c.KBClient, configSecret, func(obj client.Object, createOp bool) client.Object {
		secret := obj.(*core.Secret)
		secret.Labels = meta_util.OverwriteKeys(secret.Labels, c.db.OffshootLabels())
		if createOp {
			coreutil.EnsureOwnerReference(&secret.ObjectMeta, c.db.Owner())
		}
		secret.Data = config
		return secret
	})
	if err != nil {
		c.Log.Info("Failed to reconcile configuration Secret")
		return err
	}

	if v == kutil.VerbCreated {
		c.Log.Info(fmt.Sprintf("Configuration Secret %s/%s created", configSecret.GetName(), configSecret.GetNamespace()))
	}
	return nil
}

// ConvertIntToInt32Ptr converts an *int32 to a int
func ConvertInt32PtrToInt(ptr *int32) int {
	if ptr == nil {
		// Handle the nil pointer case appropriately
		// Here we return 0, but you could choose to handle it differently
		return 0
	}
	return int(*ptr)
}

// ConvertIntToInt32Ptr converts an int to a *int32
func ConvertIntToInt32Ptr(i int) *int32 {
	// Convert the int to int32
	val := int32(i)
	// Return the address of the int32 value
	return &val
}
