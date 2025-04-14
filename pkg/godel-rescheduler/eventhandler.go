// Copyright 2024 The Godel Rescheduler Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package godel_rescheduler

import (
	"github.com/kubewharf/godel-rescheduler/pkg/util"

	nodev1alpha1 "github.com/kubewharf/godel-scheduler-api/pkg/apis/node/v1alpha1"
	schedulingv1a1 "github.com/kubewharf/godel-scheduler-api/pkg/apis/scheduling/v1alpha1"
	crdinformers "github.com/kubewharf/godel-scheduler-api/pkg/client/informers/externalversions"
	podutil "github.com/kubewharf/godel-scheduler/pkg/util/pod"
	katalystv1alpha1 "github.com/kubewharf/katalyst-api/pkg/apis/node/v1alpha1"
	katalystinformers "github.com/kubewharf/katalyst-api/pkg/client/informers/externalversions"
	v1 "k8s.io/api/core/v1"
	policy "k8s.io/api/policy/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

func (rescheduler *Rescheduler) addNodeToCache(obj interface{}) {
	node, ok := obj.(*v1.Node)
	if !ok {
		klog.Errorf("cannot convert to *v1.Node: %v", obj)
		return
	}

	if err := rescheduler.cache.AddNode(node); err != nil {
		klog.Errorf("rescheduler cache AddNode failed: %v", err)
	}

	klog.V(6).Infof("add event for node %q", node.Name)
}

func (rescheduler *Rescheduler) updateNodeInCache(oldObj, newObj interface{}) {
	oldNode, ok := oldObj.(*v1.Node)
	if !ok {
		klog.Errorf("cannot convert oldObj to *v1.Node: %v", oldObj)
		return
	}
	newNode, ok := newObj.(*v1.Node)
	if !ok {
		klog.Errorf("cannot convert newObj to *v1.Node: %v", newObj)
		return
	}

	if err := rescheduler.cache.UpdateNode(oldNode, newNode); err != nil {
		klog.Errorf("rescheduler cache UpdateNode failed: %v", err)
	}
}

func (rescheduler *Rescheduler) deleteNodeFromCache(obj interface{}) {
	var node *v1.Node
	switch t := obj.(type) {
	case *v1.Node:
		node = t
	case cache.DeletedFinalStateUnknown:
		var ok bool
		node, ok = t.Obj.(*v1.Node)
		if !ok {
			klog.Errorf("cannot convert to *v1.Node: %v", t.Obj)
			return
		}
	default:
		klog.Errorf("cannot convert to *v1.Node: %v", t)
		return
	}
	klog.V(6).Infof("delete event for node %q", node.Name)
	// NOTE: Updates must be written to rescheduler cache before invalidating
	// equivalence cache, because we could snapshot equivalence cache after the
	// invalidation and then snapshot the cache itself. If the cache is
	// snapshotted before updates are written, we would update equivalence
	// cache with stale information which is based on snapshot of old cache.
	if err := rescheduler.cache.RemoveNode(node); err != nil {
		klog.Errorf("rescheduler cache RemoveNode failed: %v", err)
	}
}

func (rescheduler *Rescheduler) addNMNodeToCache(obj interface{}) {
	nmNode, ok := obj.(*nodev1alpha1.NMNode)
	if !ok {
		klog.Errorf("cannot convert to *nodev1alpha1.NMNode: %v", obj)
		return
	}

	if err := rescheduler.cache.AddNMNode(nmNode); err != nil {
		klog.Errorf("rescheduler cache AddNMNode failed: %v", err)
		return
	}
}

func (rescheduler *Rescheduler) updateNMNodeInCache(oldObj, newObj interface{}) {
	oldNMNode, ok := oldObj.(*nodev1alpha1.NMNode)
	if !ok {
		klog.Errorf("cannot convert oldObj to *nodev1alpha1.NMNode: %v", oldObj)
		return
	}
	newNMNode, ok := newObj.(*nodev1alpha1.NMNode)
	if !ok {
		klog.Errorf("cannot convert newObj to *nodev1alpha1.NMNode: %v", newObj)
		return
	}

	if err := rescheduler.cache.UpdateNMNode(oldNMNode, newNMNode); err != nil {
		klog.Errorf("rescheduler cache UpdateNMNode failed: %v", err)
	}
}

func (rescheduler *Rescheduler) deleteNMNodeFromCache(obj interface{}) {
	var nmNode *nodev1alpha1.NMNode
	switch t := obj.(type) {
	case *nodev1alpha1.NMNode:
		nmNode = t
	case cache.DeletedFinalStateUnknown:
		var ok bool
		nmNode, ok = t.Obj.(*nodev1alpha1.NMNode)
		if !ok {
			klog.Errorf("cannot convert to *nodev1alpha1.NMNode: %v", t.Obj)
			return
		}
	default:
		klog.Errorf("cannot convert to *nodev1alpha1.NMNode: %v", t)
		return
	}
	klog.V(6).Infof("delete event for nmNode %q", nmNode.Name)
	// NOTE: Updates must be written to rescheduler cache before invalidating
	// equivalence cache, because we could snapshot equivalence cache after the
	// invalidation and then snapshot the cache itself. If the cache is
	// snapshotted before updates are written, we would update equivalence
	// cache with stale information which is based on snapshot of old cache.
	if err := rescheduler.cache.RemoveNMNode(nmNode); err != nil {
		klog.Errorf("rescheduler cache RemoveNMNode failed: %v", err)
	}
}

func (rescheduler *Rescheduler) addCNRToCache(obj interface{}) {
	cnr, ok := obj.(*katalystv1alpha1.CustomNodeResource)
	if !ok {
		klog.Errorf("cannot convert to *katalystv1alpha1.CustomNodeResource: %v", obj)
		return
	}

	if err := rescheduler.cache.AddCNR(cnr); err != nil {
		klog.Errorf("rescheduler cache AddCNR failed: %v", err)
		return
	}
}

func (rescheduler *Rescheduler) updateCNRInCache(oldObj, newObj interface{}) {
	oldCNR, ok := oldObj.(*katalystv1alpha1.CustomNodeResource)
	if !ok {
		klog.Errorf("cannot convert oldObj to *katalystv1alpha1.CustomNodeResource: %v", oldObj)
		return
	}
	newCNR, ok := newObj.(*katalystv1alpha1.CustomNodeResource)
	if !ok {
		klog.Errorf("cannot convert newObj to *katalystv1alpha1.CustomNodeResource: %v", newObj)
		return
	}

	if err := rescheduler.cache.UpdateCNR(oldCNR, newCNR); err != nil {
		klog.Errorf("rescheduler cache UpdateCNR failed: %v", err)
	}
}

func (rescheduler *Rescheduler) deleteCNRFromCache(obj interface{}) {
	var cnr *katalystv1alpha1.CustomNodeResource
	switch t := obj.(type) {
	case *katalystv1alpha1.CustomNodeResource:
		cnr = t
	case cache.DeletedFinalStateUnknown:
		var ok bool
		cnr, ok = t.Obj.(*katalystv1alpha1.CustomNodeResource)
		if !ok {
			klog.Errorf("cannot convert to *katalystv1alpha1.CustomNodeResource: %v", t.Obj)
			return
		}
	default:
		klog.Errorf("cannot convert to *katalystv1alpha1.CustomNodeResource: %v", t)
		return
	}
	klog.V(6).Infof("delete event for cnr %q", cnr.Name)
	// NOTE: Updates must be written to rescheduler cache before invalidating
	// equivalence cache, because we could snapshot equivalence cache after the
	// invalidation and then snapshot the cache itself. If the cache is
	// snapshotted before updates are written, we would update equivalence
	// cache with stale information which is based on snapshot of old cache.
	if err := rescheduler.cache.RemoveCNR(cnr); err != nil {
		klog.Errorf("rescheduler cache RemoveCNR failed: %v", err)
	}
}

func (rescheduler *Rescheduler) addPodToCache(obj interface{}) {
	pod, err := podutil.ConvertToPod(obj)
	if err != nil {
		klog.Errorf("failed to add pod to cache: %v", err.Error())
		return
	}
	klog.V(6).Infof("add event for scheduled pod %s/%s with assigned node %s", pod.Namespace, pod.Name, pod.Spec.NodeName)

	if err := rescheduler.cache.AddPod(pod); err != nil {
		klog.Errorf("rescheduler cache AddPod failed: %v", err)
	}
}

func (rescheduler *Rescheduler) updatePodInCache(oldObj, newObj interface{}) {
	oldPod, err := podutil.ConvertToPod(oldObj)
	if err != nil {
		klog.Errorf("failed to update pod with oldObj: %v", err.Error())
		return
	}

	newPod, err := podutil.ConvertToPod(newObj)
	if err != nil {
		klog.Errorf("failed to update pod with newObj: %v", err.Error())
		return
	}

	klog.V(6).Infof("update event for scheduled pod %s/%s", newPod.Namespace, newPod.Name)

	// A Pod delete event followed by an immediate Pod add event may be merged
	// into a Pod update event. In this case, we should invalidate the old Pod, and
	// then add the new Pod.
	if oldPod.UID != newPod.UID {
		rescheduler.deletePodFromCache(oldObj)
		rescheduler.addPodToCache(newObj)
		return
	}

	// NOTE: Updates must be written to rescheduler cache before invalidating
	// equivalence cache, because we could snapshot equivalence cache after the
	// invalidation and then snapshot the cache itself. If the cache is
	// snapshotted before updates are written, we would update equivalence
	// cache with stale information which is based on snapshot of old cache.
	if err := rescheduler.cache.UpdatePod(oldPod, newPod); err != nil {
		klog.Errorf("rescheduler cache UpdatePod failed: %v", err)
	}
}

func (rescheduler *Rescheduler) deletePodFromCache(obj interface{}) {
	pod, err := podutil.ConvertToPod(obj)
	if err != nil {
		klog.Errorf("failed to delete pod from cache: %v", err.Error())
		return
	}
	klog.V(6).Infof("delete event for scheduled pod %s/%s", pod.Namespace, pod.Name)
	// NOTE: Updates must be written to rescheduler cache before invalidating
	// equivalence cache, because we could snapshot equivalence cache after the
	// invalidation and then snapshot the cache itself. If the cache is
	// snapshotted before updates are written, we would update equivalence
	// cache with stale information which is based on snapshot of old cache.
	if err := rescheduler.cache.RemovePod(pod); err != nil {
		klog.Errorf("rescheduler cache RemovePod failed: %v", err)
	}
}

func (rescheduler *Rescheduler) onPdbAdd(obj interface{}) {
	pdb, ok := obj.(*policy.PodDisruptionBudget)
	if !ok {
		klog.Errorf("cannot convert to *policy.PodDisruptionBudget: %v", obj)
		return
	}

	klog.V(6).Infof("add event for pdb %s", util.GetPDBKey(pdb))
	if err := rescheduler.cache.AddPDB(pdb); err != nil {
		klog.ErrorS(err, "failed to add pdb", "pdb", pdb)
	}
}

func (rescheduler *Rescheduler) onPdbUpdate(oldObj, newObj interface{}) {
	oldPdb, ok := oldObj.(*policy.PodDisruptionBudget)
	if !ok {
		klog.Errorf("cannot convert to *policy.PodDisruptionBudget: %v", oldObj)
		return
	}
	newPdb, ok := newObj.(*policy.PodDisruptionBudget)
	if !ok {
		klog.Errorf("cannot convert to *policy.PodDisruptionBudget: %v", newObj)
		return
	}
	klog.V(6).Infof("update event for pdb %s ", util.GetPDBKey(newPdb))
	if err := rescheduler.cache.UpdatePDB(oldPdb, newPdb); err != nil {
		klog.ErrorS(err, "failed to update pdb", "oldPdb", oldPdb, "newPdb", oldPdb)
	}
}

func (rescheduler *Rescheduler) onPdbDelete(obj interface{}) {
	pdb, ok := obj.(*policy.PodDisruptionBudget)
	if !ok {
		klog.Errorf("cannot convert to *policy.PodDisruptionBudget: %v", obj)
		return
	}
	klog.V(6).Infof("delete event for pdb %s", util.GetPDBKey(pdb))
	if err := rescheduler.cache.DeletePDB(pdb); err != nil {
		klog.ErrorS(err, "failed to delete pdb", "pdb", pdb)
	}
}

func (rescheduler *Rescheduler) addPodGroupToCache(obj interface{}) {
	podGroup, ok := obj.(*schedulingv1a1.PodGroup)
	if !ok {
		klog.Errorf("cannot convert obj to *v1alpha1.PodGroup: %v", obj)
		return
	}

	klog.V(6).Infof("add event for pod group %s/%s", podGroup.Namespace, podGroup.Name)

	if err := rescheduler.cache.AddPodGroup(podGroup); err != nil {
		klog.Errorf("scheduler cache AddPodGroup failed: %v", err)
		return
	}
}

func (rescheduler *Rescheduler) updatePodGroupToCache(oldObj interface{}, newObj interface{}) {
	oldPodGroup, ok := oldObj.(*schedulingv1a1.PodGroup)
	if !ok {
		klog.Errorf("cannot convert oldObj to *v1alpha1.PodGroup: %v", oldObj)
		return
	}
	newPodGroup, ok := newObj.(*schedulingv1a1.PodGroup)
	if !ok {
		klog.Errorf("cannot convert newObj to *v1alpha1.PodGroup: %v", newObj)
		return
	}

	if oldPodGroup.UID != newPodGroup.UID {
		rescheduler.deletePodGroupFromCache(oldPodGroup)
		rescheduler.addPodGroupToCache(oldPodGroup)
	}

	klog.V(6).Infof("update event for pod group %s/%s", newPodGroup.Namespace, newPodGroup.Name)

	if err := rescheduler.cache.UpdatePodGroup(oldPodGroup, newPodGroup); err != nil {
		klog.Errorf("scheduler cache UpdatePodGroup failed: %v", err)
		return
	}
}

func (rescheduler *Rescheduler) deletePodGroupFromCache(obj interface{}) {
	var podGroup *schedulingv1a1.PodGroup
	switch t := obj.(type) {
	case *schedulingv1a1.PodGroup:
		podGroup = t
	case cache.DeletedFinalStateUnknown:
		var ok bool
		podGroup, ok = t.Obj.(*schedulingv1a1.PodGroup)
		if !ok {
			klog.Errorf("cannot convert to *v1.PodGroup: %v", t.Obj)
			return
		}
	default:
		klog.Errorf("cannot convert to *v1.PodGroup: %v", t)
		return
	}

	klog.V(6).Infof("delete event for pod group %s/%s", podGroup.Namespace, podGroup.Name)

	if err := rescheduler.cache.RemovePodGroup(podGroup); err != nil {
		klog.Errorf("scheduler cache RemovePodGroup failed: %v", err)
	}
}

// addAllEventHandlers is a helper function used in tests and in Rescheduler
// to add event handlers for various informers.
func addAllEventHandlers(
	rescheduler *Rescheduler,
	informerFactory informers.SharedInformerFactory,
	crdInformerFactory crdinformers.SharedInformerFactory,
	katalystInformerFactory katalystinformers.SharedInformerFactory,
) {
	// scheduled pod cache
	informerFactory.Core().V1().Pods().Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    rescheduler.addPodToCache,
			UpdateFunc: rescheduler.updatePodInCache,
			DeleteFunc: rescheduler.deletePodFromCache,
		},
	)

	informerFactory.Core().V1().Nodes().Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    rescheduler.addNodeToCache,
			UpdateFunc: rescheduler.updateNodeInCache,
			DeleteFunc: rescheduler.deleteNodeFromCache,
		},
	)

	crdInformerFactory.Node().V1alpha1().NMNodes().Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    rescheduler.addNMNodeToCache,
			UpdateFunc: rescheduler.updateNMNodeInCache,
			DeleteFunc: rescheduler.deleteNMNodeFromCache,
		},
	)

	katalystInformerFactory.Node().V1alpha1().CustomNodeResources().Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    rescheduler.addCNRToCache,
			UpdateFunc: rescheduler.updateCNRInCache,
			DeleteFunc: rescheduler.deleteCNRFromCache,
		},
	)

	informerFactory.Policy().V1().PodDisruptionBudgets().Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    rescheduler.onPdbAdd,
			UpdateFunc: rescheduler.onPdbUpdate,
			DeleteFunc: rescheduler.onPdbDelete,
		},
	)

	// add PodGroup resource event listener
	crdInformerFactory.Scheduling().V1alpha1().PodGroups().Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    rescheduler.addPodGroupToCache,
			UpdateFunc: rescheduler.updatePodGroupToCache,
			DeleteFunc: rescheduler.deletePodGroupFromCache,
		},
	)
}

// assignedPod selects pods that are assigned (scheduled or running).
func assignedPod(pod *v1.Pod) bool {
	return len(pod.Spec.NodeName) != 0
}

func assumedOrBoundPod(pod *v1.Pod) bool {
	return podutil.BoundPod(pod) || podutil.AssumedPod(pod)
}
