package casbinx

import (
	"encoding/csv"
	"strings"
	"testing"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/util"
	"github.com/gfa-inc/gfa/common/config"
	"github.com/gfa-inc/gfa/common/db"
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/gfa-inc/gfa/resources"
	"github.com/spf13/cast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetup(t *testing.T) {
	config.Setup(config.WithPath("../../../.."))
	logger.Setup()
	db.Setup()

	Setup()

	_, err := Enforcer.HasRoleForUser(cast.ToString(1), "admin")
	assert.Nil(t, err)
}

// TestCasbinModelWithCSV 使用 CSV 数据验证 casbin 模型配置
func TestCasbinModelWithCSV(t *testing.T) {
	// CSV 格式：策略类型,v0,v1,v2,v3,v4,v5
	policiesCSV := `p,user1,/api/users,GET
p,user1,/api/users,POST
p,admin,/api/admin,GET
p,admin,/api/admin,POST
p,admin,/api/admin,DELETE
p,user2,/api/data,
g,user1,user1
g,user2,user2
g,admin,admin`

	// CSV 格式：主体,对象,操作,预期结果
	testCasesCSV := `user1,/api/users,GET,true
user1,/api/users,POST,true
user1,/api/users,DELETE,false
user1,/api/admin,GET,false
user2,/api/data,GET,true
user2,/api/data,POST,true
user2,/api/data,DELETE,true
user2,/api/users,GET,false
admin,/api/admin,GET,true
admin,/api/admin,POST,true
admin,/api/admin,DELETE,true
admin,/api/users,GET,true
admin,/api/anything,ANY,true`

	// 创建模型
	m, err := model.NewModelFromString(string(resources.CasbinModelConf))
	require.NoError(t, err, "加载 casbin 模型失败")

	// 创建 enforcer（使用内存适配器）
	e, err := casbin.NewEnforcer(m)
	require.NoError(t, err, "创建 enforcer 失败")

	// 添加 KeyMatch2 匹配函数
	e.AddNamedMatchingFunc("g", "KeyMatch2", util.KeyMatch2)

	// 解析并加载策略数据
	err = loadPoliciesFromCSV(e, policiesCSV)
	require.NoError(t, err, "加载策略数据失败")

	// 解析并运行测试用例
	testCases, err := parseTestCasesFromCSV(testCasesCSV)
	require.NoError(t, err, "解析测试用例失败")

	for i, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := e.Enforce(tc.sub, tc.obj, tc.act)
			assert.NoError(t, err, "测试用例 %d 执行失败", i+1)
			assert.Equal(t, tc.expected, result,
				"测试用例 %d 失败: sub=%s, obj=%s, act=%s, 期望=%v, 实际=%v",
				i+1, tc.sub, tc.obj, tc.act, tc.expected, result)
		})
	}
}

// loadPoliciesFromCSV 从 CSV 字符串加载策略数据
func loadPoliciesFromCSV(e *casbin.Enforcer, csvData string) error {
	reader := csv.NewReader(strings.NewReader(csvData))
	reader.FieldsPerRecord = -1 // 允许可变数量的字段
	records, err := reader.ReadAll()
	if err != nil {
		return err
	}

	for _, record := range records {
		if len(record) == 0 {
			continue
		}

		ptype := strings.TrimSpace(record[0])
		if ptype == "" {
			continue
		}

		// 准备参数
		params := make([]interface{}, 0, len(record)-1)
		for i := 1; i < len(record); i++ {
			params = append(params, strings.TrimSpace(record[i]))
		}

		// 根据类型添加策略
		switch ptype {
		case "p":
			if len(params) >= 2 {
				_, err := e.AddPolicy(params...)
				if err != nil {
					return err
				}
			}
		case "g":
			if len(params) >= 2 {
				_, err := e.AddGroupingPolicy(params...)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// testCase 测试用例结构
type testCase struct {
	name     string
	sub      string // 主体
	obj      string // 对象
	act      string // 操作
	expected bool   // 预期结果
}

// parseTestCasesFromCSV 从 CSV 字符串解析测试用例
func parseTestCasesFromCSV(csvData string) ([]testCase, error) {
	reader := csv.NewReader(strings.NewReader(csvData))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	testCases := make([]testCase, 0, len(records))
	for _, record := range records {
		if len(record) < 4 {
			continue
		}

		sub := strings.TrimSpace(record[0])
		obj := strings.TrimSpace(record[1])
		act := strings.TrimSpace(record[2])
		expected := strings.TrimSpace(record[3]) == "true"

		testCases = append(testCases, testCase{
			name:     sub + "_" + obj + "_" + act,
			sub:      sub,
			obj:      obj,
			act:      act,
			expected: expected,
		})
	}

	return testCases, nil
}
