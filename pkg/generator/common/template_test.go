package common

import (
	"fmt"
	"testing"

	mysql_schema "github.com/xuenqlve/common/relational_database/mysql"
)

// TestTemplate_Identity 测试身份标识类模板
func TestTemplate_Identity(t *testing.T) {
	dg := NewDataGenerator()
	fmt.Println("\n=== 身份标识类模板输出示例 ===")

	templates := []string{"uuid", "object_id", "snowflake"}
	for _, tmpl := range templates {
		col := mysql_schema.Column{Name: tmpl, DataType: "varchar"}
		for i := 0; i < 3; i++ {
			value := dg.GenerateValue(col)
			fmt.Printf("%s[%d]: %v\n", tmpl, i+1, value)
		}
		fmt.Println()
	}
}

// TestTemplate_User 测试用户信息类模板
func TestTemplate_User(t *testing.T) {
	dg := NewDataGenerator()
	fmt.Println("=== 用户信息类模板输出示例 ===")

	templates := []string{"username", "email", "phone", "password", "real_name", "nickname"}
	for _, tmpl := range templates {
		col := mysql_schema.Column{Name: tmpl, DataType: "varchar"}
		fmt.Printf("%s:\n", tmpl)
		for i := 0; i < 3; i++ {
			value := dg.GenerateValue(col)
			fmt.Printf("  %d) %v\n", i+1, value)
		}
		fmt.Println()
	}
}

// TestTemplate_Address 测试地址信息类模板
func TestTemplate_Address(t *testing.T) {
	dg := NewDataGenerator()
	fmt.Println("=== 地址信息类模板输出示例 ===")

	templates := []string{"country", "province", "city", "address", "zip_code", "ipv4", "ipv6"}
	for _, tmpl := range templates {
		col := mysql_schema.Column{Name: tmpl, DataType: "varchar"}
		fmt.Printf("%s:\n", tmpl)
		for i := 0; i < 3; i++ {
			value := dg.GenerateValue(col)
			fmt.Printf("  %d) %v\n", i+1, value)
		}
		fmt.Println()
	}
}

// TestTemplate_Business 测试业务数据类模板
func TestTemplate_Business(t *testing.T) {
	dg := NewDataGenerator()
	fmt.Println("=== 业务数据类模板输出示例 ===")

	templates := []string{"url", "domain", "company", "product", "money", "price"}
	for _, tmpl := range templates {
		col := mysql_schema.Column{Name: tmpl, DataType: "varchar"}
		fmt.Printf("%s:\n", tmpl)
		for i := 0; i < 3; i++ {
			value := dg.GenerateValue(col)
			fmt.Printf("  %d) %v\n", i+1, value)
		}
		fmt.Println()
	}
}

// TestTemplate_Status 测试状态类模板
func TestTemplate_Status(t *testing.T) {
	dg := NewDataGenerator()
	fmt.Println("=== 状态类模板输出示例 ===")

	templates := []string{"status", "gender", "boolean"}
	for _, tmpl := range templates {
		col := mysql_schema.Column{Name: tmpl, DataType: "varchar"}
		fmt.Printf("%s:\n", tmpl)
		for i := 0; i < 5; i++ {
			value := dg.GenerateValue(col)
			fmt.Printf("  %d) %v\n", i+1, value)
		}
		fmt.Println()
	}
}

// TestTemplate_Time 测试时间类模板
func TestTemplate_Time(t *testing.T) {
	dg := NewDataGenerator()
	fmt.Println("=== 时间类模板输出示例 ===")

	templates := []string{"birthday", "created_at", "updated_at"}
	for _, tmpl := range templates {
		col := mysql_schema.Column{Name: tmpl, DataType: "timestamp"}
		fmt.Printf("%s:\n", tmpl)
		for i := 0; i < 3; i++ {
			value := dg.GenerateValue(col)
			fmt.Printf("  %d) %v\n", i+1, value)
		}
		fmt.Println()
	}
}

// TestTemplate_Counter 测试计数器类模板
func TestTemplate_Counter(t *testing.T) {
	dg := NewDataGenerator()
	fmt.Println("=== 计数器类模板输出示例（自增）===")

	templates := []string{"counter", "sequence"}
	for _, tmpl := range templates {
		col := mysql_schema.Column{Name: tmpl, DataType: "bigint"}
		fmt.Printf("%s:\n", tmpl)
		for i := 0; i < 5; i++ {
			value := dg.GenerateValue(col)
			fmt.Printf("  %d) %v\n", i+1, value)
		}
		fmt.Println()
	}
}

// TestTemplate_RealWorldScenario 测试真实场景：用户注册信息
func TestTemplate_RealWorldScenario(t *testing.T) {
	dg := NewDataGenerator()
	fmt.Println("=== 真实场景：虚拟用户注册信息 ===")

	for userNum := 1; userNum <= 3; userNum++ {
		fmt.Printf("\n用户 #%d 注册信息:\n", userNum)
		fmt.Printf("  用户名:    %v\n", dg.GenerateValue(mysql_schema.Column{Name: "username"}))
		fmt.Printf("  邮箱:      %v\n", dg.GenerateValue(mysql_schema.Column{Name: "email"}))
		fmt.Printf("  手机:      %v\n", dg.GenerateValue(mysql_schema.Column{Name: "phone"}))
		fmt.Printf("  密码:      %v\n", dg.GenerateValue(mysql_schema.Column{Name: "password"}))
		fmt.Printf("  真实姓名:   %v\n", dg.GenerateValue(mysql_schema.Column{Name: "real_name"}))
		fmt.Printf("  昵称:      %v\n", dg.GenerateValue(mysql_schema.Column{Name: "nickname"}))
		fmt.Printf("  城市:      %v\n", dg.GenerateValue(mysql_schema.Column{Name: "city"}))
		fmt.Printf("  地址:      %v\n", dg.GenerateValue(mysql_schema.Column{Name: "address"}))
		fmt.Printf("  创建时间:   %v\n", dg.GenerateValue(mysql_schema.Column{Name: "created_at"}))
	}
}

// TestTemplate_RealWorldScenario_Ecommerce 测试真实场景：电商产品信息
func TestTemplate_RealWorldScenario_Ecommerce(t *testing.T) {
	dg := NewDataGenerator()
	fmt.Println("=== 真实场景：电商产品信息 ===")

	for productNum := 1; productNum <= 3; productNum++ {
		fmt.Printf("\n产品 #%d 信息:\n", productNum)
		fmt.Printf("  产品ID:    %v\n", dg.GenerateValue(mysql_schema.Column{Name: "uuid"}))
		fmt.Printf("  产品名:    %v\n", dg.GenerateValue(mysql_schema.Column{Name: "product"}))
		fmt.Printf("  公司:      %v\n", dg.GenerateValue(mysql_schema.Column{Name: "company"}))
		fmt.Printf("  价格:      %v\n", dg.GenerateValue(mysql_schema.Column{Name: "price"}))
		fmt.Printf("  金额:      %v\n", dg.GenerateValue(mysql_schema.Column{Name: "money"}))
		fmt.Printf("  官网:      %v\n", dg.GenerateValue(mysql_schema.Column{Name: "url"}))
		fmt.Printf("  域名:      %v\n", dg.GenerateValue(mysql_schema.Column{Name: "domain"}))
		fmt.Printf("  上线时间:   %v\n", dg.GenerateValue(mysql_schema.Column{Name: "created_at"}))
		fmt.Printf("  更新时间:   %v\n", dg.GenerateValue(mysql_schema.Column{Name: "updated_at"}))
	}
}

// TestTemplate_AvailableTemplates 测试获取所有可用模板
func TestTemplate_AvailableTemplates(t *testing.T) {
	templates := GetAvailableTemplates()
	fmt.Println("\n=== 所有可用模板（共", len(templates), "个） ===")
	for i, tmpl := range templates {
		fmt.Printf("%2d) %s\n", i+1, tmpl)
	}
}

// TestTemplate_CustomTemplate 测试自定义模板
func TestTemplate_CustomTemplate(t *testing.T) {
	dg := NewDataGenerator()
	fmt.Println("\n=== 自定义模板示例 ===")

	// 注册自定义模板：订单ID
	counter := int64(0)
	dg.RegisterTemplate("order_id", func() any {
		counter++
		return fmt.Sprintf("ORD-%010d", counter)
	})

	// 注册自定义模板：邀请码
	dg.RegisterTemplate("invitation_code", func() any {
		return fmt.Sprintf("INVITE-%06d", 100000+counter)
	})

	col1 := mysql_schema.Column{Name: "order_id"}
	col2 := mysql_schema.Column{Name: "invitation_code"}

	fmt.Println("订单ID:")
	for i := 0; i < 3; i++ {
		fmt.Printf("  %d) %v\n", i+1, dg.GenerateValue(col1))
	}

	fmt.Println("\n邀请码:")
	for i := 0; i < 3; i++ {
		fmt.Printf("  %d) %v\n", i+1, dg.GenerateValue(col2))
	}
}
