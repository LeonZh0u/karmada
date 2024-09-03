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

	workv1alpha2 "github.com/karmada-io/karmada/pkg/apis/work/v1alpha2"
	"github.com/karmada-io/karmada/pkg/estimator/pb"
	"github.com/karmada-io/karmada/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CandidateVictim struct {
	NodeResources   map[string]*util.Resource
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

			if _, ok := candidateVictims[permanentID]; !ok {
				// Construct a new candidate victim
				candidateVictims[permanentID] = CandidateVictim{
					NodeResources: map[string]*util.Resource{},
					ResourceBinding: &pb.ObjectReference{
						APIVersion: workv1alpha2.SchemeGroupVersion.String(),
						Kind:       kind,
						Namespace:  pod.Namespace,
						Name:       pod.Annotations[workv1alpha2.ResourceBindingNameAnnotationKey],
					},
					Priority: pod.Spec.Priority,
				}
			} else {
				// Update the node resources for the candidate victim
				updatedNodeResource := candidateVictims[permanentID].NodeResources[nodeName].AddPodRequest(&pod.Spec)
				candidateVictims[permanentID].NodeResources[permanentID] = updatedNodeResource
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
	allPods, err := f(selector, podLister)
	if err != nil {
		return []*corev1.Pod{}, err
	}
	return allPods, nil
}
