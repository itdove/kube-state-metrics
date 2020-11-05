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
	"testing"
	"time"

	clusterv1 "github.com/open-cluster-management/api/cluster/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

var (
	mc1Replicas int32 = 200
	mc2Replicas int32 = 5

	mc1MaxUnavailable = intstr.FromInt(10)
	mc2MaxUnavailable = intstr.FromString("25%")

	mc1MaxSurge = intstr.FromInt(10)
	mc2MaxSurge = intstr.FromString("20%")
)

func TestManagedClusterStore(t *testing.T) {
	// Fixed metadata on type and help text. We prepend this to every expected
	// output so we only have to modify a single place when doing adjustments.
	const metadata = `
		# HELP acm_managedcluster_created Unix creation timestamp
		# TYPE acm_managedcluster_created gauge
	`
	cases := []generateMetricsTestCase{
		{
			Obj: &clusterv1.ManagedCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "mc1",
					CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
					Generation:        21,
				},
			},
			Want: metadata + `
        acm_managedcluster_created{hub_name="mc1"} 1.5e+09
`,
		},
	}

	for i, c := range cases {
		c.Func = generator.ComposeMetricGenFuncs(managedClusterMetricFamilies)
		c.Headers = generator.ExtractMetricFamilyHeaders(managedClusterMetricFamilies)
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
		}
	}
}
