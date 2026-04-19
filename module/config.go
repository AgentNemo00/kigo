package module

import "github.com/AgentNemo00/kigo-core/order"

type Config struct {
	Name 			string
	PubSubURL 		string
	RenderTo   		string
	UIConfiguration order.UIConfiguration
}