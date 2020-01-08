package claimstore

import (
	"net/rpc"

	"github.com/cnabio/cnab-go/utils/crud"
	"github.com/hashicorp/go-plugin"
)

// PluginInterface for the instance-store interface.
const PluginInterface = "instance-store"

var _ plugin.Plugin = &Plugin{}

// Plugin is a generic type of plugin for working with any implementation of a claim store.
type Plugin struct {
	Impl crud.Store
}

func (p *Plugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &Server{Impl: p.Impl}, nil
}

func (Plugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &Client{client: c}, nil
}
