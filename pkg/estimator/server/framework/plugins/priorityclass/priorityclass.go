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

	"k8s.io/klog/v2"

	"github.com/karmada-io/karmada/pkg/estimator/pb"
	"github.com/karmada-io/karmada/pkg/estimator/server/framework"
	"github.com/karmada-io/karmada/pkg/estimator/server/preemptee"
	"github.com/karmada-io/karmada/pkg/features"
)

const (
	// Name is the name of the plugin used in Registry and configurations.
	Name                   = "PriorityClassFilter"
	resourceRequestsPrefix = "requests."
	resourceLimitsPrefix   = "limits."
)

// PriorityClassFilter is to filter out victim resourcebindings with higher or null priorityClass value for a given pb.PreemptionRequest
// Kubernetes PriorityClass object defines the relative importance of a pod for preemption.
type PriorityClassFilter struct {
	enabled bool
}

var _ framework.PreemptionFilterPlugin = &PriorityClassFilter{}

// New initializes a new plugin and returns it.
func New(fh framework.Handle) (framework.Plugin, error) {
	enabled := features.FeatureGate.Enabled(features.PriorityClassFilterEstimate)
	if !enabled {
		// Disabled, won't do anything.
		return &PriorityClassFilter{}, nil
	}
	return &PriorityClassFilter{
		enabled: enabled,
	}, nil
}

// Name returns name of the plugin. It is used in logs, etc.
func (pl *PriorityClassFilter) Name() string {
	return Name
}

// Filter/remove the pods with higher (or null) priority than the request.
func (pl *PriorityClassFilter) Filter(_ context.Context,
	victims *map[string]preemptee.CandidateVictim, request *pb.PreemptionRequest) *framework.Result {
	if !pl.enabled {
		klog.V(5).Info("PriorityClass Filter Plugin", "name", Name, "enabled", pl.enabled)
		return framework.NewResult(framework.Noopperation, fmt.Sprintf("%s is disabled", pl.Name()))
	}
	priorityValue := request.ReplicaRequirements.Priority

	for rbID, victim := range *victims {
		if victim.Priority == nil || priorityValue <= *victim.Priority {
			delete(*victims, rbID)
		}
	}

	var result *framework.Result
	if len(*victims) == 0 {
		result = framework.NewResult(framework.Unschedulable, fmt.Sprintf("zero candidate victims remain after filtering by %s", pl.Name()))
	} else {
		result = framework.NewResult(framework.Success)
	}
	return result
}
