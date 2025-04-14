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

package testdata

import "github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/policycontroller/generator/relation"

type TestData struct {
	Relations relation.Relations
}

var TestDataSet = []TestData{
	{
		relation.Relations{
			relation.MakeRelation().Name("dp1/pod1").From("A").To("B").Obj(),
			relation.MakeRelation().Name("dp1/pod2").From("B").To("C").Obj(),
		},
	},
	{
		relation.Relations{
			relation.MakeRelation().Name("dp1/pod1").From("A").To("B").Obj(),
			relation.MakeRelation().Name("dp2/pod1").From("B").To("C").Obj(),
			relation.MakeRelation().Name("dp1/pod2").From("C").To("D").Obj(),
			relation.MakeRelation().Name("dp2/pod2").From("D").To("E").Obj(),
			relation.MakeRelation().Name("dp2/pod3").From("E").To("B").Obj(),
			relation.MakeRelation().Name("dp3/pod1").From("B").To("F").Obj(),
			relation.MakeRelation().Name("dp1/pod3").From("C").To("F").Obj(),
			relation.MakeRelation().Name("dp1/pod4").From("F").To("G").Obj(),
			relation.MakeRelation().Name("dp3/pod2").From("F").To("H").Obj(),
			relation.MakeRelation().Name("dp1/pod5").From("G").To("H").Obj(),
			relation.MakeRelation().Name("dp2/pod4").From("G").To("I").Obj(),
		},
	},
	{
		// {idx:0,id:4},{idx:1,id:3},{idx:2,id:2},{idx:3,id:1}
		// {idx:4,id:8},{idx:5,id:7},{idx:6,id:9},{idx:7,id:6},{idx:8,id:5},
		relation.Relations{
			// SccNode 1
			relation.MakeRelation().Name("dp1/pod1").From("A").To("B").Obj(),
			relation.MakeRelation().Name("dp2/pod1").From("B").To("C").Obj(),
			relation.MakeRelation().Name("dp1/pod2").From("C").To("D").Obj(),
			relation.MakeRelation().Name("dp2/pod2").From("A").To("D").Obj(),
			// SccNode 2
			relation.MakeRelation().Name("dp3/pod1").From("E").To("G").Obj(),
			relation.MakeRelation().Name("dp4/pod1").From("F").To("G").Obj(),
			relation.MakeRelation().Name("dp3/pod2").From("F").To("H").Obj(),
			relation.MakeRelation().Name("dp4/pod2").From("G").To("H").Obj(),
			relation.MakeRelation().Name("dp3/pod3").From("H").To("I").Obj(),
		},
	},
	{
		relation.Relations{
			relation.MakeRelation().Name("dp1/pod1").From("A").To("A").Obj(),
			relation.MakeRelation().Name("dp2/pod1").From("B").To("B").Obj(),
			relation.MakeRelation().Name("dp1/pod2").From("A").To("B").Obj(),
			relation.MakeRelation().Name("dp2/pod2").From("B").To("A").Obj(),
		},
	},
}

func SelectTestData(i int) TestData {
	return TestDataSet[i]
}
