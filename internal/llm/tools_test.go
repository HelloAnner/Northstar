/**
 * LLM 工具定义单元测试
 *
 * @author Anner
 * @since 12.0
 * Created on 2026/2/1
 */
package llm

import (
	"testing"

	"github.com/tmc/langchaingo/llms"
)

// TestTools 测试工具定义完整性
func TestTools(t *testing.T) {
	tools := Tools()

	// 验证工具数量
	if len(tools) != 2 {
		t.Errorf("期望 2 个工具，实际 %d", len(tools))
	}

	// 验证工具名称
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		if tool.Function != nil {
			toolNames[tool.Function.Name] = true
		}
	}

	if !toolNames[ToolSetIndicatorTargets] {
		t.Error("缺少 set_indicator_targets 工具")
	}
	if !toolNames[ToolUpdateCompanies] {
		t.Error("缺少 update_companies 工具")
	}
}

// TestIndicatorTargetsTool 测试指标目标工具定义
func TestIndicatorTargetsTool(t *testing.T) {
	tool := indicatorTargetsTool()

	// 验证工具类型
	if tool.Type != "function" {
		t.Errorf("工具类型期望 'function'，实际 '%s'", tool.Type)
	}

	// 验证函数定义
	if tool.Function == nil {
		t.Fatal("函数定义不应为 nil")
	}

	// 验证名称
	if tool.Function.Name != ToolSetIndicatorTargets {
		t.Errorf("工具名称期望 '%s'，实际 '%s'", ToolSetIndicatorTargets, tool.Function.Name)
	}

	// 验证描述
	if tool.Function.Description == "" {
		t.Error("工具描述不应为空")
	}

	// 验证参数定义
	if tool.Function.Parameters == nil {
		t.Fatal("参数定义不应为 nil")
	}
}

// TestUpdateCompaniesTool 测试企业更新工具定义
func TestUpdateCompaniesTool(t *testing.T) {
	tool := updateCompaniesTool()

	// 验证工具类型
	if tool.Type != "function" {
		t.Errorf("工具类型期望 'function'，实际 '%s'", tool.Type)
	}

	// 验证函数定义
	if tool.Function == nil {
		t.Fatal("函数定义不应为 nil")
	}

	// 验证名称
	if tool.Function.Name != ToolUpdateCompanies {
		t.Errorf("工具名称期望 '%s'，实际 '%s'", ToolUpdateCompanies, tool.Function.Name)
	}

	// 验证描述
	if tool.Function.Description == "" {
		t.Error("工具描述不应为空")
	}
}

// TestCompanyPatchProperties 测试企业字段属性定义
func TestCompanyPatchProperties(t *testing.T) {
	// 验证关键字段存在
	requiredFields := []string{
		// 批零企业字段
		"salesCurrentMonth",
		"salesLastYearMonth",
		"retailCurrentMonth",
		"retailLastYearMonth",
		"salesMonthRate",
		"retailMonthRate",
		// 住餐企业字段
		"revenueCurrentMonth",
		"revenueLastYearMonth",
		"roomCurrentMonth",
		"foodCurrentMonth",
		"goodsCurrentMonth",
		// 通用字段
		"isSmallMicro",
		"isEatWearUse",
	}

	for _, field := range requiredFields {
		if _, ok := companyPatchProperties[field]; !ok {
			t.Errorf("缺少字段定义: %s", field)
		}
	}

	// 验证字段类型定义
	for field, def := range companyPatchProperties {
		prop, ok := def.(map[string]any)
		if !ok {
			t.Errorf("字段 %s 定义格式错误", field)
			continue
		}
		if prop["type"] == nil {
			t.Errorf("字段 %s 缺少类型定义", field)
		}
	}
}

// TestIndicatorTargetsSchema 测试指标目标 Schema
func TestIndicatorTargetsSchema(t *testing.T) {
	// 验证类型
	if indicatorTargetsSchema["type"] != "object" {
		t.Error("Schema 类型应为 object")
	}

	// 验证属性
	properties, ok := indicatorTargetsSchema["properties"].(map[string]any)
	if !ok {
		t.Fatal("properties 定义错误")
	}

	// 验证 targets 属性
	targets, ok := properties["targets"].(map[string]any)
	if !ok {
		t.Fatal("targets 定义错误")
	}

	if targets["type"] != "object" {
		t.Error("targets 类型应为 object")
	}

	// 验证 required
	required, ok := indicatorTargetsSchema["required"].([]string)
	if !ok || len(required) != 1 || required[0] != "targets" {
		t.Error("required 应包含 'targets'")
	}
}

// TestUpdateCompaniesSchema 测试企业更新 Schema
func TestUpdateCompaniesSchema(t *testing.T) {
	// 验证类型
	if updateCompaniesSchema["type"] != "object" {
		t.Error("Schema 类型应为 object")
	}

	// 验证属性
	properties, ok := updateCompaniesSchema["properties"].(map[string]any)
	if !ok {
		t.Fatal("properties 定义错误")
	}

	// 验证 updates 属性
	updates, ok := properties["updates"].(map[string]any)
	if !ok {
		t.Fatal("updates 定义错误")
	}

	if updates["type"] != "array" {
		t.Error("updates 类型应为 array")
	}

	// 验证数组项定义
	items, ok := updates["items"].(map[string]any)
	if !ok {
		t.Fatal("items 定义错误")
	}

	itemProps, ok := items["properties"].(map[string]any)
	if !ok {
		t.Fatal("item properties 定义错误")
	}

	// 验证 id 和 patch 字段
	if _, ok := itemProps["id"]; !ok {
		t.Error("item 缺少 id 字段定义")
	}
	if _, ok := itemProps["patch"]; !ok {
		t.Error("item 缺少 patch 字段定义")
	}

	// 验证 required
	itemRequired, ok := items["required"].([]string)
	if !ok {
		t.Fatal("item required 定义错误")
	}

	hasID := false
	hasPatch := false
	for _, r := range itemRequired {
		if r == "id" {
			hasID = true
		}
		if r == "patch" {
			hasPatch = true
		}
	}

	if !hasID {
		t.Error("item required 应包含 'id'")
	}
	if !hasPatch {
		t.Error("item required 应包含 'patch'")
	}
}

// TestToolConstants 测试工具常量
func TestToolConstants(t *testing.T) {
	// 验证常量值
	if ToolSetIndicatorTargets != "set_indicator_targets" {
		t.Errorf("ToolSetIndicatorTargets 常量值错误: %s", ToolSetIndicatorTargets)
	}

	if ToolUpdateCompanies != "update_companies" {
		t.Errorf("ToolUpdateCompanies 常量值错误: %s", ToolUpdateCompanies)
	}
}

// TestToolCompatibility 测试工具兼容性
func TestToolCompatibility(t *testing.T) {
	// 验证工具可以被正确转换为 llms.Tool
	tools := Tools()

	for _, tool := range tools {
		// 验证是有效的 llms.Tool
		var _ llms.Tool = tool

		// 验证 Function 不为 nil
		if tool.Function == nil {
			t.Error("工具 Function 不应为 nil")
		}

		// 验证 Parameters 是有效的 map
		if tool.Function.Parameters == nil {
			t.Error("工具 Parameters 不应为 nil")
		}
	}
}

// TestCompanyPatchPropertyTypes 测试字段类型定义
func TestCompanyPatchPropertyTypes(t *testing.T) {
	// 数值字段
	numericFields := []string{
		"salesCurrentMonth",
		"retailCurrentMonth",
		"revenueCurrentMonth",
		"roomCurrentMonth",
		"foodCurrentMonth",
		"goodsCurrentMonth",
	}

	for _, field := range numericFields {
		def, ok := companyPatchProperties[field].(map[string]any)
		if !ok {
			t.Errorf("字段 %s 定义不存在", field)
			continue
		}
		if def["type"] != "number" {
			t.Errorf("字段 %s 类型应为 number", field)
		}
	}

	// 布尔字段
	boolFields := []string{"isSmallMicro", "isEatWearUse"}

	for _, field := range boolFields {
		def, ok := companyPatchProperties[field].(map[string]any)
		if !ok {
			t.Errorf("字段 %s 定义不存在", field)
			continue
		}
		if def["type"] != "boolean" {
			t.Errorf("字段 %s 类型应为 boolean", field)
		}
	}
}
