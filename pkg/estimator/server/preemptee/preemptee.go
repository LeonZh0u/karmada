/*
Copyright 2021 The Karmada Authors.

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

package preemptee

import (
	corev1 "k8s.io/api/core/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	listcorev1 "k8s.io/client-go/listers/core/v1"

	// Add this line to import the klog package
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	workv1alpha2 "github.com/karmada-io/karmada/pkg/apis/work/v1alpha2"
	"github.com/karmada-io/karmada/pkg/estimator/pb"
	"github.com/karmada-io/karmada/pkg/util"
)

type CandidateVictim struct {
	// maps node name to the resource that candidate victim holds
	NodeResources map[string]*util.Resource
	// maps node name to the number of pods in the candidate victim
	NumVictimPods   map[string]int64
	ResourceBinding *pb.ObjectReference
	Priority        *int32
}

type PodListFunc func(labels.Selector, listcorev1.PodLister) ([]*corev1.Pod, error)

type PreempteeLister struct {
	//Handler    framework.Handle
	PodLister  listcorev1.PodLister
	NodeLister listcorev1.NodeLister
	//PdbLister  policylisters.PodDisruptionBudgetLister
	// State      *framework.CycleState
	// Interface
}

func ListCandidateVictims(kind string, pl *PreempteeLister) (
	*map[string]CandidateVictim, error) {
	podListFunc := func(selector labels.Selector, podLister listcorev1.PodLister) ([]*corev1.Pod, error) {
		// List pods from all namespaces
		ret, err := podLister.Pods("").List(selector)
		return ret, err
	}

	pods, err := listPods(kind, podListFunc, pl.PodLister)

	if err != nil {
		return nil, err
	}

	// Construct CandidateVictim objects
	// by grouping pods by label resourcebinding permanent-id
	// and attribute each rb resources onto their respective nodes
	candidateVictims := map[string]CandidateVictim{}
	for _, pod := range pods {
		permanentID := pod.Labels[workv1alpha2.ResourceBindingPermanentIDLabel]
		if permanentID != "" {
			nodeName := pod.Spec.NodeName

			// Construct a new candidate victim with permanentID maps to empty value
			if _, ok := candidateVictims[permanentID]; !ok {
				candidateVictims[permanentID] = CandidateVictim{
					NodeResources: map[string]*util.Resource{},
					ResourceBinding: &pb.ObjectReference{
						APIVersion: workv1alpha2.GroupVersion.Version,
						Kind:       workv1alpha2.ResourceKindResourceBinding,
						Namespace:  pod.Namespace,
						Name:       pod.Annotations[workv1alpha2.ResourceBindingNameAnnotationKey],
					},
					Priority:      pod.Spec.Priority,
					NumVictimPods: map[string]int64{},
				}
			}
			// Update the node resources under the current candidate victim
			if _, ok := candidateVictims[permanentID].NodeResources[nodeName]; !ok {
				candidateVictims[permanentID].NodeResources[nodeName] = &util.Resource{}
			}
			candidateVictims[permanentID].NodeResources[nodeName].AddPodRequest(&pod.Spec)
			if _, ok := candidateVictims[permanentID].NumVictimPods[nodeName]; !ok {
				candidateVictims[permanentID].NumVictimPods[nodeName] = 1
			} else {
				candidateVictims[permanentID].NumVictimPods[nodeName] += 1
			}
		}
	}
	return &candidateVictims, nil
}

func listPods(kind string, f PodListFunc, podLister listcorev1.PodLister) ([]*corev1.Pod, error) {
	selector, err := metav1.LabelSelectorAsSelector(
		&metav1.LabelSelector{
			MatchLabels: map[string]string{"app.kubernetes.io/kind": kind},
		})
	if err != nil {
		return nil, err
	}
	// allPods, err := podLister.Pods("").List(labels.Everything())
	// if err == nil {
	// 	return nil, fmt.Errorf("pods %d %s %s", len(allPods), kind, allPods[0])
	// }
	allPods, err := f(selector, podLister)
	if err != nil {
		return []*corev1.Pod{}, err
	}
	return allPods, nil
}
