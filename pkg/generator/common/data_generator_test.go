package common

import (
	"testing"

	mysql_schema "github.com/xuenqlve/common/relational_database/mysql"
)

func TestDataGenerator_GenerateInteger(t *testing.T) {
	dg := NewDataGenerator()

	col := mysql_schema.Column{
		Name:     "age",
		DataType: "int",
	}

	val := dg.GenerateValue(col)
	if val == nil {
		t.Fatal("expected non-nil value")
	}

	// 验证返回类型为整数
	_, ok := val.(int64)
	if !ok {
		t.Fatalf("expected int64, got %T", val)
	}
	t.Logf("mock: %v", val)
}

func TestDataGenerator_GenerateString(t *testing.T) {
	dg := NewDataGenerator()

	col := mysql_schema.Column{
		Name:     "username",
		DataType: "varchar",
	}

	val := dg.GenerateValue(col)
	if val == nil {
		t.Fatal("expected non-nil value")
	}

	str, ok := val.(string)
	if !ok {
		t.Fatalf("expected string, got %T", val)
	}

	if str == "" {
		t.Fatal("expected non-empty string")
	}
	t.Logf("mock: %v", val)
}

func TestDataGenerator_GenerateBool(t *testing.T) {
	dg := NewDataGenerator()

	col := mysql_schema.Column{
		Name:     "is_active",
		DataType: "bool",
	}

	val := dg.GenerateValue(col)
	if val == nil {
		t.Fatal("expected non-nil value")
	}

	_, ok := val.(bool)
	if !ok {
		t.Fatalf("expected bool, got %T", val)
	}
	t.Logf("mock: %v", val)
}

func TestDataGenerator_RegisterTemplate(t *testing.T) {
	dg := NewDataGenerator()

	// 注册自定义生成器
	dg.RegisterTemplate("email", func() any {
		return "test@example.com"
	})

	col := mysql_schema.Column{
		Name:     "email",
		DataType: "varchar",
	}

	val := dg.GenerateValue(col)
	if val != "test@example.com" {
		t.Errorf("expected custom value, got %v", val)
	}
	t.Logf("mock: %v", val)
}

func TestDataGenerator_SetSeed(t *testing.T) {
	dg1 := NewDataGenerator()
	dg1.SetSeed(12345)

	dg2 := NewDataGenerator()
	dg2.SetSeed(12345)

	col := mysql_schema.Column{
		Name:     "value",
		DataType: "int",
	}

	// 相同的 seed 应该生成相同的值
	val1 := dg1.GenerateValue(col)
	val2 := dg2.GenerateValue(col)

	if val1 != val2 {
		t.Errorf("expected same value with same seed, got %v != %v", val1, val2)
	}
	t.Logf("mock val1: %v val2: %v ", val1, val2)
}

func TestDataGenerator_GenerateStringWithLength(t *testing.T) {
	dg := NewDataGenerator()

	str := dg.GenerateStringWithLength(20)
	if len(str) != 20 {
		t.Errorf("expected length 20, got %d", len(str))
	}
	t.Logf("mock: %v", str)
}

func TestDataGenerator_GenerateRandomInt(t *testing.T) {
	dg := NewDataGenerator()

	for i := 0; i < 100; i++ {
		val := dg.GenerateRandomInt(10, 100)
		if val < 10 || val > 100 {
			t.Errorf("value %d out of range [10, 100]", val)
		}
		t.Logf("mock: %v", val)
	}
}

func TestParseCount(t *testing.T) {
	tests := []struct {
		input     string
		expected  int
		shouldErr bool
	}{
		{"", 1, false},
		{"10", 10, false},
		{"100", 100, false},
		{"0", 0, false},
		{"-5", 0, true},
		{"random", 0, false}, // 返回值在 1-100 之间
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		val, err := ParseCount(tt.input)

		t.Logf("val: %v", val)
		if tt.shouldErr && err == nil {
			t.Errorf("expected error for input %q", tt.input)
		}

		if !tt.shouldErr && err != nil {
			t.Errorf("unexpected error for input %q: %v", tt.input, err)
		}

		if tt.input == "random" {
			// 验证随机值在有效范围内
			if val < 1 || val > 100 {
				t.Errorf("random value %d out of expected range [1, 100]", val)
			}
		} else if !tt.shouldErr && val != tt.expected {
			t.Errorf("for input %q: expected %d, got %d", tt.input, tt.expected, val)
		}
	}
}

func TestIsValidOperation(t *testing.T) {
	tests := []struct {
		op       string
		expected bool
	}{
		{"insert", true},
		{"update", true},
		{"delete", true},
		{"replace", true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {

		result := IsValidOperation(tt.op)
		t.Logf("mock: %v", result)
		if result != tt.expected {
			t.Errorf("IsValidOperation(%q): expected %v, got %v", tt.op, tt.expected, result)
		}
	}
}

func TestNeedsOldValue(t *testing.T) {
	tests := []struct {
		op       string
		expected bool
	}{
		{"update", true},
		{"update_join", true},
		{"insert", false},
		{"delete", false},
	}

	for _, tt := range tests {
		result := NeedsOldValue(tt.op)
		if result != tt.expected {
			t.Errorf("NeedsOldValue(%q): expected %v, got %v", tt.op, tt.expected, result)
		}
	}
}

func TestNeedsGuideKeys(t *testing.T) {
	tests := []struct {
		op       string
		expected bool
	}{
		{"update", true},
		{"update_join", true},
		{"delete", true},
		{"insert", false},
		{"replace", false},
	}

	for _, tt := range tests {
		result := NeedsGuideKeys(tt.op)
		if result != tt.expected {
			t.Errorf("NeedsGuideKeys(%q): expected %v, got %v", tt.op, tt.expected, result)
		}
	}
}

// ============ Template System Tests ============

func TestDataGenerator_TemplateUUID(t *testing.T) {
	dg := NewDataGenerator()
	col := mysql_schema.Column{Name: "uuid", DataType: "varchar"}

	// 生成多个UUID值，验证格式和唯一性
	uuids := make(map[string]bool)
	for i := 0; i < 10; i++ {
		val := dg.GenerateValue(col)
		uuidStr, ok := val.(string)
		if !ok {
			t.Fatalf("expected string, got %T", val)
		}

		// 验证UUID格式 (8-4-4-4-12)
		if len(uuidStr) != 36 || uuidStr[8] != '-' || uuidStr[13] != '-' || uuidStr[18] != '-' || uuidStr[23] != '-' {
			t.Errorf("invalid UUID format: %s", uuidStr)
		}

		if uuids[uuidStr] {
			t.Errorf("duplicate UUID detected: %s", uuidStr)
		}
		uuids[uuidStr] = true
	}
}

func TestDataGenerator_TemplateEmail(t *testing.T) {
	dg := NewDataGenerator()
	col := mysql_schema.Column{Name: "email", DataType: "varchar"}

	for i := 0; i < 10; i++ {
		val := dg.GenerateValue(col)
		email, ok := val.(string)
		if !ok {
			t.Fatalf("expected string, got %T", val)
		}

		// 验证邮箱格式
		if !contains(email, "@") || !contains(email, ".") {
			t.Errorf("invalid email format: %s", email)
		}
	}
}

func TestDataGenerator_TemplatePhone(t *testing.T) {
	dg := NewDataGenerator()
	col := mysql_schema.Column{Name: "phone", DataType: "varchar"}

	for i := 0; i < 10; i++ {
		val := dg.GenerateValue(col)
		phone, ok := val.(string)
		if !ok {
			t.Fatalf("expected string, got %T", val)
		}

		// 验证手机号长度为11位
		if len(phone) != 11 {
			t.Errorf("invalid phone length: %s (len=%d)", phone, len(phone))
		}

		// 验证只包含数字
		for _, ch := range phone {
			if ch < '0' || ch > '9' {
				t.Errorf("phone contains non-digit character: %s", phone)
			}
		}
	}
}

func TestDataGenerator_TemplatePassword(t *testing.T) {
	dg := NewDataGenerator()
	col := mysql_schema.Column{Name: "password", DataType: "varchar"}

	for i := 0; i < 10; i++ {
		val := dg.GenerateValue(col)
		pwd, ok := val.(string)
		if !ok {
			t.Fatalf("expected string, got %T", val)
		}

		// 验证密码长度合理
		if len(pwd) < 8 || len(pwd) > 50 {
			t.Errorf("invalid password length: %s (len=%d)", pwd, len(pwd))
		}

		// 验证包含大小写字母和特殊字符
		hasUpper, hasLower, hasSpecial := false, false, false
		for _, ch := range pwd {
			if ch >= 'A' && ch <= 'Z' {
				hasUpper = true
			}
			if ch >= 'a' && ch <= 'z' {
				hasLower = true
			}
			if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')) {
				hasSpecial = true
			}
		}
		if !hasUpper || !hasLower || !hasSpecial {
			t.Errorf("password lacks complexity: %s", pwd)
		}
	}
}

func TestDataGenerator_TemplateCity(t *testing.T) {
	dg := NewDataGenerator()
	col := mysql_schema.Column{Name: "city", DataType: "varchar"}

	cities := make(map[string]bool)
	for i := 0; i < 10; i++ {
		val := dg.GenerateValue(col)
		city, ok := val.(string)
		if !ok {
			t.Fatalf("expected string, got %T", val)
		}

		if city == "" {
			t.Errorf("empty city generated")
		}
		cities[city] = true
	}

	// 验证至少生成了一些不同的城市
	if len(cities) < 3 {
		t.Errorf("expected at least 3 different cities, got %d", len(cities))
	}
}

func TestDataGenerator_TemplateIPv4(t *testing.T) {
	dg := NewDataGenerator()
	col := mysql_schema.Column{Name: "ipv4", DataType: "varchar"}

	for i := 0; i < 10; i++ {
		val := dg.GenerateValue(col)
		ip, ok := val.(string)
		if !ok {
			t.Fatalf("expected string, got %T", val)
		}

		// 验证IPv4格式
		parts := split(ip, ".")
		if len(parts) != 4 {
			t.Errorf("invalid IPv4 format: %s", ip)
		}

		for _, part := range parts {
			// 验证每部分都是数字
			for _, ch := range part {
				if ch < '0' || ch > '9' {
					t.Errorf("invalid IPv4 part: %s", part)
				}
			}
		}
	}
}

func TestDataGenerator_TemplateURL(t *testing.T) {
	dg := NewDataGenerator()
	col := mysql_schema.Column{Name: "url", DataType: "varchar"}

	for i := 0; i < 10; i++ {
		val := dg.GenerateValue(col)
		url, ok := val.(string)
		if !ok {
			t.Fatalf("expected string, got %T", val)
		}

		// 验证URL格式
		if !contains(url, "https://") && !contains(url, "http://") {
			t.Errorf("invalid URL format: %s", url)
		}
	}
}

func TestDataGenerator_TemplateCounter(t *testing.T) {
	dg := NewDataGenerator()
	col := mysql_schema.Column{Name: "counter", DataType: "bigint"}

	// 计数器应该自增
	prevVal := int64(-1)
	for i := 0; i < 10; i++ {
		val := dg.GenerateValue(col)
		counter, ok := val.(int64)
		if !ok {
			t.Fatalf("expected int64, got %T", val)
		}

		if counter <= prevVal {
			t.Errorf("counter not incrementing: %d <= %d", counter, prevVal)
		}
		prevVal = counter
	}
}

func TestDataGenerator_TemplateSequence(t *testing.T) {
	dg := NewDataGenerator()
	col := mysql_schema.Column{Name: "sequence", DataType: "bigint"}

	// 序列号应该自增且从1000000000开始
	prevVal := int64(999999999)
	for i := 0; i < 10; i++ {
		val := dg.GenerateValue(col)
		seq, ok := val.(int64)
		if !ok {
			t.Fatalf("expected int64, got %T", val)
		}

		if seq <= prevVal {
			t.Errorf("sequence not incrementing: %d <= %d", seq, prevVal)
		}
		prevVal = seq
	}
}

func TestDataGenerator_TemplateGender(t *testing.T) {
	dg := NewDataGenerator()
	col := mysql_schema.Column{Name: "gender", DataType: "varchar"}

	validGenders := map[string]bool{"male": true, "female": true, "other": true}
	for i := 0; i < 20; i++ {
		val := dg.GenerateValue(col)
		gender, ok := val.(string)
		if !ok {
			t.Fatalf("expected string, got %T", val)
		}

		if !validGenders[gender] {
			t.Errorf("invalid gender value: %s", gender)
		}
	}
}

func TestDataGenerator_TemplateBoolean(t *testing.T) {
	dg := NewDataGenerator()
	col := mysql_schema.Column{Name: "boolean", DataType: "varchar"}

	validBools := map[string]bool{"true": true, "false": true}
	for i := 0; i < 20; i++ {
		val := dg.GenerateValue(col)
		boolStr, ok := val.(string)
		if !ok {
			t.Fatalf("expected string, got %T", val)
		}

		if !validBools[boolStr] {
			t.Errorf("invalid boolean value: %s", boolStr)
		}
	}
}

func TestDataGenerator_AllTemplatesAvailable(t *testing.T) {
	templates := GetAvailableTemplates()

	// 应该至少有30个模板
	if len(templates) < 30 {
		t.Errorf("expected at least 30 templates, got %d", len(templates))
	}

	// 验证一些必要的模板存在
	requiredTemplates := []string{
		"uuid", "object_id", "snowflake",
		"username", "email", "phone", "password", "real_name", "nickname",
		"country", "province", "city", "address", "zip_code", "ipv4", "ipv6",
		"url", "domain", "company", "product", "money", "price",
		"status", "gender", "boolean",
		"birthday", "created_at", "updated_at",
		"counter", "sequence",
	}

	templateMap := make(map[string]bool)
	for _, tmpl := range templates {
		templateMap[tmpl] = true
	}

	for _, required := range requiredTemplates {
		if !templateMap[required] {
			t.Errorf("required template '%s' not found", required)
		}
	}
}

func TestDataGenerator_TemplateIntegration(t *testing.T) {
	dg := NewDataGenerator()

	// 测试完整的用户注册场景
	cols := []string{"username", "email", "phone", "password", "real_name", "nickname", "city", "address"}
	values := make(map[string]any)

	for _, colName := range cols {
		col := mysql_schema.Column{Name: colName, DataType: "varchar"}
		val := dg.GenerateValue(col)
		if val == nil {
			t.Errorf("failed to generate value for column: %s", colName)
		}
		values[colName] = val
	}

	// 验证所有值都被生成了
	if len(values) != len(cols) {
		t.Errorf("expected %d values, got %d", len(cols), len(values))
	}
}

// 辅助函数
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func split(s, sep string) []string {
	var result []string
	var current string
	for _, ch := range s {
		if string(ch) == sep {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}
