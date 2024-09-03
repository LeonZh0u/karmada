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

package priorityclass

import (
	"context"
	"fmt"
	"sort"

	"github.com/karmada-io/karmada/pkg/estimator/pb"
	"github.com/karmada-io/karmada/pkg/estimator/server/framework"
	"github.com/karmada-io/karmada/pkg/estimator/server/preemptee"
	"github.com/karmada-io/karmada/pkg/features"
	"github.com/karmada-io/karmada/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

const (
	// Name is the name of the plugin used in Registry and configurations.
	Name = "DefaultPreemptionOrderEstimate"
)

// resourceQuotaEstimator is to estimate how many replica allowed by the ResourceQuota constrain for a given pb.ReplicaRequirements
// Kubernetes ResourceQuota object provides constraints that limit aggregate resource consumption per namespace
type PreemptionOrderEstimator struct {
	enabled bool
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

// Name returns name of the plugin. It is used in logs, etc.
func (pl *PreemptionOrderEstimator) Name() string {
	return Name
}

// Estimate the replica allowed by the ResourceQuota
func (pl *PreemptionOrderEstimator) Order(_ context.Context,
	victims *map[string]preemptee.CandidateVictim, request *pb.PreemptionRequest) ([]*preemptee.CandidateVictim, *framework.Result) {
	if !pl.enabled {
		klog.V(5).Info("PriorityClass Filter Plugin", "name", Name, "enabled", pl.enabled)
		return []*preemptee.CandidateVictim{}, framework.NewResult(framework.Noopperation, fmt.Sprintf("%s is disabled", pl.Name()))
	}
	// Convert CandidateVictim map to slice
	victimsSlice := make([]*preemptee.CandidateVictim, len(*victims))
	for _, victim := range *victims {
		victimsSlice = append(victimsSlice, &victim)
	}
	// Sort CandidateVictim slice by priority first and then break ties by total resource.
	sort.SliceStable(victimsSlice, func(i, j int) bool {
		// Sort by priority
		if *victimsSlice[i].Priority != *victimsSlice[j].Priority {
			return *victimsSlice[i].Priority > *victimsSlice[j].Priority
		}
		// Break ties by total resource
		resourcei := util.Resource{}
		for _, resource := range victimsSlice[i].NodeResources {
			resourcei.Add(resource.ResourceList())
		}
		resourcej := util.Resource{}
		for _, resource := range victimsSlice[j].NodeResources {
			resourcei.Add(resource.ResourceList())
		}
		return weightedResourceScore(resourcei.ResourceList()) > weightedResourceScore(resourcej.ResourceList())
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
	corev1.ResourceCPU:              0,
	corev1.ResourceMemory:           0,
	corev1.ResourceStorage:          0,
	corev1.ResourceEphemeralStorage: 0,
	"nvidia.com/gpu":                1,
}

func weightedResourceScore(resource corev1.ResourceList) int64 {
	var score int64 = 0
	for resourceType, weight := range resourceWeightChart {
		resource := resource[resourceType]
		value, ok := resource.AsInt64()
		if ok {
			score += value * weight
		}

	}
	return score
}
