/*
Copyright 2024 The Karmada Authors.

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

package preemptionorder

import (
	"context"
	"fmt"
	"sort"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"

	"github.com/karmada-io/karmada/pkg/estimator/pb"
	"github.com/karmada-io/karmada/pkg/estimator/server/framework"
	"github.com/karmada-io/karmada/pkg/estimator/server/preemptee"
	"github.com/karmada-io/karmada/pkg/features"
	"github.com/karmada-io/karmada/pkg/util"
)

const (
	// Name is the name of the plugin used in Registry and configurations.
	Name = "DefaultPreemptionOrderEstimate"
)

var SortByPriority = func(o1, o2 *preemptee.CandidateVictim) bool {
	return *o1.Priority < *o2.Priority
}

var SortByWeightedResource = func(o1, o2 *preemptee.CandidateVictim) bool {
	resourcei := util.Resource{}
	for _, resource := range o1.NodeResources {
		resourcei.Add(resource.ResourceList())
	}
	resourcej := util.Resource{}
	for _, resource := range o2.NodeResources {
		resourcej.Add(resource.ResourceList())
	}
	return weightedResourceScore(resourcei.ResourceList()) > weightedResourceScore(resourcej.ResourceList())
}

// PreemptionOrderEstimator is to decide the relative order of the preemptable victim resource bindings by most resource released.
// By preempting victims that can release the most amount of resources, less user will be affected by each preemption.
type PreemptionOrderEstimator struct {
	orderedSortingCriterias []framework.OrderedSortingCriteria
	enabled                 bool
}

var _ framework.PreemptionOrderPlugin = &PreemptionOrderEstimator{}

// New initializes a new plugin and returns it.
func New(fh framework.Handle) (framework.Plugin, error) {
	enabled := features.FeatureGate.Enabled(features.DefaultPreemptionOrderEstimate)
	if !enabled {
		// Disabled, won't do anything.
		return &PreemptionOrderEstimator{}, nil
	}
	return &PreemptionOrderEstimator{
		enabled: enabled,
	}, nil
}

func (pl *PreemptionOrderEstimator) SetSortingCriterias(orderedSortingCriterias []framework.OrderedSortingCriteria) {
	pl.orderedSortingCriterias = orderedSortingCriterias
}

// Name returns name of the plugin. It is used in logs, etc.
func (pl *PreemptionOrderEstimator) Name() string {
	return Name
}

// Order the victims according to the amount of resources each resource binding holds.
func (pl *PreemptionOrderEstimator) Order(_ context.Context,
	victims *map[string]preemptee.CandidateVictim, request *pb.PreemptionRequest) ([]*preemptee.CandidateVictim, *framework.Result) {
	if !pl.enabled {
		klog.V(5).Info("PriorityClass Filter Plugin", "name", Name, "enabled", pl.enabled)
		return []*preemptee.CandidateVictim{}, framework.NewResult(framework.Noopperation, fmt.Sprintf("%s is disabled", pl.Name()))
	}
	// Convert CandidateVictim map to slice
	victimsSlice := []*preemptee.CandidateVictim{}
	for _, victim := range *victims {
		// Ignore victim resource binding with null priority
		if victim.Priority != nil {
			victimsSlice = append(victimsSlice, &victim)
		}
	}
	// Sort CandidateVictim slice by priority first and then break ties by total resource.
	sort.SliceStable(victimsSlice, func(i, j int) bool {
		for orderedSortingCriteria := range pl.orderedSortingCriterias {
			if pl.orderedSortingCriterias[orderedSortingCriteria](victimsSlice[i], victimsSlice[j]) {
				return true
			}
		}
		return false
	})
	var result *framework.Result
	if len(*victims) == 0 {
		result = framework.NewResult(framework.Unschedulable, fmt.Sprintf("zero candidate victims remain after filtering by %s", pl.Name()))
	} else {
		result = framework.NewResult(framework.Success)
	}
	return victimsSlice, result
}

var resourceWeightChart = map[corev1.ResourceName]int64{
	corev1.ResourceCPU:              1,
	corev1.ResourceMemory:           0,
	corev1.ResourceStorage:          0,
	corev1.ResourceEphemeralStorage: 0,
	"nvidia.com/gpu":                4,
}

func weightedResourceScore(resource corev1.ResourceList) int64 {
	var score int64 = 0
	gpus := resource["nvidia.com/gpu"]
	score += resource.Cpu().Value()*resourceWeightChart[corev1.ResourceCPU] +
		resource.Memory().Value()*resourceWeightChart[corev1.ResourceMemory] +
		resource.Storage().Value()*resourceWeightChart[corev1.ResourceStorage] +
		resource.StorageEphemeral().Value()*resourceWeightChart[corev1.ResourceEphemeralStorage] +
		gpus.Value()*resourceWeightChart["nvidia.com/gpu"]
	return score
}
