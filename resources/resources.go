package resources

import (
	_ "embed"
)

//go:embed casbin/model.conf
var CasbinModelConf []byte
