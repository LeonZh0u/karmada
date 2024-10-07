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

package runtime

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"time"

	"k8s.io/client-go/informers"
	clientset "k8s.io/client-go/kubernetes"

	"github.com/karmada-io/karmada/pkg/estimator/pb"
	"github.com/karmada-io/karmada/pkg/estimator/server/framework"
	"github.com/karmada-io/karmada/pkg/estimator/server/framework/plugins/preemptionorder"
	"github.com/karmada-io/karmada/pkg/estimator/server/metrics"
	"github.com/karmada-io/karmada/pkg/estimator/server/preemptee"
	schedcache "github.com/karmada-io/karmada/pkg/util/lifted/scheduler/cache"
	utilmetrics "github.com/karmada-io/karmada/pkg/util/metrics"
)

const (
	estimator = "Estimator"
)

// frameworkImpl implements the Framework interface and is responsible for initializing and running scheduler
// plugins.
type frameworkImpl struct {
	estimateReplicasPlugins []framework.EstimateReplicasPlugin
	preemptionFilterPlugins []framework.PreemptionFilterPlugin
	preemptionOrderPlugin   framework.PreemptionOrderPlugin
	clientSet               clientset.Interface
	informerFactory         informers.SharedInformerFactory
}

var _ framework.Framework = &frameworkImpl{}

type frameworkOptions struct {
	clientSet               clientset.Interface
	informerFactory         informers.SharedInformerFactory
	orderedSortingCriterias []framework.OrderedSortingCriteria
}

func WithDefaultSortingCriterias() Option {
	return func(o *frameworkOptions) {
		o.orderedSortingCriterias = []framework.OrderedSortingCriteria{preemptionorder.SortByPriority, preemptionorder.SortByWeightedResource}
	}
}

// Option for the frameworkImpl.
type Option func(*frameworkOptions)

func defaultFrameworkOptions() frameworkOptions {
	return frameworkOptions{}
}

// WithClientSet sets clientSet for the scheduling frameworkImpl.
func WithClientSet(clientSet clientset.Interface) Option {
	return func(o *frameworkOptions) {
		o.clientSet = clientSet
	}
}

// WithInformerFactory sets informer factory for the scheduling frameworkImpl.
func WithInformerFactory(informerFactory informers.SharedInformerFactory) Option {
	return func(o *frameworkOptions) {
		o.informerFactory = informerFactory
	}
}

// NewFramework creates a scheduling framework by registry.
func NewFramework(r Registry, opts ...Option) (framework.Framework, error) {
	options := defaultFrameworkOptions()
	for _, opt := range opts {
		opt(&options)
	}
	f := &frameworkImpl{
		informerFactory: options.informerFactory,
	}
	estimateReplicasPluginsList := reflect.ValueOf(&f.estimateReplicasPlugins).Elem()
	estimateReplicasType := estimateReplicasPluginsList.Type().Elem()
	preemptionFilterPluginsList := reflect.ValueOf(&f.preemptionFilterPlugins).Elem()
	preemptionFilterType := preemptionFilterPluginsList.Type().Elem()
	preemptionOrderPlugin := reflect.ValueOf(&f.preemptionOrderPlugin).Elem()
	preemptionOrderType := preemptionOrderPlugin.Type()

	for name, factory := range r {
		p, err := factory(f)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize plugin %q: %w", name, err)
		}
		addPluginToList(p, estimateReplicasType, &estimateReplicasPluginsList)
		addPluginToList(p, preemptionFilterType, &preemptionFilterPluginsList)
		if reflect.TypeOf(p).Implements(preemptionOrderType) {
			preemptionOrderPlugin.Set(reflect.ValueOf(p))
			f.preemptionOrderPlugin.SetSortingCriterias(options.orderedSortingCriterias)
		}
	}
	return f, nil
}

func addPluginToList(plugin framework.Plugin, pluginType reflect.Type, pluginList *reflect.Value) {
	if reflect.TypeOf(plugin).Implements(pluginType) {
		newPlugins := reflect.Append(*pluginList, reflect.ValueOf(plugin))
		pluginList.Set(newPlugins)
	}
}

// ClientSet returns a kubernetes clientset.
func (frw *frameworkImpl) ClientSet() clientset.Interface {
	return frw.clientSet
}

// SharedInformerFactory returns a shared informer factory.
func (frw *frameworkImpl) SharedInformerFactory() informers.SharedInformerFactory {
	return frw.informerFactory
}

// RunEstimateReplicasPlugins runs the set of configured EstimateReplicasPlugins
// for estimating replicas based on the given replicaRequirements.
// It returns an integer and an error.
// The integer represents the minimum calculated value of estimated replicas from each EstimateReplicasPlugin.
func (frw *frameworkImpl) RunEstimateReplicasPlugins(ctx context.Context, snapshot *schedcache.Snapshot, replicaRequirements *pb.ReplicaRequirements) (int32, *framework.Result) {
	startTime := time.Now()
	defer func() {
		metrics.FrameworkExtensionPointDuration.WithLabelValues(estimator).Observe(utilmetrics.DurationInSeconds(startTime))
	}()
	var replica int32 = math.MaxInt32
	results := make(framework.PluginToResult)
	for _, pl := range frw.estimateReplicasPlugins {
		plReplica, ret := frw.runEstimateReplicasPlugins(ctx, pl, snapshot, replicaRequirements)
		if (ret.IsSuccess() || ret.IsUnschedulable()) && plReplica < replica {
			replica = plReplica
		}
		results[pl.Name()] = ret
	}
	return replica, results.Merge()
}

func (frw *frameworkImpl) runEstimateReplicasPlugins(
	ctx context.Context,
	pl framework.EstimateReplicasPlugin,
	snapshot *schedcache.Snapshot,
	replicaRequirements *pb.ReplicaRequirements,
) (int32, *framework.Result) {
	startTime := time.Now()
	replica, ret := pl.Estimate(ctx, snapshot, replicaRequirements)
	metrics.PluginExecutionDuration.WithLabelValues(pl.Name(), estimator).Observe(utilmetrics.DurationInSeconds(startTime))
	return replica, ret
}

// RunPreemptionFilterPlugins runs the set of configured PreemptionFilterPlugins
// for filtering resourcebindings based on the given PreemptionRequest.
// It returns an integer and an error.
// The integer represents the minimum calculated value of estimated replicas from each EstimateReplicasPlugin.
func (frw *frameworkImpl) RunPreemptionFilterPlugins(ctx context.Context, candidateVictims *map[string]preemptee.CandidateVictim, preemptionRequest *pb.PreemptionRequest) (*map[string]preemptee.CandidateVictim, *framework.Result) {
	startTime := time.Now()
	defer func() {
		metrics.FrameworkExtensionPointDuration.WithLabelValues(estimator).Observe(utilmetrics.DurationInSeconds(startTime))
	}()
	results := make(framework.PluginToResult)
	for _, pl := range frw.preemptionFilterPlugins {
		ret := frw.runPreemptionFilterPlugins(ctx, pl, candidateVictims, preemptionRequest)
		results[pl.Name()] = ret
	}
	return candidateVictims, results.Merge()
}

func (frw *frameworkImpl) runPreemptionFilterPlugins(
	ctx context.Context,
	pl framework.PreemptionFilterPlugin,
	candidateVictims *map[string]preemptee.CandidateVictim,
	preemptionRequest *pb.PreemptionRequest,
) *framework.Result {
	startTime := time.Now()
	ret := pl.Filter(ctx, candidateVictims, preemptionRequest)
	metrics.PluginExecutionDuration.WithLabelValues(pl.Name(), estimator).Observe(utilmetrics.DurationInSeconds(startTime))
	return ret
}

// RunPreemptionFilterPlugins runs the set of configured PreemptionFilterPlugins
// for filtering resourcebindings based on the given PreemptionRequest.
// It returns an integer and an error.
// The integer represents the minimum calculated value of estimated replicas from each EstimateReplicasPlugin.
func (frw *frameworkImpl) RunPreemptionOrderPlugin(ctx context.Context, candidateVictims *map[string]preemptee.CandidateVictim, preemptionRequest *pb.PreemptionRequest) ([]*preemptee.CandidateVictim, *framework.Result) {
	startTime := time.Now()
	defer func() {
		metrics.FrameworkExtensionPointDuration.WithLabelValues(estimator).Observe(utilmetrics.DurationInSeconds(startTime))
	}()
	results := make(framework.PluginToResult)
	// Only one ordering plugin is supported for now
	pl := frw.preemptionOrderPlugin
	orderedCandidateVictims, ret := frw.runPreemptionOrderPlugin(ctx, pl, candidateVictims, preemptionRequest)
	results[pl.Name()] = ret
	return orderedCandidateVictims, results.Merge()
}

func (frw *frameworkImpl) runPreemptionOrderPlugin(
	ctx context.Context,
	pl framework.PreemptionOrderPlugin,
	candidateVictims *map[string]preemptee.CandidateVictim,
	preemptionRequest *pb.PreemptionRequest,
) ([]*preemptee.CandidateVictim, *framework.Result) {
	startTime := time.Now()
	orderedCandidateVictims, ret := pl.Order(ctx, candidateVictims, preemptionRequest)
	metrics.PluginExecutionDuration.WithLabelValues(pl.Name(), estimator).Observe(utilmetrics.DurationInSeconds(startTime))
	return orderedCandidateVictims, ret
}
