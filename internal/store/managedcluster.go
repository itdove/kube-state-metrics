/*
Copyright 2016 The Kubernetes Authors All rights reserved.

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

package store

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kube-state-metrics/v2/pkg/metric"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"

	clusterv1 "github.com/open-cluster-management/api/cluster/v1"
	"k8s.io/client-go/rest"
)

var (
	mcGVR = schema.GroupVersionResource{
		Group:    "cluster.open-cluster-management.io",
		Version:  "v1",
		Resource: "managedclusters",
	}

	descManagedClusterLabelsName          = "acm_managedcluster_labels"
	descManagedClusterLabelsHelp          = "ACM labels converted to Prometheus labels."
	descManagedClusterLabelsDefaultLabels = []string{"hub_name"}
	managedClusterMetricFamilies          = []generator.FamilyGenerator{
		*generator.NewFamilyGenerator(
			"acm_managedcluster_created",
			"Unix creation timestamp",
			metric.Gauge,
			"",
			wrapManagedClusterFunc(func(d *clusterv1.ManagedCluster) *metric.Family {
				ms := []*metric.Metric{}

				if !d.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{
						Value: float64(d.CreationTimestamp.Unix()),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
	}
)

func wrapManagedClusterFunc(f func(*clusterv1.ManagedCluster) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		mc := obj.(*clusterv1.ManagedCluster)

		metricFamily := f(mc)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys = append(descManagedClusterLabelsDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{mc.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}

func createManagedClusterListWatch(kubeClient clientset.Interface, ns string) cache.ListerWatcher {

	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			config, err := rest.InClusterConfig()
			if err != nil {
				return nil, err
			}
			clientset, err := dynamic.NewForConfig(config)
			if err != nil {
				return nil, err
			}
			u, err := clientset.Resource(mcGVR).List(context.TODO(), opts)
			if err != nil {
				return nil, err
			}
			mcList := &clusterv1.ManagedClusterList{}
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &mcList); err != nil {
				return nil, err
			}
			return mcList, err
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			config, err := rest.InClusterConfig()
			if err != nil {
				return nil, err
			}
			clientset, err := dynamic.NewForConfig(config)
			if err != nil {
				return nil, err
			}
			return clientset.Resource(mcGVR).Watch(context.TODO(), opts)
		},
	}
}
