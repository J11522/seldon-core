package pipeline

import (
	"errors"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/seldonio/seldon-core/scheduler/apis/mlops/scheduler"
)

func TestCreatePipelineFromProto(t *testing.T) {
	g := NewGomegaWithT(t)
	type test struct {
		name     string
		proto    *scheduler.Pipeline
		version  uint32
		pipeline *PipelineVersion
		err      error
	}

	tests := []test{
		{
			name:    "simple",
			version: 1,
			proto: &scheduler.Pipeline{
				Name: "pipeline",
				Steps: []*scheduler.PipelineStep{
					{
						Name:   "a",
						Inputs: []string{},
					},
					{
						Name:   "b",
						Inputs: []string{"a"},
					},
				},
				Output: &scheduler.PipelineOutput{
					Inputs: []string{"b"},
				},
			},
			pipeline: &PipelineVersion{
				Name:    "pipeline",
				Version: 1,
				Steps: map[string]*PipelineStep{
					"a": {
						Name:   "a",
						Inputs: []string{},
					},
					"b": {
						Name:   "b",
						Inputs: []string{"a"},
					},
				},
				Output: &PipelineOutput{
					Inputs: []string{"b"},
				},
				State: &PipelineState{},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pv, err := CreatePipelineFromProto(test.proto, test.version)
			if test.err == nil {
				pv.UID = ""
				g.Expect(pv.Name).To(Equal(test.pipeline.Name))
				for k, v := range pv.Steps {
					g.Expect(test.pipeline.Steps[k]).To(Equal(v))
				}
			} else {
				g.Expect(errors.Is(err, test.err)).To(BeTrue())
			}
		})
	}

}
