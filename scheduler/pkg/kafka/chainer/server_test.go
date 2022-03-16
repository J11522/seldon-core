package chainer

import (
	"testing"

	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
)

func TestCreateTopicSources(t *testing.T) {
	g := NewGomegaWithT(t)

	type test struct {
		name         string
		server       *ChainerServer
		pipelineName string
		inputs       []string
		sources      []string
	}

	tests := []test{
		{
			name: "misc inputs",
			server: &ChainerServer{
				logger:    log.New(),
				namespace: "ns1",
			},
			pipelineName: "p1",
			inputs: []string{
				"a",
				"b.inputs",
				"c.inputs.t1",
			},
			sources: []string{
				"seldon.ns1.model.a",
				"seldon.ns1.model.b.inputs",
				"seldon.ns1.model.c.inputs.t1",
			},
		},
		{
			name: "misc inputs",
			server: &ChainerServer{
				logger:    log.New(),
				namespace: "ns1",
			},
			pipelineName: "p1",
			inputs:       []string{},
			sources: []string{
				"seldon.ns1.pipeline.p1.inputs",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sources := test.server.createTopicSources(test.inputs, test.pipelineName)
			g.Expect(sources).To(Equal(test.sources))
		})
	}
}
