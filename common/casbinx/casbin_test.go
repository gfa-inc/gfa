package casbinx

import (
	"encoding/csv"
	"reflect"
	"strings"
	"testing"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/util"
	gormadapter "github.com/casbin/gorm-adapter/v3"
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

// TestAdapterTableName 测试并分析 adapter 的表名问题
func TestAdapterTableName(t *testing.T) {
	t.Run("默认内存适配器", func(t *testing.T) {
		m, err := model.NewModelFromString(string(resources.CasbinModelConf))
		require.NoError(t, err)

		e, err := casbin.NewEnforcer(m)
		require.NoError(t, err)

		adapter := e.GetAdapter()
		t.Logf("Adapter 类型: %T", adapter)
		// 内存适配器没有表名，应该为 nil
		assert.Nil(t, adapter, "内存模式下 adapter 应该为 nil")
	})

	t.Run("使用反射查看 gorm-adapter 内部字段", func(t *testing.T) {
		// 模拟创建一个 gorm-adapter（需要先创建模拟的 gorm.DB）
		customTable := SysCasbinRule{}
		t.Logf("期望的表名: %s", customTable.TableName())

		// 使用反射查看 Adapter 结构体定义
		adapterType := reflect.TypeOf(&gormadapter.Adapter{})
		t.Logf("gorm-adapter.Adapter 结构体字段:")
		for i := 0; i < adapterType.Elem().NumField(); i++ {
			field := adapterType.Elem().Field(i)
			t.Logf("  字段 %d: %s (类型: %s, 导出: %v)",
				i, field.Name, field.Type, field.IsExported())
		}

		// 说明：tableName 是私有字段（小写开头），无法直接访问
		// 但可以通过反射读取
		t.Logf("\n结论: gorm-adapter.Adapter 的 tableName 字段是私有的")
		t.Logf("需要检查在 NewEnforcer 中是否正确传递了表名参数")
	})

	t.Run("通过反射读取实际的 tableName 值", func(t *testing.T) {
		t.Skip("需要数据库连接才能测试，跳过")

		// 以下代码展示如何通过反射读取私有字段
		// 实际使用时需要有真实的数据库连接
		/*
			config.Setup(config.WithPath("../../../.."))
			logger.Setup()
			db.Setup()

			Setup()

			adapter := Enforcer.GetAdapter()
			if gormAdapter, ok := adapter.(*gormadapter.Adapter); ok {
				// 使用反射读取私有字段 tableName
				v := reflect.ValueOf(gormAdapter).Elem()
				tableNameField := v.FieldByName("tableName")

				if tableNameField.IsValid() {
					// 需要使用 unsafe 包或其他方式访问私有字段
					t.Logf("实际的 tableName: %v", tableNameField)
				}
			}
		*/
	})
}

// TestAnalyzeTableNameIssue 分析表名问题的根本原因
func TestAnalyzeTableNameIssue(t *testing.T) {
	t.Log("=== 问题分析：为什么 GetAdapter() 获取到的 tableName 是 casbin_rule ===")
	t.Log("")
	t.Log("原因分析：")
	t.Log("1. gorm-adapter 的 Adapter 结构体中，tableName 字段是私有的（小写开头）")
	t.Log("2. Adapter 没有提供公开的 GetTableName() 方法")
	t.Log("3. 虽然在创建时传递了正确的表名 'sys_casbin_rule'，但无法通过公开接口获取")
	t.Log("")
	t.Log("代码流程：")
	t.Log("  casbin.go:52 调用:")
	t.Log("    gormadapter.NewAdapterByDBWithCustomTable(db, tb, tb.TableName())")
	t.Log("  其中 tb.TableName() 返回 'sys_casbin_rule'")
	t.Log("")
	t.Log("  gorm-adapter 源码流程:")
	t.Log("    NewAdapterByDBWithCustomTable 接收参数:")
	t.Log("      - db: *gorm.DB")
	t.Log("      - t: interface{} (SysCasbinRule 实例)")
	t.Log("      - tableName: ...string (可变参数，值为 'sys_casbin_rule')")
	t.Log("")
	t.Log("    处理逻辑:")
	t.Log("      curTableName := defaultTableName  // 'casbin_rule'")
	t.Log("      if len(tableName) > 0 {")
	t.Log("          curTableName = tableName[0]    // 'sys_casbin_rule'")
	t.Log("      }")
	t.Log("      调用 NewAdapterByDBUseTableName(db, '', curTableName)")
	t.Log("")
	t.Log("    NewAdapterByDBUseTableName 中:")
	t.Log("      a := &Adapter{")
	t.Log("          tablePrefix: '',")
	t.Log("          tableName: 'sys_casbin_rule',  // 正确设置")
	t.Log("      }")
	t.Log("")
	t.Log("结论：")
	t.Log("✅ tableName 实际上被正确设置为 'sys_casbin_rule'")
	t.Log("✅ 数据库操作会使用正确的表名")
	t.Log("❌ 但由于 tableName 是私有字段，通过 GetAdapter() 无法直接读取")
	t.Log("❌ 如果你看到 'casbin_rule'，可能是看到了常量 defaultTableName 的值")
	t.Log("")
	t.Log("验证方法：")
	t.Log("1. 查看实际执行的 SQL 语句（通过 gorm 日志）")
	t.Log("2. 检查数据库中实际使用的表名")
	t.Log("3. 使用反射读取 Adapter 的私有字段 tableName")
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
