package gormx

import (
	"fmt"
	"github.com/gfa-inc/gfa/core"
	"gorm.io/gen/field"
	"time"
)

type Pagination struct {
	Current  *int `form:"current,default=1"`
	PageSize *int `form:"pageSize,default=20"`
}

func (p *Pagination) ToOffsetAndLimit() (int, int) {
	offset := (*p.Current - 1) * (*p.PageSize)
	limit := *p.PageSize

	p.Current = nil
	p.PageSize = nil

	return offset, limit
}

type Dao interface {
	TableName() string
	GetFieldByName(fieldName string) (field.OrderExpr, bool)
}

type Sorter struct {
	Field string `form:"sorterField,default=create_time"`
	Order string `form:"sorterOrder"`
}

func (s *Sorter) ToExpr(dao Dao) (field.Expr, error) {
	f, ok := dao.GetFieldByName(s.Field)
	if !ok {
		err := core.NewParamErr(fmt.Errorf("field %s not found in table %s", s.Field, dao.TableName()))
		return nil, err
	}
	s.Field = ""

	order := s.Order
	s.Order = ""

	if order == "ascend" {
		return f.Asc(), nil
	} else if order == "descend" {
		return f.Desc(), nil
	} else {
		return f.Desc(), nil
	}
}

type DateTimeRange struct {
	StartTime string `form:"start_time" binding:"omitempty,datetime=2006-01-02 15:04:05"`
	EndTime   string `form:"end_time" binding:"omitempty,datetime=2006-01-02 15:04:05"`
}

func (s *DateTimeRange) ToExpr(field field.Time) field.Expr {
	if s.StartTime == "" || s.EndTime == "" {
		return nil
	}

	startTime, _ := time.Parse(time.DateTime, s.StartTime)
	endTime, _ := time.Parse(time.DateTime, s.EndTime)

	s.StartTime = ""
	s.EndTime = ""
	return field.Between(startTime, endTime)
}
