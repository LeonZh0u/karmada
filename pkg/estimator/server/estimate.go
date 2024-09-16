/*
Copyright 2022 The Karmada Authors.

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

package server

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/karmada-io/karmada/pkg/estimator/server/preemptee"
	corev1 "k8s.io/api/core/v1"
	utiltrace "k8s.io/utils/trace"

	"github.com/karmada-io/karmada/pkg/estimator/pb"
	nodeutil "github.com/karmada-io/karmada/pkg/estimator/server/nodes"
	"github.com/karmada-io/karmada/pkg/util"
	schedcache "github.com/karmada-io/karmada/pkg/util/lifted/scheduler/cache"
	"github.com/karmada-io/karmada/pkg/util/lifted/scheduler/framework"
)

// EstimateReplicas returns max available replicas in terms of request and cluster status.
func (es *AccurateSchedulerEstimatorServer) EstimateReplicas(ctx context.Context, object string, request *pb.MaxAvailableReplicasRequest) (int32, error) {
	trace := utiltrace.New("Estimating", utiltrace.Field{Key: "namespacedName", Value: object})
	defer trace.LogIfLong(100 * time.Millisecond)

	snapShot := schedcache.NewEmptySnapshot()
	if err := es.Cache.UpdateSnapshot(snapShot); err != nil {
		return 0, err
	}
	trace.Step("Snapshotting estimator cache and node infos done")

	if snapShot.NumNodes() == 0 {
		return 0, nil
	}

	maxAvailableReplicas, err := es.estimateReplicas(ctx, snapShot, request.ReplicaRequirements)
	if err != nil {
		return 0, err
	}
	trace.Step("Computing estimation done")

	return maxAvailableReplicas, nil
}

func (es *AccurateSchedulerEstimatorServer) estimateReplicas(
	ctx context.Context,
	snapshot *schedcache.Snapshot,
	requirements pb.ReplicaRequirements,
) (int32, error) {
	allNodes, err := snapshot.NodeInfos().List()
	if err != nil {
		return 0, err
	}
	var (
		affinity    = nodeutil.GetRequiredNodeAffinity(requirements)
		tolerations []corev1.Toleration
	)

	if requirements.NodeClaim != nil {
		tolerations = requirements.NodeClaim.Tolerations
	}

	var res int32
	replicas, ret := es.estimateFramework.RunEstimateReplicasPlugins(ctx, snapshot, &requirements)

	// No replicas can be scheduled on the cluster, skip further checks and return 0
	if ret.IsUnschedulable() {
		return 0, nil
	}

	if !ret.IsSuccess() && !ret.IsNoOperation() {
		return replicas, fmt.Errorf(fmt.Sprintf("estimate replice plugins fails with %s", ret.Reasons()))
	}
	processNode := func(i int) {
		node := allNodes[i]
		if !nodeutil.IsNodeAffinityMatched(node.Node(), affinity) || !nodeutil.IsTolerationMatched(node.Node(), tolerations) {
			return
		}
		maxReplica := es.nodeMaxAvailableReplica(node, requirements.ResourceRequest)
		atomic.AddInt32(&res, maxReplica)
	}
	es.parallelizer.Until(ctx, len(allNodes), processNode)

	if ret.IsSuccess() && replicas < res {
		res = replicas
	}
	return res, nil
}

func (es *AccurateSchedulerEstimatorServer) nodeMaxAvailableReplica(node *framework.NodeInfo, rl corev1.ResourceList) int32 {
	rest := node.Allocatable.Clone().SubResource(node.Requested)
	// The number of pods in a node is a kind of resource in node allocatable resources.
	// However, total requested resources of all pods on this node, i.e. `node.Requested`,
	// do not contain pod resources. So after subtraction, we should cope with allowed pod
	// number manually which is the upper bound of this node available replicas.
	rest.AllowedPodNumber = util.MaxInt64(rest.AllowedPodNumber-int64(len(node.Pods)), 0)
	return int32(rest.MaxDivided(rl))
}

// SelectVictims returns max available replicas in terms of request and cluster status.
func (es *AccurateSchedulerEstimatorServer) SelectVictims(ctx context.Context, request *pb.PreemptionRequest) ([]pb.ObjectReference, error) {
	trace := utiltrace.New("Preempting", utiltrace.Field{Key: "namespacedName", Value: request.Preemptor})
	defer trace.LogIfLong(100 * time.Millisecond)

	snapShot := schedcache.NewEmptySnapshot()
	if err := es.Cache.UpdateSnapshot(snapShot); err != nil {
		return []pb.ObjectReference{}, err
	}
	trace.Step("Snapshotting estimator cache and node infos for preemption completed")

	if snapShot.NumNodes() == 0 {
		return []pb.ObjectReference{}, nil
	}

	candidateVictims, err := es.selectVictims(ctx, snapShot, request)
	if err != nil {
		return []pb.ObjectReference{}, err
	}
	trace.Step("Preemption completed")

	return *candidateVictims, nil
}

func (es *AccurateSchedulerEstimatorServer) selectVictims(
	ctx context.Context,
	snapshot *schedcache.Snapshot,
	request *pb.PreemptionRequest,
) (*[]pb.ObjectReference, error) {
	allNodes, err := snapshot.NodeInfos().List()
	if err != nil {
		return &[]pb.ObjectReference{}, err
	}
	var (
		requirements     = request.ReplicaRequirements
		requiredReplicas = request.NumReplicas
		affinity         = nodeutil.GetRequiredNodeAffinity(requirements)
		tolerations      []corev1.Toleration
	)

	if requirements.NodeClaim != nil {
		tolerations = requirements.NodeClaim.Tolerations
	}
	candidateVictims, err := preemptee.ListCandidateVictims(request.Preemptor.Kind, &es.PreempteeLister)
	if err != nil {
		return &[]pb.ObjectReference{}, err
	}
	// run preemption plugins to filter out eligible victims for preemption
	candidateVictims, ret := es.estimateFramework.RunPreemptionFilterPlugins(ctx, candidateVictims, request)

	// No victims can be preempted on the cluster, skip further checks and return 0
	if ret.IsUnschedulable() {
		return &[]pb.ObjectReference{}, nil
	}

	if !ret.IsSuccess() && !ret.IsNoOperation() {
		return &[]pb.ObjectReference{}, fmt.Errorf(fmt.Sprintf("preemption filter plugins fails with %s", ret.Reasons()))
	}

	orderedCandidateVictims, ret := es.estimateFramework.RunPreemptionOrderPlugins(ctx, candidateVictims, request)
	if !ret.IsSuccess() && !ret.IsNoOperation() {
		return &[]pb.ObjectReference{}, fmt.Errorf(fmt.Sprintf("preemption order plugins fails with %s", ret.Reasons()))
	}
	victimResourceBindings := make([]pb.ObjectReference, len(orderedCandidateVictims))
	for _, candidateVictim := range orderedCandidateVictims {
		victimResourceBindings = append(victimResourceBindings, *candidateVictim.ResourceBinding)
	}

	// Subtract all requested resources and pod numbers from allocatable for the node.
	for _, node := range allNodes {
		node.Allocatable.SubResource(node.Requested)
		node.Allocatable.AllowedPodNumber = util.MaxInt64(node.Allocatable.AllowedPodNumber-int64(len(node.Pods)), 0)
	}

	for cnt, candidateVictim := range orderedCandidateVictims {
		var victimRbs int32
		processNode := func(i int) {

			node := allNodes[i]
			if !nodeutil.IsNodeAffinityMatched(node.Node(), affinity) || !nodeutil.IsTolerationMatched(node.Node(), tolerations) {
				return
			}

			nodeName := node.Node().Name
			// Add each additional victim's released resources to the allocatable for the node
			resource := candidateVictim.NodeResources[nodeName].ResourceList()
			node.Allocatable.Add(resource)
			// Update AllowedPodNumber for the node with number of released pods
			node.Allocatable.AllowedPodNumber += candidateVictim.NumVictimPods[nodeName]
			// Calculate the max replicas that can be scheduled on the node after preempting the victim
			maxVictimReplica := int32(node.Allocatable.MaxDivided(requirements.ResourceRequest))
			atomic.AddInt32(&victimRbs, maxVictimReplica)
		}
		es.parallelizer.Until(ctx, len(allNodes), processNode)
		// If the schedulable replicas in cluster after preempting victims is greater than or equal to the required replicas, return victims
		if victimRbs >= requiredReplicas {
			selectedVictims := victimResourceBindings[:cnt+1]
			return &selectedVictims, nil
		}
	}
	if ret.IsSuccess() {
		return &victimResourceBindings, nil
	}
	return &[]pb.ObjectReference{}, fmt.Errorf(fmt.Sprintf("preemption failed with %s", ret.Reasons()))
}
