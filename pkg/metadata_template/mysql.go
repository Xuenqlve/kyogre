package metadata_template

import (
	"github.com/xuenqlve/kyogre/internal/metadata"
	"github.com/xuenqlve/kyogre/pkg/metadata/mysql"
)

const (
	EcommerceTemplate = "ecommerce"
	SchoolTemplate    = "school"
	OfficeTemplate    = "office"
	HospitalTemplate  = "hospital"
)

func init() {
	// 注册商品订单网购场景
	metadata.MustRegister(EcommerceTemplate, EcommerceSchema)

	// 注册学生课程表学校场景
	metadata.MustRegister(SchoolTemplate, SchoolSchema)

	// 注册领导员工办公场景
	metadata.MustRegister(OfficeTemplate, OfficeSchema)

	// 注册医生医院场景
	metadata.MustRegister(HospitalTemplate, HospitalSchema)
}

// ================== 商品订单网购场景 ==================

// EcommerceSchema 返回电商网购场景的数据库配置
// 包含用户、商品、订单、订单项、购物车、评价等表
func EcommerceSchema() any {
	return mysql.Database{
		"ecommerce": {
			// 用户表
			{
				Table: "users",
				Columns: []mysql.Column{
					{Column: "id", Type: mysql.Primary},
					{Column: "username", Type: mysql.Varchar},
					{Column: "email", Type: mysql.Varchar},
					{Column: "phone", Type: mysql.Varchar},
					{Column: "password_hash", Type: mysql.Varchar},
					{Column: "status", Type: mysql.Tinyint},
					{Column: "created_at", Type: mysql.Datetime},
					{Column: "updated_at", Type: mysql.TimestampUpdate},
				},
				Indexes: []mysql.Index{
					{Name: "Primary", Columns: []string{"id"}, IsPrimary: true},
					{Name: "uk_username", Columns: []string{"username"}, IsUnique: true},
					{Name: "uk_email", Columns: []string{"email"}, IsUnique: true},
				},
			},
			// 商品表
			{
				Table: "products",
				Columns: []mysql.Column{
					{Column: "id", Type: mysql.Primary},
					{Column: "category_id", Type: mysql.Bigint},
					{Column: "product_name", Type: mysql.Varchar},
					{Column: "description", Type: mysql.Text},
					{Column: "price", Type: mysql.Decimal},
					{Column: "cost", Type: mysql.Decimal},
					{Column: "stock", Type: mysql.Int},
					{Column: "status", Type: mysql.Tinyint},
					{Column: "created_at", Type: mysql.Datetime},
					{Column: "updated_at", Type: mysql.TimestampUpdate},
				},
				Indexes: []mysql.Index{
					{Name: "pk_product_id", Columns: []string{"id"}, IsPrimary: true},
					{Name: "idx_category", Columns: []string{"category_id"}},
					{Name: "idx_status", Columns: []string{"status"}},
				},
			},
			// 订单表
			{
				Table: "orders",
				Columns: []mysql.Column{
					{Column: "id", Type: mysql.Primary},
					{Column: "user_id", Type: mysql.Bigint},
					{Column: "order_no", Type: mysql.Varchar},
					{Column: "total_amount", Type: mysql.Decimal},
					{Column: "discount_amount", Type: mysql.Decimal},
					{Column: "pay_amount", Type: mysql.Decimal},
					{Column: "status", Type: mysql.Tinyint},
					{Column: "payment_method", Type: mysql.Tinyint},
					{Column: "shipping_address", Type: mysql.VarcharLarge},
					{Column: "created_at", Type: mysql.Datetime},
					{Column: "updated_at", Type: mysql.TimestampUpdate},
				},
				Indexes: []mysql.Index{
					{Name: "pk_order_id", Columns: []string{"id"}, IsPrimary: true},
					{Name: "idx_user_id", Columns: []string{"user_id"}},
					{Name: "uk_order_no", Columns: []string{"order_no"}, IsUnique: true},
					{Name: "idx_status", Columns: []string{"status"}},
				},
			},
			// 订单项目表
			{
				Table: "order_items",
				Columns: []mysql.Column{
					{Column: "id", Type: mysql.Primary},
					{Column: "order_id", Type: mysql.Bigint},
					{Column: "product_id", Type: mysql.Bigint},
					{Column: "quantity", Type: mysql.Int},
					{Column: "unit_price", Type: mysql.Decimal},
					{Column: "subtotal", Type: mysql.Decimal},
					{Column: "created_at", Type: mysql.Datetime},
				},
				Indexes: []mysql.Index{
					{Name: "pk_order_item_id", Columns: []string{"id"}, IsPrimary: true},
					{Name: "idx_order_id", Columns: []string{"order_id"}},
					{Name: "idx_product_id", Columns: []string{"product_id"}},
				},
			},
			// 购物车表
			{
				Table: "shopping_carts",
				Columns: []mysql.Column{
					{Column: "id", Type: mysql.Primary},
					{Column: "user_id", Type: mysql.Bigint},
					{Column: "product_id", Type: mysql.Bigint},
					{Column: "quantity", Type: mysql.Int},
					{Column: "created_at", Type: mysql.Datetime},
					{Column: "updated_at", Type: mysql.TimestampUpdate},
				},
				Indexes: []mysql.Index{
					{Name: "pk_cart_id", Columns: []string{"id"}, IsPrimary: true},
					{Name: "uk_user_product", Columns: []string{"user_id", "product_id"}, IsUnique: true},
				},
			},
			// 评价表
			{
				Table: "reviews",
				Columns: []mysql.Column{
					{Column: "id", Type: mysql.Primary},
					{Column: "product_id", Type: mysql.Bigint},
					{Column: "user_id", Type: mysql.Bigint},
					{Column: "order_id", Type: mysql.Bigint},
					{Column: "rating", Type: mysql.Tinyint},
					{Column: "title", Type: mysql.Varchar},
					{Column: "content", Type: mysql.Text},
					{Column: "helpful_count", Type: mysql.Int},
					{Column: "created_at", Type: mysql.Datetime},
				},
				Indexes: []mysql.Index{
					{Name: "pk_review_id", Columns: []string{"id"}, IsPrimary: true},
					{Name: "idx_product_id", Columns: []string{"product_id"}},
					{Name: "idx_user_id", Columns: []string{"user_id"}},
				},
			},
		},
	}
}

// ================== 学生课程表学校场景 ==================

// SchoolSchema 返回学校教育场景的数据库配置
// 包含学生、教师、课程、教室、课表、成绩等表
func SchoolSchema() any {
	return mysql.Database{
		"school": {
			// 学生表
			{
				Table: "students",
				Columns: []mysql.Column{
					{Column: "id", Type: mysql.Primary},
					{Column: "student_no", Type: mysql.Varchar},
					{Column: "name", Type: mysql.Varchar},
					{Column: "gender", Type: mysql.Tinyint},
					{Column: "date_of_birth", Type: mysql.Date},
					{Column: "class_id", Type: mysql.Bigint},
					{Column: "phone", Type: mysql.Varchar},
					{Column: "status", Type: mysql.Tinyint},
					{Column: "created_at", Type: mysql.Datetime},
					{Column: "updated_at", Type: mysql.TimestampUpdate},
				},
				Indexes: []mysql.Index{
					{Name: "pk_student_id", Columns: []string{"id"}, IsPrimary: true},
					{Name: "uk_student_no", Columns: []string{"student_no"}, IsUnique: true},
					{Name: "idx_class_id", Columns: []string{"class_id"}},
				},
			},
			// 教师表
			{
				Table: "teachers",
				Columns: []mysql.Column{
					{Column: "id", Type: mysql.Primary},
					{Column: "teacher_no", Type: mysql.Varchar},
					{Column: "name", Type: mysql.Varchar},
					{Column: "department_id", Type: mysql.Bigint},
					{Column: "specialty", Type: mysql.Varchar},
					{Column: "phone", Type: mysql.Varchar},
					{Column: "status", Type: mysql.Tinyint},
					{Column: "created_at", Type: mysql.Datetime},
				},
				Indexes: []mysql.Index{
					{Name: "pk_teacher_id", Columns: []string{"id"}, IsPrimary: true},
					{Name: "uk_teacher_no", Columns: []string{"teacher_no"}, IsUnique: true},
					{Name: "idx_department_id", Columns: []string{"department_id"}},
				},
			},
			// 课程表
			{
				Table: "courses",
				Columns: []mysql.Column{
					{Column: "id", Type: mysql.Primary},
					{Column: "course_no", Type: mysql.Varchar},
					{Column: "course_name", Type: mysql.Varchar},
					{Column: "teacher_id", Type: mysql.Bigint},
					{Column: "credits", Type: mysql.Decimal},
					{Column: "description", Type: mysql.Text},
					{Column: "status", Type: mysql.Tinyint},
					{Column: "created_at", Type: mysql.Datetime},
				},
				Indexes: []mysql.Index{
					{Name: "pk_course_id", Columns: []string{"id"}, IsPrimary: true},
					{Name: "uk_course_no", Columns: []string{"course_no"}, IsUnique: true},
					{Name: "idx_teacher_id", Columns: []string{"teacher_id"}},
				},
			},
			// 教室表
			{
				Table: "classrooms",
				Columns: []mysql.Column{
					{Column: "id", Type: mysql.Primary},
					{Column: "room_no", Type: mysql.Varchar},
					{Column: "building", Type: mysql.Varchar},
					{Column: "floor", Type: mysql.Tinyint},
					{Column: "capacity", Type: mysql.Int},
					{Column: "is_available", Type: mysql.Boolean},
					{Column: "created_at", Type: mysql.Datetime},
				},
				Indexes: []mysql.Index{
					{Name: "pk_classroom_id", Columns: []string{"id"}, IsPrimary: true},
					{Name: "uk_room_no", Columns: []string{"room_no"}, IsUnique: true},
				},
			},
			// 课程表（上课时间安排）
			{
				Table: "class_schedules",
				Columns: []mysql.Column{
					{Column: "id", Type: mysql.Primary},
					{Column: "course_id", Type: mysql.Bigint},
					{Column: "classroom_id", Type: mysql.Bigint},
					{Column: "day_of_week", Type: mysql.Tinyint},
					{Column: "start_time", Type: mysql.Time},
					{Column: "end_time", Type: mysql.Time},
					{Column: "start_date", Type: mysql.Date},
					{Column: "end_date", Type: mysql.Date},
					{Column: "status", Type: mysql.Tinyint},
				},
				Indexes: []mysql.Index{
					{Name: "pk_schedule_id", Columns: []string{"id"}, IsPrimary: true},
					{Name: "idx_course_id", Columns: []string{"course_id"}},
					{Name: "idx_classroom_id", Columns: []string{"classroom_id"}},
				},
			},
			// 成绩表
			{
				Table: "grades",
				Columns: []mysql.Column{
					{Column: "id", Type: mysql.Primary},
					{Column: "student_id", Type: mysql.Bigint},
					{Column: "course_id", Type: mysql.Bigint},
					{Column: "semester", Type: mysql.Varchar},
					{Column: "score", Type: mysql.Decimal},
					{Column: "gpa", Type: mysql.Decimal},
					{Column: "status", Type: mysql.Varchar},
					{Column: "recorded_at", Type: mysql.Datetime},
				},
				Indexes: []mysql.Index{
					{Name: "pk_grade_id", Columns: []string{"id"}, IsPrimary: true},
					{Name: "uk_student_course", Columns: []string{"student_id", "course_id", "semester"}, IsUnique: true},
					{Name: "idx_course_id", Columns: []string{"course_id"}},
				},
			},
		},
	}
}

// ================== 领导员工办公场景 ==================

// OfficeSchema 返回办公企业场景的数据库配置
// 包含部门、员工、角色、权限、日志、请假审批等表
func OfficeSchema() any {
	return mysql.Database{
		"office": {
			// 部门表
			{
				Table: "departments",
				Columns: []mysql.Column{
					{Column: "id", Type: mysql.Primary},
					{Column: "department_name", Type: mysql.Varchar},
					{Column: "parent_id", Type: mysql.Bigint},
					{Column: "manager_id", Type: mysql.Bigint},
					{Column: "description", Type: mysql.Text},
					{Column: "status", Type: mysql.Tinyint},
					{Column: "created_at", Type: mysql.Datetime},
					{Column: "updated_at", Type: mysql.TimestampUpdate},
				},
				Indexes: []mysql.Index{
					{Name: "pk_department_id", Columns: []string{"id"}, IsPrimary: true},
					{Name: "idx_parent_id", Columns: []string{"parent_id"}},
					{Name: "idx_manager_id", Columns: []string{"manager_id"}},
				},
			},
			// 员工表
			{
				Table: "employees",
				Columns: []mysql.Column{
					{Column: "id", Type: mysql.Primary},
					{Column: "employee_no", Type: mysql.Varchar},
					{Column: "name", Type: mysql.Varchar},
					{Column: "department_id", Type: mysql.Bigint},
					{Column: "position", Type: mysql.Varchar},
					{Column: "email", Type: mysql.Varchar},
					{Column: "phone", Type: mysql.Varchar},
					{Column: "gender", Type: mysql.Tinyint},
					{Column: "date_of_birth", Type: mysql.Date},
					{Column: "hire_date", Type: mysql.Date},
					{Column: "salary", Type: mysql.Decimal},
					{Column: "status", Type: mysql.Tinyint},
					{Column: "created_at", Type: mysql.Datetime},
					{Column: "updated_at", Type: mysql.TimestampUpdate},
				},
				Indexes: []mysql.Index{
					{Name: "pk_employee_id", Columns: []string{"id"}, IsPrimary: true},
					{Name: "uk_employee_no", Columns: []string{"employee_no"}, IsUnique: true},
					{Name: "idx_department_id", Columns: []string{"department_id"}},
					{Name: "idx_status", Columns: []string{"status"}},
				},
			},
			// 角色表
			{
				Table: "roles",
				Columns: []mysql.Column{
					{Column: "id", Type: mysql.Primary},
					{Column: "role_name", Type: mysql.Varchar},
					{Column: "description", Type: mysql.Text},
					{Column: "status", Type: mysql.Tinyint},
					{Column: "created_at", Type: mysql.Datetime},
				},
				Indexes: []mysql.Index{
					{Name: "pk_role_id", Columns: []string{"id"}, IsPrimary: true},
					{Name: "uk_role_name", Columns: []string{"role_name"}, IsUnique: true},
				},
			},
			// 员工角色关联表
			{
				Table: "employee_roles",
				Columns: []mysql.Column{
					{Column: "id", Type: mysql.Primary},
					{Column: "employee_id", Type: mysql.Bigint},
					{Column: "role_id", Type: mysql.Bigint},
					{Column: "assigned_at", Type: mysql.Datetime},
				},
				Indexes: []mysql.Index{
					{Name: "pk_id", Columns: []string{"id"}, IsPrimary: true},
					{Name: "uk_employee_role", Columns: []string{"employee_id", "role_id"}, IsUnique: true},
				},
			},
			// 权限表
			{
				Table: "permissions",
				Columns: []mysql.Column{
					{Column: "id", Type: mysql.Primary},
					{Column: "permission_name", Type: mysql.Varchar},
					{Column: "resource", Type: mysql.Varchar},
					{Column: "action", Type: mysql.Varchar},
					{Column: "description", Type: mysql.Text},
					{Column: "status", Type: mysql.Tinyint},
				},
				Indexes: []mysql.Index{
					{Name: "pk_permission_id", Columns: []string{"id"}, IsPrimary: true},
					{Name: "uk_permission_name", Columns: []string{"permission_name"}, IsUnique: true},
				},
			},
			// 角色权限关联表
			{
				Table: "role_permissions",
				Columns: []mysql.Column{
					{Column: "id", Type: mysql.Primary},
					{Column: "role_id", Type: mysql.Bigint},
					{Column: "permission_id", Type: mysql.Bigint},
				},
				Indexes: []mysql.Index{
					{Name: "pk_id", Columns: []string{"id"}, IsPrimary: true},
					{Name: "uk_role_permission", Columns: []string{"role_id", "permission_id"}, IsUnique: true},
				},
			},
			// 请假申请表
			{
				Table: "leave_requests",
				Columns: []mysql.Column{
					{Column: "id", Type: mysql.Primary},
					{Column: "employee_id", Type: mysql.Bigint},
					{Column: "leave_type", Type: mysql.Varchar},
					{Column: "start_date", Type: mysql.Datetime},
					{Column: "end_date", Type: mysql.Datetime},
					{Column: "reason", Type: mysql.Text},
					{Column: "status", Type: mysql.Tinyint},
					{Column: "approver_id", Type: mysql.Bigint},
					{Column: "approval_comment", Type: mysql.Text},
					{Column: "created_at", Type: mysql.Datetime},
					{Column: "updated_at", Type: mysql.TimestampUpdate},
				},
				Indexes: []mysql.Index{
					{Name: "pk_leave_id", Columns: []string{"id"}, IsPrimary: true},
					{Name: "idx_employee_id", Columns: []string{"employee_id"}},
					{Name: "idx_status", Columns: []string{"status"}},
				},
			},
			// 操作日志表
			{
				Table: "operation_logs",
				Columns: []mysql.Column{
					{Column: "id", Type: mysql.Primary},
					{Column: "employee_id", Type: mysql.Bigint},
					{Column: "operation", Type: mysql.Varchar},
					{Column: "resource", Type: mysql.Varchar},
					{Column: "resource_id", Type: mysql.Bigint},
					{Column: "old_value", Type: mysql.Text},
					{Column: "new_value", Type: mysql.Text},
					{Column: "ip_address", Type: mysql.Varchar},
					{Column: "created_at", Type: mysql.Datetime},
				},
				Indexes: []mysql.Index{
					{Name: "pk_log_id", Columns: []string{"id"}, IsPrimary: true},
					{Name: "idx_employee_id", Columns: []string{"employee_id"}},
					{Name: "idx_created_at", Columns: []string{"created_at"}},
				},
			},
		},
	}
}

// ================== 医生医院场景 ==================

// HospitalSchema 返回医疗健康场景的数据库配置
// 包含医生、患者、挂号、诊断、处方、药房等表
func HospitalSchema() any {
	return mysql.Database{
		"hospital": {
			// 科室表
			{
				Table: "departments",
				Columns: []mysql.Column{
					{Column: "id", Type: mysql.Primary},
					{Column: "department_name", Type: mysql.Varchar},
					{Column: "description", Type: mysql.Text},
					{Column: "status", Type: mysql.Tinyint},
					{Column: "created_at", Type: mysql.Datetime},
				},
				Indexes: []mysql.Index{
					{Name: "pk_department_id", Columns: []string{"id"}, IsPrimary: true},
				},
			},
			// 医生表
			{
				Table: "doctors",
				Columns: []mysql.Column{
					{Column: "id", Type: mysql.Primary},
					{Column: "doctor_no", Type: mysql.Varchar},
					{Column: "name", Type: mysql.Varchar},
					{Column: "department_id", Type: mysql.Bigint},
					{Column: "title", Type: mysql.Varchar},
					{Column: "specialization", Type: mysql.Varchar},
					{Column: "phone", Type: mysql.Varchar},
					{Column: "email", Type: mysql.Varchar},
					{Column: "experience_years", Type: mysql.Tinyint},
					{Column: "status", Type: mysql.Tinyint},
					{Column: "created_at", Type: mysql.Datetime},
				},
				Indexes: []mysql.Index{
					{Name: "pk_doctor_id", Columns: []string{"id"}, IsPrimary: true},
					{Name: "uk_doctor_no", Columns: []string{"doctor_no"}, IsUnique: true},
					{Name: "idx_department_id", Columns: []string{"department_id"}},
				},
			},
			// 患者表
			{
				Table: "patients",
				Columns: []mysql.Column{
					{Column: "id", Type: mysql.Primary},
					{Column: "patient_no", Type: mysql.Varchar},
					{Column: "name", Type: mysql.Varchar},
					{Column: "gender", Type: mysql.Tinyint},
					{Column: "date_of_birth", Type: mysql.Date},
					{Column: "phone", Type: mysql.Varchar},
					{Column: "id_card", Type: mysql.Varchar},
					{Column: "address", Type: mysql.VarcharLarge},
					{Column: "blood_type", Type: mysql.Varchar},
					{Column: "allergies", Type: mysql.Text},
					{Column: "created_at", Type: mysql.Datetime},
					{Column: "updated_at", Type: mysql.TimestampUpdate},
				},
				Indexes: []mysql.Index{
					{Name: "pk_patient_id", Columns: []string{"id"}, IsPrimary: true},
					{Name: "uk_patient_no", Columns: []string{"patient_no"}, IsUnique: true},
					{Name: "uk_id_card", Columns: []string{"id_card"}, IsUnique: true},
				},
			},
			// 挂号表
			{
				Table: "registrations",
				Columns: []mysql.Column{
					{Column: "id", Type: mysql.Primary},
					{Column: "patient_id", Type: mysql.Bigint},
					{Column: "doctor_id", Type: mysql.Bigint},
					{Column: "department_id", Type: mysql.Bigint},
					{Column: "registration_no", Type: mysql.Varchar},
					{Column: "registration_date", Type: mysql.Datetime},
					{Column: "visit_date", Type: mysql.Datetime},
					{Column: "fee", Type: mysql.Decimal},
					{Column: "status", Type: mysql.Tinyint},
					{Column: "created_at", Type: mysql.Datetime},
				},
				Indexes: []mysql.Index{
					{Name: "pk_registration_id", Columns: []string{"id"}, IsPrimary: true},
					{Name: "uk_registration_no", Columns: []string{"registration_no"}, IsUnique: true},
					{Name: "idx_patient_id", Columns: []string{"patient_id"}},
					{Name: "idx_doctor_id", Columns: []string{"doctor_id"}},
				},
			},
			// 诊断记录表
			{
				Table: "diagnoses",
				Columns: []mysql.Column{
					{Column: "id", Type: mysql.Primary},
					{Column: "registration_id", Type: mysql.Bigint},
					{Column: "patient_id", Type: mysql.Bigint},
					{Column: "doctor_id", Type: mysql.Bigint},
					{Column: "chief_complaint", Type: mysql.VarcharLarge},
					{Column: "present_illness", Type: mysql.Text},
					{Column: "past_medical_history", Type: mysql.Text},
					{Column: "physical_examination", Type: mysql.Text},
					{Column: "diagnosis_result", Type: mysql.Text},
					{Column: "treatment_plan", Type: mysql.Text},
					{Column: "notes", Type: mysql.Text},
					{Column: "diagnosis_date", Type: mysql.Datetime},
				},
				Indexes: []mysql.Index{
					{Name: "pk_diagnosis_id", Columns: []string{"id"}, IsPrimary: true},
					{Name: "idx_patient_id", Columns: []string{"patient_id"}},
					{Name: "idx_doctor_id", Columns: []string{"doctor_id"}},
				},
			},
			// 处方表
			{
				Table: "prescriptions",
				Columns: []mysql.Column{
					{Column: "id", Type: mysql.Primary},
					{Column: "diagnosis_id", Type: mysql.Bigint},
					{Column: "patient_id", Type: mysql.Bigint},
					{Column: "doctor_id", Type: mysql.Bigint},
					{Column: "prescription_no", Type: mysql.Varchar},
					{Column: "total_fee", Type: mysql.Decimal},
					{Column: "status", Type: mysql.Tinyint},
					{Column: "prescribed_date", Type: mysql.Datetime},
					{Column: "execution_date", Type: mysql.Datetime},
				},
				Indexes: []mysql.Index{
					{Name: "pk_prescription_id", Columns: []string{"id"}, IsPrimary: true},
					{Name: "uk_prescription_no", Columns: []string{"prescription_no"}, IsUnique: true},
					{Name: "idx_patient_id", Columns: []string{"patient_id"}},
					{Name: "idx_doctor_id", Columns: []string{"doctor_id"}},
				},
			},
			// 处方药物详情表
			{
				Table: "prescription_items",
				Columns: []mysql.Column{
					{Column: "id", Type: mysql.Primary},
					{Column: "prescription_id", Type: mysql.Bigint},
					{Column: "medicine_id", Type: mysql.Bigint},
					{Column: "quantity", Type: mysql.Decimal},
					{Column: "unit", Type: mysql.Varchar},
					{Column: "usage", Type: mysql.VarcharLarge},
					{Column: "dosage", Type: mysql.VarcharLarge},
					{Column: "frequency", Type: mysql.Varchar},
					{Column: "duration_days", Type: mysql.Tinyint},
					{Column: "unit_price", Type: mysql.Decimal},
					{Column: "subtotal", Type: mysql.Decimal},
				},
				Indexes: []mysql.Index{
					{Name: "pk_item_id", Columns: []string{"id"}, IsPrimary: true},
					{Name: "idx_prescription_id", Columns: []string{"prescription_id"}},
					{Name: "idx_medicine_id", Columns: []string{"medicine_id"}},
				},
			},
			// 药品表
			{
				Table: "medicines",
				Columns: []mysql.Column{
					{Column: "id", Type: mysql.Primary},
					{Column: "medicine_name", Type: mysql.Varchar},
					{Column: "generic_name", Type: mysql.Varchar},
					{Column: "category", Type: mysql.Varchar},
					{Column: "price", Type: mysql.Decimal},
					{Column: "manufacturer", Type: mysql.Varchar},
					{Column: "specification", Type: mysql.Varchar},
					{Column: "side_effects", Type: mysql.Text},
					{Column: "contraindications", Type: mysql.Text},
					{Column: "storage_conditions", Type: mysql.Varchar},
					{Column: "expiry_date", Type: mysql.Date},
					{Column: "status", Type: mysql.Tinyint},
				},
				Indexes: []mysql.Index{
					{Name: "pk_medicine_id", Columns: []string{"id"}, IsPrimary: true},
					{Name: "idx_category", Columns: []string{"category"}},
				},
			},
			// 医疗记录表
			{
				Table: "medical_records",
				Columns: []mysql.Column{
					{Column: "id", Type: mysql.Primary},
					{Column: "patient_id", Type: mysql.Bigint},
					{Column: "diagnosis_id", Type: mysql.Bigint},
					{Column: "test_name", Type: mysql.Varchar},
					{Column: "test_type", Type: mysql.Varchar},
					{Column: "test_date", Type: mysql.Datetime},
					{Column: "result", Type: mysql.Text},
					{Column: "normal_range", Type: mysql.Varchar},
					{Column: "doctor_notes", Type: mysql.Text},
					{Column: "created_at", Type: mysql.Datetime},
				},
				Indexes: []mysql.Index{
					{Name: "pk_record_id", Columns: []string{"id"}, IsPrimary: true},
					{Name: "idx_patient_id", Columns: []string{"patient_id"}},
					{Name: "idx_test_date", Columns: []string{"test_date"}},
				},
			},
		},
	}
}
