package module

import (
	"context"
	"sync"

	"github.com/AgentNemo00/sca-instruments/database"
	"github.com/AgentNemo00/sca-instruments/database/lectors"
	"gorm.io/gorm"
)
type Database struct {
	instance 		*database.Database
	mu 					sync.RWMutex
}

func NewDatabase(connection string) (*Database, error) {
	db, err := database.WithLector(lectors.SqliteByPath(connection))
	if err != nil {
		return nil, err
	}
	err = db.ApplySchemas(&Module{})
	if err != nil {
		return nil, err
	}
	return &Database{
		instance: db,
	}, nil
}

func (d *Database) FindModulesDB(ctx context.Context) ([]*Module, error) {
	modules, err := gorm.G[Module](d.instance.DB).Find(ctx)
	if err != nil {
		return nil, err
	}
	ret := make([]*Module, 0)
	for _, mod := range modules {
		ret = append(ret, &mod)
	}
	return ret, nil
}

func (d *Database) SaveModulesDB(ctx context.Context, modules []*Module) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, mod := range modules {
		if mod.ID == 0 {
			err := gorm.G[Module](d.instance.DB).Create(ctx, mod)
			if err != nil {
				return err
			}
			continue
		}
		_, err := gorm.G[Module](d.instance.DB).Updates(ctx, *mod)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *Database) DeleteModuleDB(ctx context.Context, mod *Module) error {
	if mod.ID <= 0 {
		return nil
	}
	_, err := gorm.G[Module](d.instance.DB).Where("id = ?", mod.ID).Delete(ctx)
	return err
}

func (d *Database) SaveModuleDB(ctx context.Context, mod *Module) error {
	_, err :=  gorm.G[Module](d.instance.DB).Where("id = ?", mod.ID).Updates(ctx, *mod)
	return err
}

func (d *Database) CreateModuleDB(ctx context.Context, mod *Module) error {
	return gorm.G[Module](d.instance.DB).Create(ctx, mod)
}
