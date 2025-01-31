// Copyright 2022 Intel Corporation. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package balloons

import (
	"sort"
	"strconv"
	"strings"

	"github.com/containers/nri-plugins/pkg/resmgr/policy"
	"github.com/containers/nri-plugins/pkg/utils/cpuset"
	"github.com/prometheus/client_golang/prometheus"
)

// Prometheus Metric descriptor indices and descriptor table
const (
	balloonsDesc = iota
)

var descriptors = []*prometheus.Desc{
	balloonsDesc: prometheus.NewDesc(
		"balloons",
		"CPUs",
		[]string{
			"balloon_type",
			"cpu_class",
			"cpus_min",
			"cpus_max",
			"balloon",
			"groups",
			"cpus",
			"cpus_count",
			"numas",
			"numas_count",
			"dies",
			"dies_count",
			"packages",
			"packages_count",
			"sharedidlecpus",
			"sharedidlecpus_count",
			"cpus_allowed",
			"cpus_allowed_count",
			"mems",
			"containers",
			"tot_req_millicpu",
		}, nil,
	),
}

// Metrics defines the balloons-specific metrics from policy level.
type Metrics struct {
	Balloons []*BalloonMetrics
}

// BalloonMetrics define metrics of a balloon instance.
type BalloonMetrics struct {
	// Balloon type metrics
	DefName  string
	CpuClass string
	MinCpus  int
	MaxCpus  int
	// Balloon instance metrics
	PrettyName            string
	Groups                string
	Cpus                  cpuset.CPUSet
	CpusCount             int
	Numas                 []string
	NumasCount            int
	Dies                  []string
	DiesCount             int
	Packages              []string
	PackagesCount         int
	SharedIdleCpus        cpuset.CPUSet
	SharedIdleCpusCount   int
	CpusAllowed           cpuset.CPUSet
	CpusAllowedCount      int
	Mems                  string
	ContainerNames        string
	ContainerReqMilliCpus int
}

func (p *balloons) GetMetrics() policy.Metrics {
	policyMetrics := &Metrics{}
	policyMetrics.Balloons = make([]*BalloonMetrics, len(p.balloons))
	for index, bln := range p.balloons {
		cpuLoc := p.cpuTree.CpuLocations(bln.Cpus)
		bm := &BalloonMetrics{}
		policyMetrics.Balloons[index] = bm
		bm.DefName = bln.Def.Name
		bm.CpuClass = bln.Def.CpuClass
		bm.MinCpus = bln.Def.MinCpus
		bm.MaxCpus = bln.Def.MaxCpus
		bm.PrettyName = bln.PrettyName()
		groups := []string{}
		for group, cCount := range bln.Groups {
			if cCount > 0 {
				groups = append(groups, group)
			}
		}
		sort.Strings(groups)
		bm.Groups = strings.Join(groups, ",")
		bm.Cpus = bln.Cpus
		bm.CpusCount = bm.Cpus.Size()
		if len(cpuLoc) > 3 {
			bm.Numas = cpuLoc[3]
			bm.NumasCount = len(bm.Numas)
			bm.Dies = cpuLoc[2]
			bm.DiesCount = len(bm.Dies)
			bm.Packages = cpuLoc[1]
			bm.PackagesCount = len(bm.Packages)
		}
		bm.SharedIdleCpus = bln.SharedIdleCpus
		bm.SharedIdleCpusCount = bm.SharedIdleCpus.Size()
		bm.CpusAllowed = bm.Cpus.Union(bm.SharedIdleCpus)
		bm.CpusAllowedCount = bm.CpusAllowed.Size()
		bm.Mems = bln.Mems.String()
		cNames := []string{}
		// Get container names and total requested milliCPUs.
		for _, containerIDs := range bln.PodIDs {
			for _, containerID := range containerIDs {
				if c, ok := p.cch.LookupContainer(containerID); ok {
					cNames = append(cNames, c.PrettyName())
					bm.ContainerReqMilliCpus += p.containerRequestedMilliCpus(containerID)
				}
			}
		}
		sort.Strings(cNames)
		bm.ContainerNames = strings.Join(cNames, ",")
	}

	return policyMetrics
}

func (m *Metrics) Describe(ch chan<- *prometheus.Desc) {
	for _, d := range descriptors {
		ch <- d
	}
}

func (m *Metrics) Collect(ch chan<- prometheus.Metric) {
	if m == nil {
		return
	}

	for _, bm := range m.Balloons {
		ch <- prometheus.MustNewConstMetric(
			descriptors[balloonsDesc],
			prometheus.GaugeValue,
			float64(bm.Cpus.Size()),
			bm.DefName,
			bm.CpuClass,
			strconv.Itoa(bm.MinCpus),
			strconv.Itoa(bm.MaxCpus),
			bm.PrettyName,
			bm.Groups,
			bm.Cpus.String(),
			strconv.Itoa(bm.CpusCount),
			strings.Join(bm.Numas, ","),
			strconv.Itoa(bm.NumasCount),
			strings.Join(bm.Dies, ","),
			strconv.Itoa(bm.DiesCount),
			strings.Join(bm.Packages, ","),
			strconv.Itoa(bm.PackagesCount),
			bm.SharedIdleCpus.String(),
			strconv.Itoa(bm.SharedIdleCpusCount),
			bm.CpusAllowed.String(),
			strconv.Itoa(bm.CpusAllowedCount),
			bm.Mems,
			bm.ContainerNames,
			strconv.Itoa(bm.ContainerReqMilliCpus))
	}
}
