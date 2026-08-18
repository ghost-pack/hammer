package goservice

import (
	"context"
	"testing"

	"github.com/ghost-pack/hammer/internal/oam"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestPipeline_build(t *testing.T) {
	t.Run("fails when component is unparseable", func(t *testing.T) {
		p := &Pipeline{
			component: &oam.Component{
				Name: "testComponent",
				Type: "goservice",
				Properties: yaml.Node{
					Kind: yaml.MappingNode,
					Tag:  "!!map",
					Content: []*yaml.Node{
						{Kind: yaml.ScalarNode, Tag: "!!str", Value: "path"},
						{
							Kind: yaml.SequenceNode,
							Tag:  "!!seq",
							Content: []*yaml.Node{
								{Kind: yaml.ScalarNode, Tag: "!!str", Value: "item"},
							},
						},
					},
				},
			},
		}

		err := p.build(context.Background())
		require.Error(t, err)
	})
}
