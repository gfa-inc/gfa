package casbinx

import (
	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/util"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/gfa-inc/gfa/common/db/mysqlx"
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/gfa-inc/gfa/resources"
	"gorm.io/gorm"
)

type CasbinRuleTable interface {
	TableName() string
}

var Enforcer *casbin.Enforcer

func Setup() {
	var err error
	Enforcer, err = NewEnforcer(mysqlx.Client, SysCasbinRule{}, string(resources.CasbinModelConf))

	if err != nil {
		logger.Panic(err)
		return
	}
}

func NewEnforcer(db *gorm.DB, tb CasbinRuleTable, mdl string) (*casbin.Enforcer, error) {
	gormadapter.TurnOffAutoMigrate(db)

	adapter, err := gormadapter.NewAdapterByDBWithCustomTable(db, tb, tb.TableName())
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	m, err := model.NewModelFromString(mdl)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	enforcer, err := casbin.NewEnforcer(m, adapter)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	enforcer.AddNamedMatchingFunc("g", "KeyMatch2", util.KeyMatch2)
	return enforcer, nil
}
