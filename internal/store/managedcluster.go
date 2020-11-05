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
	"time"

	clusterv1client "github.com/open-cluster-management/api/client/cluster/clientset/versioned"
	clusterv1informers "github.com/open-cluster-management/api/client/cluster/informers/externalversions"
	clusterv1 "github.com/open-cluster-management/api/cluster/v1"
	libgoconfig "github.com/open-cluster-management/library-go/pkg/config"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"k8s.io/kube-state-metrics/v2/pkg/metric"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

var (
	mcGVR = schema.GroupVersionResource{
		Group:    "cluster.open-cluster-management.io",
		Version:  "v1",
		Resource: "managedclusters",
	}

	cdGVR = schema.GroupVersionResource{
		Group:    "hive.openshift.io",
		Version:  "v1",
		Resource: "clusterdeployments",
	}
	descManagedClusterLabelsName          = "acm_managedcluster_labels"
	descManagedClusterLabelsHelp          = "ACM labels converted to Prometheus labels."
	descManagedClusterLabelsDefaultLabels = []string{"hub_name", "vendor", "cloud", "created_via", "version"}
	managedClusterMetricFamilies          = []generator.FamilyGenerator{
		*generator.NewFamilyGenerator(
			"acm_managedcluster_created",
			"Unix creation timestamp",
			metric.Gauge,
			"",
			wrapManagedClusterFunc(func(mc *clusterv1.ManagedCluster) *metric.Family {
				ms := []*metric.Metric{}

				ms = append(ms, &metric.Metric{
					Value: float64(1),
				})

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
	}
)

func wrapManagedClusterFunc(f func(*clusterv1.ManagedCluster) *metric.Family) func(interface{}) *metric.Family {
	klog.Info("start wrapManagedClusterFunc")
	klog.Info("start createManagedClusterListWatch")
	config, err := libgoconfig.LoadConfig("", "", "")
	if err != nil {
		klog.Errorf("Error: %s", err)
		return nil
	}
	clientset, err := dynamic.NewForConfig(config)
	if err != nil {
		klog.Errorf("Error: %s", err)
		return nil
	}

	return func(obj interface{}) *metric.Family {
		mc := obj.(*clusterv1.ManagedCluster)
		metricFamily := f(mc)
		createdVia := "hive"
		_, err = clientset.Resource(cdGVR).Namespace(mc.GetName()).Get(context.TODO(), mc.GetName(), metav1.GetOptions{})
		if errors.IsNotFound(err) {
			createdVia = "imported"
		}

		for _, m := range metricFamily.Metrics {
			m.LabelKeys = append(descManagedClusterLabelsDefaultLabels, m.LabelKeys...)
			mcLabels := mc.GetLabels()
			mcVendor := ""
			mcCloud := ""

			if v, ok := mcLabels["vendor"]; ok {
				mcVendor = v
			}
			if v, ok := mcLabels["cloud"]; ok {
				mcCloud = v
			}
			m.LabelValues = append([]string{mc.Name, mcVendor, mcCloud, createdVia, mc.Status.Version.Kubernetes}, m.LabelValues...)
		}

		return metricFamily
	}
}

func createManagedClusterListWatch(kubeClient clientset.Interface, ns string) cache.ListerWatcher {
	klog.Info("start createManagedClusterListWatch")
	config, err := libgoconfig.LoadConfig("", "", "")
	if err != nil {
		klog.Errorf("Error: %s", err)
		return nil
	}
	clusterClient, err := clusterv1client.NewForConfig(config)
	if err != nil {
		klog.Errorf("Error: %s", err)
		return nil
	}
	clusterInformers := clusterv1informers.NewSharedInformerFactory(clusterClient, 10*time.Second)
	go clusterInformers.Start(context.TODO().Done())

	// clientset, err := dynamic.NewForConfig(config)
	// if err != nil {
	// 	klog.Errorf("Error: %s", err)
	// 	return nil
	// }

	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			klog.Info("ListFunc ManagedCluster")
			mcList := &clusterv1.ManagedClusterList{}
			ls := labels.NewSelector()
			mcs, err := clusterInformers.Cluster().V1().ManagedClusters().Lister().List(ls)
			if err != nil {
				klog.Errorf("Error: %s", err)
				return nil, err
			}
			for i, _ := range mcs {
				// createdVia := "hive"
				// _, err = clientset.Resource(cdGVR).Namespace(mc.GetName()).Get(context.TODO(), mc.GetName(), metav1.GetOptions{})
				// if errors.IsNotFound(err) {
				// 	createdVia = "imported"
				// }
				// labels := mcs[i].GetLabels()
				// if labels == nil {
				// 	labels = make(map[string]string)
				// }
				// labels["created_via"] = createdVia
				// mcs[i].SetLabels(labels)
				mcList.Items = append(mcList.Items, *mcs[i])
			}
			klog.Infof("mc # %d\n", len(mcList.Items))
			return mcList, err
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			klog.Info("WatchFunc ManagedCluster")
			return clusterClient.ClusterV1().ManagedClusters().Watch(context.TODO(), opts)
		},
	}
}
