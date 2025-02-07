/*
Copyright 2022 NVIDIA CORPORATION & AFFILIATES
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package state //nolint:dupl

import (
	"strconv"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/source"

	mellanoxv1alpha1 "github.com/Mellanox/network-operator/api/v1alpha1"
	"github.com/Mellanox/network-operator/pkg/config"
	"github.com/Mellanox/network-operator/pkg/consts"
	"github.com/Mellanox/network-operator/pkg/nodeinfo"
	"github.com/Mellanox/network-operator/pkg/render"
	"github.com/Mellanox/network-operator/pkg/utils"
)

// NewStateIBKubernetes creates a new ib-kubernetes state
func NewStateIBKubernetes(k8sAPIClient client.Client, scheme *runtime.Scheme, manifestDir string) (State, error) {
	files, err := utils.GetFilesWithSuffix(manifestDir, render.ManifestFileSuffix...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get files from manifest dir")
	}

	renderer := render.NewRenderer(files)
	return &stateIBKubernetes{
		stateSkel: stateSkel{
			name:        "state-ib-kubernetes",
			description: "ib-kubernetes deployed in the cluster",
			client:      k8sAPIClient,
			scheme:      scheme,
			renderer:    renderer,
		}}, nil
}

type stateIBKubernetes struct {
	stateSkel
}

type IBKubernetesSpec struct {
	runtimeSpec
	OSName string
}
type IBKubernetesManifestRenderData struct {
	CrSpec                      *mellanoxv1alpha1.IBKubernetesSpec
	PeriodicUpdateSecondsString string
	NodeAffinity                *v1.NodeAffinity
	DeployInitContainer         bool
	RuntimeSpec                 *IBKubernetesSpec
}

// Sync attempt to get the system to match the desired state which State represent.
// a sync operation must be relatively short and must not block the execution thread.
//
//nolint:dupl
func (s *stateIBKubernetes) Sync(customResource interface{}, infoCatalog InfoCatalog) (SyncState, error) {
	cr := customResource.(*mellanoxv1alpha1.NicClusterPolicy)
	log.V(consts.LogLevelInfo).Info(
		"Sync Custom resource", "State:", s.name, "Name:", cr.Name, "Namespace:", cr.Namespace)

	if cr.Spec.IBKubernetes == nil {
		// Either this state was not required to run or an update occurred and we need to remove
		// the resources that where created.
		log.V(consts.LogLevelInfo).Info("ib-kubernetes spec in CR is nil, no action required")
		return SyncStateIgnore, nil
	}

	nodeInfo := infoCatalog.GetNodeInfoProvider()
	if nodeInfo == nil {
		return SyncStateError, errors.New("unexpected state, catalog does not provide node information")
	}

	objs, err := s.getManifestObjects(cr, nodeInfo)
	if err != nil {
		return SyncStateNotReady, errors.Wrap(err, "failed to create k8s objects from manifest")
	}
	if len(objs) == 0 {
		return SyncStateNotReady, nil
	}

	// Create objects if they dont exist, Update objects if they do exist
	err = s.createOrUpdateObjs(func(obj *unstructured.Unstructured) error {
		if err := controllerutil.SetControllerReference(cr, obj, s.scheme); err != nil {
			return errors.Wrap(err, "failed to set controller reference for object")
		}
		return nil
	}, objs)
	if err != nil {
		return SyncStateNotReady, errors.Wrap(err, "failed to create/update objects")
	}
	// Check objects status
	syncState, err := s.getSyncState(objs)
	if err != nil {
		return SyncStateNotReady, errors.Wrap(err, "failed to get sync state")
	}
	return syncState, nil
}

// Get a map of source kinds that should be watched for the state keyed by the source kind name
func (s *stateIBKubernetes) GetWatchSources() map[string]*source.Kind {
	wr := make(map[string]*source.Kind)
	wr["Deployment"] = &source.Kind{Type: &appsv1.Deployment{}}
	return wr
}

func (s *stateIBKubernetes) getManifestObjects(
	cr *mellanoxv1alpha1.NicClusterPolicy, nodeInfo nodeinfo.Provider) ([]*unstructured.Unstructured, error) {
	attrs := nodeInfo.GetNodesAttributes(
		nodeinfo.NewNodeLabelFilterBuilder().
			Build())
	if len(attrs) == 0 {
		log.V(consts.LogLevelInfo).Info("No nodes with Mellanox NICs where found in the cluster.")
		return []*unstructured.Unstructured{}, nil
	}

	renderData := &IBKubernetesManifestRenderData{
		CrSpec:                      cr.Spec.IBKubernetes,
		PeriodicUpdateSecondsString: strconv.Itoa(cr.Spec.IBKubernetes.PeriodicUpdateSeconds),
		NodeAffinity:                cr.Spec.NodeAffinity,
		DeployInitContainer:         cr.Spec.OFEDDriver != nil,
		RuntimeSpec: &IBKubernetesSpec{
			runtimeSpec: runtimeSpec{config.FromEnv().State.NetworkOperatorResourceNamespace},
			OSName:      attrs[0].Attributes[nodeinfo.AttrTypeOSName],
		},
	}
	// render objects
	log.V(consts.LogLevelDebug).Info("Rendering objects", "data:", renderData)
	objs, err := s.renderer.RenderObjects(&render.TemplatingData{Data: renderData})
	if err != nil {
		return nil, errors.Wrap(err, "failed to render objects")
	}
	log.V(consts.LogLevelDebug).Info("Rendered", "objects:", objs)
	return objs, nil
}
