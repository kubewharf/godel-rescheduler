// Copyright 2024 The Godel Rescheduler Authors
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

package internal

import "fmt"

type sccImpl struct {
	dfn, low           []int
	id, size           []int
	timestamp, scc_cnt int
}

var _ SCC = &sccImpl{}

func newSCC(g Graph) *sccImpl {
	return &sccImpl{
		dfn:       make([]int, g.NumNode()),
		low:       make([]int, g.NumNode()),
		id:        make([]int, g.NumNode()),
		size:      make([]int, g.NumNode()),
		timestamp: 0,
		scc_cnt:   0,
	}
}

func (scc *sccImpl) GetID(x int) int {
	return scc.id[x]
}

func (scc *sccImpl) GetSccCnt() int {
	return scc.scc_cnt
}

func (scc *sccImpl) String() string {
	var nodes, sccsize string
	for i := 0; i < len(scc.dfn); i++ {
		// nodes += fmt.Sprintf("{idx:%v,dfn:%v,low:%v,id:%v}", i, scc.dfn[i], scc.low[i], scc.id[i])
		nodes += fmt.Sprintf("{idx:%v,id:%v},", i, scc.id[i])
	}
	for i := 0; i < int(scc.scc_cnt); i++ {
		sccsize += fmt.Sprintf("{sccid:%v,size:%v},", i, scc.size[i])
	}
	return fmt.Sprintf("[%v,%v]", nodes, sccsize)
}

func SCCTarjan(g Graph) SCC {
	scc := newSCC(g)

	stk := make([]int, g.NumNode()+1)
	in_stk := make([]bool, g.NumNode()+1)
	top := 0

	var tarjan func(u int)
	tarjan = func(u int) {
		{
			scc.timestamp++
			scc.dfn[u], scc.low[u] = scc.timestamp, scc.timestamp
		}
		{
			top++
			stk[top], in_stk[u] = u, true
		}
		for i := g.Head(u); i != -1; i = g.Next(i) {
			j := g.Edge(i)
			if scc.dfn[j] == 0 {
				tarjan(j)
				if scc.low[j] < scc.low[u] {
					scc.low[u] = scc.low[j]
				}
			} else if in_stk[j] {
				if scc.dfn[j] < scc.low[u] {
					scc.low[u] = scc.dfn[j]
				}
			}
		}
		if scc.dfn[u] == scc.low[u] {
			for {
				y := stk[top]
				top--
				in_stk[y] = false
				scc.id[y] = scc.scc_cnt
				scc.size[scc.scc_cnt]++
				if y == u {
					break
				}
			}
			scc.scc_cnt++
		}
	}

	for i := 0; i < g.NumNode(); i++ {
		if scc.dfn[i] == 0 {
			tarjan(i)
		}
	}
	return scc
}
