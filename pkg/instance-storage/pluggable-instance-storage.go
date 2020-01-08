package instancestorage

import (
	"get.porter.sh/porter/pkg/config"
	"get.porter.sh/porter/pkg/instance-storage/claimstore"
	"get.porter.sh/porter/pkg/plugins/pluggable"
	"github.com/cnabio/cnab-go/claim"
	"github.com/cnabio/cnab-go/utils/crud"
	"github.com/pkg/errors"
)

var _ StorageProvider = &PluggableInstanceStorage{}

// A sad hack because claim.Store has a method called Store which prevents us from embedding it as a field
type ClaimStore = claim.Store

// PluggableInstanceStorage provides access to instance storage (claims) by instantiating plugins that
// implement claim (CRUD) storage.
type PluggableInstanceStorage struct {
	*config.Config
	ClaimStore
}

func NewPluggableInstanceStorage(c *config.Config) *PluggableInstanceStorage {
	l := &PluggableInstanceStorage{
		Config: c,
	}

	crud := NewDynamicCrudStore(l.connect)
	// this is silly, we can't embed Store because it has a method named Store...
	l.ClaimStore = claim.NewClaimStore(crud)

	return l
}

// NewPluginTypeConfig for instance storage.
func NewPluginTypeConfig() pluggable.PluginTypeConfig {
	return pluggable.PluginTypeConfig{
		Interface: claimstore.PluginInterface,
		Plugin:    &claimstore.Plugin{},
		GetDefaultPluggable: func(datastore *config.Data) string {
			return datastore.GetDefaultInstanceStore()
		},
		GetPluggable: func(datastore *config.Data, name string) (pluggable.Entry, error) {
			return datastore.GetInstanceStore(name)
		},
		GetDefaultPlugin: func(datastore *config.Data) string {
			return datastore.GetInstanceStoragePlugin()
		},
	}
}

func (d *PluggableInstanceStorage) connect() (crud.Store, func(), error) {
	pluginType := NewPluginTypeConfig()

	l := pluggable.NewPluginLoader(d.Config)
	raw, cleanup, err := l.Load(pluginType)
	if err != nil {
		return nil, nil, err
	}

	store, ok := raw.(crud.Store)
	if !ok {
		cleanup()
		return nil, nil, errors.Errorf("the interface exposed by the %s plugin was not instance-store.ClaimStore", l.SelectedPluginKey)
	}

	return store, cleanup, nil
}
