package main

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"regexp"

	"github.com/go-playground/validator/v10"
)

// UserStatus 用户状态枚举
type UserStatus int

const (
	UserStatusInactive UserStatus = 0
	UserStatusActive   UserStatus = 1
	UserStatusBanned   UserStatus = 2
)

// User 用户结构体，演示三种自定义验证方式
type User struct {
	Username  string         `validate:"required,min=3,max=20,username_format"`
	Email     string         `validate:"required,email"`
	Age       int            `validate:"required,gte=18,lte=100"`
	Status    UserStatus     `validate:"required"`
	Phone     string         `validate:"required,phone_format"`
	NickName  sql.NullString `validate:"omitempty"`
	FirstName string         `json:"first_name"`
	LastName  string         `json:"last_name"`
}

var validate *validator.Validate

func main() {
	validate = validator.New()

	// 1. 注册字段级别的自定义验证
	err := validate.RegisterValidation("username_format", validateUsernameFormat)
	if err != nil {
		fmt.Printf("注册验证失败: %v\n", err)
		return
	}
	err = validate.RegisterValidation("phone_format", validatePhoneFormat)
	if err != nil {
		fmt.Printf("注册验证失败: %v\n", err)
		return
	}

	// 2. 注册自定义类型函数
	validate.RegisterCustomTypeFunc(ValidateValuer, sql.NullString{})

	// 3. 注册结构体级别验证
	validate.RegisterStructValidation(UserStructValidation, User{})

	// 测试用例 1: 有效的用户
	validUser := User{
		Username:  "zhang_san",
		Email:     "zhangsan@example.com",
		Age:       25,
		Status:    UserStatusActive,
		Phone:     "13800138000",
		FirstName: "San",
		LastName:  "Zhang",
	}

	fmt.Println("=== Test 1: Valid User ===")
	if err := validate.Struct(validUser); err != nil {
		fmt.Printf("Validation failed: %v\n", err)
	} else {
		fmt.Println("✓ Validation passed")
	}

	// 测试用例 2: 无效的用户名格式
	invalidUser := User{
		Username:  "张三", // 包含中文，不符合格式
		Email:     "zhangsan@example.com",
		Age:       25,
		Status:    UserStatusActive,
		Phone:     "13800138000",
		FirstName: "San",
		LastName:  "Zhang",
	}

	fmt.Println("\n=== Test 2: Invalid Username Format ===")
	if err := validate.Struct(invalidUser); err != nil {
		fmt.Printf("✓ Expected validation failure: %v\n", err)
	}

	// 测试用例 3: 缺少姓名
	noNameUser := User{
		Username: "test_user",
		Email:    "test@example.com",
		Age:      25,
		Status:   UserStatusActive,
		Phone:    "13800138000",
		// FirstName 和 LastName 都为空
	}

	fmt.Println("\n=== Test 3: Missing Name ===")
	if err := validate.Struct(noNameUser); err != nil {
		fmt.Printf("✓ Expected validation failure: %v\n", err)
	}

	// 测试用例 4: 无效的电话号码
	invalidPhoneUser := User{
		Username:  "test_user",
		Email:     "test@example.com",
		Age:       25,
		Status:    UserStatusActive,
		Phone:     "12345", // 无效的电话号码
		FirstName: "San",
		LastName:  "Zhang",
	}

	fmt.Println("\n=== Test 4: Invalid Phone Number ===")
	if err := validate.Struct(invalidPhoneUser); err != nil {
		fmt.Printf("✓ Expected validation failure: %v\n", err)
	}

	// 测试用例 5: 使用 NullString
	nullStringUser := User{
		Username:  "test_user",
		Email:     "test@example.com",
		Age:       25,
		Status:    UserStatusActive,
		Phone:     "13800138000",
		NickName:  sql.NullString{String: "TestNick", Valid: true},
		FirstName: "San",
		LastName:  "Zhang",
	}

	fmt.Println("\n=== Test 5: With NullString ===")
	if err := validate.Struct(nullStringUser); err != nil {
		fmt.Printf("Validation failed: %v\n", err)
	} else {
		fmt.Println("✓ Validation passed")
	}
}

// validateUsernameFormat 字段级别自定义验证：用户名格式
// 用户名只能包含字母、数字和下划线
func validateUsernameFormat(fl validator.FieldLevel) bool {
	username := fl.Field().String()
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_]+$`, username)
	return matched
}

// validatePhoneFormat 字段级别自定义验证：手机号格式
// 简单的中国手机号验证（以 13-19 开头的 11 位数字）
func validatePhoneFormat(fl validator.FieldLevel) bool {
	phone := fl.Field().String()
	matched, _ := regexp.MatchString(`^1[3-9]\d{9}$`, phone)
	return matched
}

// ValidateValuer 自定义类型函数：处理 sql.Null* 类型
func ValidateValuer(field reflect.Value) interface{} {
	if valuer, ok := field.Interface().(driver.Valuer); ok {
		val, err := valuer.Value()
		if err == nil {
			return val
		}
	}
	return nil
}

// UserStructValidation 结构体级别验证：确保至少有一个名字
func UserStructValidation(sl validator.StructLevel) {
	user := sl.Current().Interface().(User)

	// 验证必须有 FirstName 或 LastName 其中之一
	if len(user.FirstName) == 0 && len(user.LastName) == 0 {
		sl.ReportError(user.FirstName, "first_name", "FirstName", "require_name", "")
		sl.ReportError(user.LastName, "last_name", "LastName", "require_name", "")
	}
}
