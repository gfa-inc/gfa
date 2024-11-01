package casbinx

import "time"

const TableNameSysCasbinRule = "sys_casbin_rule"

// SysCasbinRule 系统 Casbin 规则表
type SysCasbinRule struct {
	ID         int64      `gorm:"column:id;primaryKey;autoIncrement:true;comment:规则ID" json:"id,omitempty"`                                       // 规则ID
	Ptype      string     `gorm:"column:ptype;not null;comment:策略类型" json:"ptype,omitempty"`                                                      // 策略类型
	V0         string     `gorm:"column:v0;not null;comment:v0" json:"v0,omitempty"`                                                              // v0
	V1         string     `gorm:"column:v1;not null;comment:v1" json:"v1,omitempty"`                                                              // v1
	V2         string     `gorm:"column:v2;not null;comment:v2" json:"v2,omitempty"`                                                              // v2
	V3         string     `gorm:"column:v3;not null;comment:v3" json:"v3,omitempty"`                                                              // v3
	V4         string     `gorm:"column:v4;not null;comment:v4" json:"v4,omitempty"`                                                              // v4
	V5         string     `gorm:"column:v5;not null;comment:v5" json:"v5,omitempty"`                                                              // v5
	CreateTime *time.Time `gorm:"column:create_time;not null;default:CURRENT_TIMESTAMP;autoCreateTime;comment:创建时间" json:"create_time,omitempty"` // 创建时间
	UpdateTime *time.Time `gorm:"column:update_time;not null;default:CURRENT_TIMESTAMP;autoUpdateTime;comment:更新时间" json:"update_time,omitempty"` // 更新时间
}

// TableName SysCasbinRule's table name
func (SysCasbinRule) TableName() string {
	return TableNameSysCasbinRule
}
