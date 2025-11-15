package mock

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"sync/atomic"
	"time"
)

// TemplateType 模板类型常量
type TemplateType string

// 所有支持的模板类型常量
const (
	// 身份标识类
	TemplateUUID      TemplateType = "uuid"
	TemplateObjectID  TemplateType = "object_id"
	TemplateSnowflake TemplateType = "snowflake"

	// 用户信息类
	TemplateUsername TemplateType = "username"
	TemplateEmail    TemplateType = "email"
	TemplatePhone    TemplateType = "phone"
	TemplatePassword TemplateType = "password"
	TemplateRealName TemplateType = "real_name"
	TemplateNickname TemplateType = "nickname"

	// 地址信息类
	TemplateCountry  TemplateType = "country"
	TemplateProvince TemplateType = "province"
	TemplateCity     TemplateType = "city"
	TemplateAddress  TemplateType = "address"
	TemplateZipCode  TemplateType = "zip_code"
	TemplateIPv4     TemplateType = "ipv4"
	TemplateIPv6     TemplateType = "ipv6"

	// 业务数据类
	TemplateURL     TemplateType = "url"
	TemplateDomain  TemplateType = "domain"
	TemplateCompany TemplateType = "company"
	TemplateProduct TemplateType = "product"
	TemplateMoney   TemplateType = "money"
	TemplatePrice   TemplateType = "price"

	// 状态类
	TemplateStatus  TemplateType = "status"
	TemplateGender  TemplateType = "gender"
	TemplateBoolean TemplateType = "boolean"

	// 时间类
	TemplateBirthday  TemplateType = "birthday"
	TemplateCreatedAt TemplateType = "created_at"
	TemplateUpdatedAt TemplateType = "updated_at"

	// 计数器类
	TemplateCounter  TemplateType = "counter"
	TemplateSequence TemplateType = "sequence"
)

// 全局计数器，用于生成自增序列
var (
	globalCounter  int64 = 0
	globalSequence int64 = 1000000000
)

// RegisterAllBuiltinTemplates 注册所有内置模板
// 在 DataGenerator 初始化时调用
func RegisterAllBuiltinTemplates(dg *DataGenerator) {
	// 身份标识类
	registerIdentityTemplates(dg)

	// 用户信息类
	registerUserTemplates(dg)

	// 地址信息类
	registerAddressTemplates(dg)

	// 业务数据类
	registerBusinessTemplates(dg)

	// 状态类
	registerStatusTemplates(dg)

	// 时间类
	registerTimeTemplates(dg)

	// 计数器类
	registerCounterTemplates(dg)
}

// ============ 身份标识类模板 ============

func registerIdentityTemplates(dg *DataGenerator) {
	dg.RegisterTemplate("id", generateId)
	dg.RegisterTemplate("uuid", generateUUID)
	dg.RegisterTemplate("object_id", generateObjectID)
	dg.RegisterTemplate("snowflake", generateSnowflake)
}

var id int64 = 1

func generateId() any {
	defer atomic.AddInt64(&id, 1)
	return id
}

// generateUUID 生成 UUID v4 格式字符串
func generateUUID() any {
	rnd := rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), uint64(time.Now().UnixNano())))
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		rnd.IntN(0xffffffff),
		rnd.IntN(0xffff),
		rnd.IntN(0xffff)|0x4000,
		rnd.IntN(0xffff)|0x8000,
		rnd.Int64()&0xffffffffffff)
}

// generateObjectID 生成 MongoDB ObjectID 风格的 ID
func generateObjectID() any {
	rnd := rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), uint64(time.Now().UnixNano())))
	timestamp := time.Now().Unix()
	return fmt.Sprintf("%016x%016x", timestamp, rnd.Int64())
}

// generateSnowflake 生成 Snowflake ID
func generateSnowflake() any {
	rnd := rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), uint64(time.Now().UnixNano())))
	// Snowflake: 41 bits timestamp + 10 bits machine + 12 bits sequence
	timestamp := time.Now().UnixMilli()
	machineID := int64(rnd.IntN(1024))
	sequence := int64(rnd.IntN(4096))
	return (timestamp << 22) | (machineID << 12) | sequence
}

// ============ 用户信息类模板 ============

func registerUserTemplates(dg *DataGenerator) {
	dg.RegisterTemplate("username", generateUsername)
	dg.RegisterTemplate("email", generateEmail)
	dg.RegisterTemplate("phone", generatePhone)
	dg.RegisterTemplate("password", generatePassword)
	dg.RegisterTemplate("real_name", generateRealName)
	dg.RegisterTemplate("nickname", generateNickname)
}

// generateUsername 生成用户名
func generateUsername() any {
	prefixes := []string{"user", "player", "member", "admin", "guest", "test"}
	prefix := prefixes[rand.IntN(len(prefixes))]
	return fmt.Sprintf("%s_%d", prefix, rand.IntN(1000000))
}

// generateEmail 生成邮箱地址
func generateEmail() any {
	domains := []string{"gmail.com", "qq.com", "163.com", "outlook.com", "yahoo.com", "hotmail.com"}
	domain := domains[rand.IntN(len(domains))]
	return fmt.Sprintf("user_%d@%s", rand.IntN(1000000), domain)
}

// generatePhone 生成手机号（中国格式）
func generatePhone() any {
	prefixes := []string{"130", "131", "132", "133", "134", "135", "136", "137", "138", "139",
		"150", "151", "152", "153", "155", "156", "157", "158", "159",
		"180", "181", "182", "183", "184", "185", "186", "187", "188", "189"}
	prefix := prefixes[rand.IntN(len(prefixes))]
	return fmt.Sprintf("%s%08d", prefix, rand.IntN(100000000))
}

// generatePassword 生成密码（仅供测试，生产环境应使用加密密码）
func generatePassword() any {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%"
	password := make([]byte, 12)
	for i := range password {
		password[i] = charset[rand.IntN(len(charset))]
	}
	return string(password)
}

// generateRealName 生成真实姓名（中文）
func generateRealName() any {
	surnames := []string{"张", "王", "李", "赵", "刘", "陈", "杨", "黄", "周", "吴",
		"徐", "孙", "朱", "马", "胡", "郭", "何", "林", "高", "郑"}
	givenNames := []string{"伟", "明", "浩", "强", "涛", "峰", "磊", "刚", "军", "生",
		"娟", "秀", "美", "霞", "英", "莉", "华", "敏", "丽", "玲"}

	surname := surnames[rand.IntN(len(surnames))]
	givenName := givenNames[rand.IntN(len(givenNames))]

	// 某些情况生成三字名
	if rand.IntN(3) == 0 {
		givenName = givenNames[rand.IntN(len(givenNames))] + givenNames[rand.IntN(len(givenNames))]
	}

	return surname + givenName
}

// generateNickname 生成昵称
func generateNickname() any {
	adjectives := []string{"Happy", "Lucky", "Cool", "Smart", "Strong", "Quick", "Bright", "Brave"}
	nouns := []string{"Tiger", "Eagle", "Phoenix", "Dragon", "Wolf", "Lion", "Bear", "Shark"}
	adj := adjectives[rand.IntN(len(adjectives))]
	noun := nouns[rand.IntN(len(nouns))]
	return fmt.Sprintf("%s%s_%d", adj, noun, rand.IntN(1000))
}

// ============ 地址信息类模板 ============

func registerAddressTemplates(dg *DataGenerator) {
	dg.RegisterTemplate("country", generateCountry)
	dg.RegisterTemplate("province", generateProvince)
	dg.RegisterTemplate("city", generateCity)
	dg.RegisterTemplate("address", generateAddress)
	dg.RegisterTemplate("zip_code", generateZipCode)
	dg.RegisterTemplate("ipv4", generateIPv4)
	dg.RegisterTemplate("ipv6", generateIPv6)
}

// generateCountry 生成国家名称
func generateCountry() any {
	countries := []string{"China", "United States", "India", "Brazil", "Russia",
		"Japan", "Germany", "United Kingdom", "France", "Italy"}
	return countries[rand.IntN(len(countries))]
}

// generateProvince 生成省份名称（中国）
func generateProvince() any {
	provinces := []string{"北京", "上海", "广东", "浙江", "江苏", "山东", "四川", "湖北",
		"福建", "安徽", "湖南", "河南", "陕西", "云南", "辽宁", "江西"}
	return provinces[rand.IntN(len(provinces))]
}

// generateCity 生成城市名称
func generateCity() any {
	cities := []string{"北京", "上海", "广州", "深圳", "杭州", "成都", "武汉", "西安",
		"苏州", "南京", "天津", "重庆", "长沙", "济南", "青岛", "郑州"}
	return cities[rand.IntN(len(cities))]
}

// generateAddress 生成详细地址
func generateAddress() any {
	streets := []string{"中关村", "淮海路", "春熙路", "天府大道", "汉口路", "解放路"}
	street := streets[rand.IntN(len(streets))]
	number := rand.IntN(1000) + 1
	building := "写字楼" + fmt.Sprintf("%c", byte('A')+byte(rand.IntN(5)))
	room := rand.IntN(1000) + 1
	return fmt.Sprintf("%s%d号%s%d室", street, number, building, room)
}

// generateZipCode 生成邮编
func generateZipCode() any {
	return fmt.Sprintf("%06d", rand.IntN(1000000))
}

// generateIPv4 生成 IPv4 地址
func generateIPv4() any {
	return fmt.Sprintf("%d.%d.%d.%d",
		rand.IntN(256), rand.IntN(256),
		rand.IntN(256), rand.IntN(256))
}

// generateIPv6 生成 IPv6 地址
func generateIPv6() any {
	parts := make([]string, 8)
	for i := 0; i < 8; i++ {
		parts[i] = fmt.Sprintf("%04x", rand.IntN(0x10000))
	}
	return strings.Join(parts, ":")
}

// ============ 业务数据类模板 ============

func registerBusinessTemplates(dg *DataGenerator) {
	dg.RegisterTemplate("url", generateURL)
	dg.RegisterTemplate("domain", generateDomain)
	dg.RegisterTemplate("company", generateCompany)
	dg.RegisterTemplate("product", generateProduct)
	dg.RegisterTemplate("money", generateMoney)
	dg.RegisterTemplate("price", generatePrice)
}

// generateURL 生成 URL
func generateURL() any {
	paths := []string{"home", "product", "user", "order", "cart", "checkout", "payment"}
	path := paths[rand.IntN(len(paths))]
	id := rand.IntN(100000)
	return fmt.Sprintf("https://example.com/%s/%d", path, id)
}

// generateDomain 生成域名
func generateDomain() any {
	names := []string{"example", "demo", "test", "sample", "app", "service", "api"}
	tlds := []string{"com", "cn", "io", "net", "org", "co"}
	name := names[rand.IntN(len(names))]
	tld := tlds[rand.IntN(len(tlds))]
	number := rand.IntN(1000)
	if number > 0 {
		return fmt.Sprintf("%s%d.%s", name, number, tld)
	}
	return fmt.Sprintf("%s.%s", name, tld)
}

// generateCompany 生成公司名称
func generateCompany() any {
	prefixes := []string{"北京", "上海", "深圳", "杭州", "成都"}
	companies := []string{"科技有限公司", "互联网有限公司", "信息技术有限公司", "网络有限公司", "创意有限公司"}
	prefix := prefixes[rand.IntN(len(prefixes))]
	company := companies[rand.IntN(len(companies))]
	return fmt.Sprintf("%s%s", prefix, company)
}

// generateProduct 生成产品名称
func generateProduct() any {
	adjectives := []string{"Premium", "Smart", "Pro", "Ultra", "Max", "Plus"}
	categories := []string{"Phone", "Tablet", "Laptop", "Watch", "Headphone", "Speaker"}
	adj := adjectives[rand.IntN(len(adjectives))]
	cat := categories[rand.IntN(len(categories))]
	version := rand.IntN(10) + 1
	return fmt.Sprintf("%s %s V%d", adj, cat, version)
}

// generateMoney 生成金额（元）
func generateMoney() any {
	return fmt.Sprintf("%.2f", rand.Float64()*10000)
}

// generatePrice 生成价格
func generatePrice() any {
	return fmt.Sprintf("¥%.2f", rand.Float64()*1000)
}

// ============ 状态类模板 ============

func registerStatusTemplates(dg *DataGenerator) {
	dg.RegisterTemplate("status", generateStatus)
	dg.RegisterTemplate("gender", generateGender)
	dg.RegisterTemplate("boolean", generateBooleanString)
}

// generateStatus 生成状态值
func generateStatus() any {
	statuses := []string{"active", "inactive", "pending", "deleted", "archived", "disabled"}
	return statuses[rand.IntN(len(statuses))]
}

// generateGender 生成性别
func generateGender() any {
	genders := []string{"male", "female", "other"}
	return genders[rand.IntN(len(genders))]
}

// generateBooleanString 生成布尔值字符串
func generateBooleanString() any {
	if rand.IntN(2) == 0 {
		return "true"
	}
	return "false"
}

// ============ 时间类模板 ============

func registerTimeTemplates(dg *DataGenerator) {
	dg.RegisterTemplate("birthday", generateBirthday)
	dg.RegisterTemplate("created_at", generateCreatedAt)
	dg.RegisterTemplate("updated_at", generateUpdatedAt)
}

// generateBirthday 生成生日
func generateBirthday() any {
	now := time.Now()
	year := now.Year() - rand.IntN(60) - 18 // 18-78 岁
	month := rand.IntN(12) + 1
	day := rand.IntN(28) + 1
	return fmt.Sprintf("%04d-%02d-%02d", year, month, day)
}

// generateCreatedAt 生成创建时间（最近 365 天内）
func generateCreatedAt() any {
	now := time.Now()
	daysAgo := rand.IntN(365)
	timestamp := now.AddDate(0, 0, -daysAgo).Format("2006-01-02 15:04:05")
	return timestamp
}

// generateUpdatedAt 生成更新时间（最近 30 天内）
func generateUpdatedAt() any {
	now := time.Now()
	daysAgo := rand.IntN(30)
	timestamp := now.AddDate(0, 0, -daysAgo).Format("2006-01-02 15:04:05")
	return timestamp
}

// ============ 计数器类模板 ============

func registerCounterTemplates(dg *DataGenerator) {
	dg.RegisterTemplate("counter", generateCounter)
	dg.RegisterTemplate("sequence", generateSequence)
}

// generateCounter 生成计数器值
func generateCounter() any {
	globalCounter++
	return globalCounter
}

// generateSequence 生成序列号
func generateSequence() any {
	globalSequence++
	return globalSequence
}

// 辅助函数：获取所有可用的模板类型列表
func GetAvailableTemplates() []string {
	return []string{
		// 身份标识
		string(TemplateUUID), string(TemplateObjectID), string(TemplateSnowflake),
		// 用户信息
		string(TemplateUsername), string(TemplateEmail), string(TemplatePhone),
		string(TemplatePassword), string(TemplateRealName), string(TemplateNickname),
		// 地址信息
		string(TemplateCountry), string(TemplateProvince), string(TemplateCity),
		string(TemplateAddress), string(TemplateZipCode), string(TemplateIPv4), string(TemplateIPv6),
		// 业务数据
		string(TemplateURL), string(TemplateDomain), string(TemplateCompany),
		string(TemplateProduct), string(TemplateMoney), string(TemplatePrice),
		// 状态
		string(TemplateStatus), string(TemplateGender), string(TemplateBoolean),
		// 时间
		string(TemplateBirthday), string(TemplateCreatedAt), string(TemplateUpdatedAt),
		// 计数器
		string(TemplateCounter), string(TemplateSequence),
	}
}
