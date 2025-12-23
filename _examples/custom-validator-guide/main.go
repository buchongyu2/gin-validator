package main

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"regexp"

	"github.com/go-playground/validator/v10"
)

// Pre-compiled regular expressions for better performance
var (
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	phoneRegex    = regexp.MustCompile(`^1[3-9]\d{9}$`)
)

// UserStatus represents user status enumeration
type UserStatus int

const (
	UserStatusInactive UserStatus = 0
	UserStatusActive   UserStatus = 1
	UserStatusBanned   UserStatus = 2
)

// User struct demonstrates three custom validation approaches
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

	// 1. Register field-level custom validation
	err := validate.RegisterValidation("username_format", validateUsernameFormat)
	if err != nil {
		fmt.Printf("Failed to register validation: %v\n", err)
		return
	}
	err = validate.RegisterValidation("phone_format", validatePhoneFormat)
	if err != nil {
		fmt.Printf("Failed to register validation: %v\n", err)
		return
	}

	// 2. Register custom type function
	validate.RegisterCustomTypeFunc(ValidateValuer, sql.NullString{})

	// 3. Register struct-level validation
	validate.RegisterStructValidation(UserStructValidation, User{})

	// Test case 1: Valid user
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

	// Test case 2: Invalid username format
	invalidUser := User{
		Username:  "张三", // Contains Chinese characters, doesn't match format
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

	// Test case 3: Missing name
	noNameUser := User{
		Username: "test_user",
		Email:    "test@example.com",
		Age:      25,
		Status:   UserStatusActive,
		Phone:    "13800138000",
		// FirstName and LastName are both empty
	}

	fmt.Println("\n=== Test 3: Missing Name ===")
	if err := validate.Struct(noNameUser); err != nil {
		fmt.Printf("✓ Expected validation failure: %v\n", err)
	}

	// Test case 4: Invalid phone number
	invalidPhoneUser := User{
		Username:  "test_user",
		Email:     "test@example.com",
		Age:       25,
		Status:    UserStatusActive,
		Phone:     "12345", // Invalid phone number
		FirstName: "San",
		LastName:  "Zhang",
	}

	fmt.Println("\n=== Test 4: Invalid Phone Number ===")
	if err := validate.Struct(invalidPhoneUser); err != nil {
		fmt.Printf("✓ Expected validation failure: %v\n", err)
	}

	// Test case 5: Using NullString
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

// validateUsernameFormat is a field-level custom validation for username format
// Username can only contain letters, numbers and underscores
func validateUsernameFormat(fl validator.FieldLevel) bool {
	username := fl.Field().String()
	return usernameRegex.MatchString(username)
}

// validatePhoneFormat is a field-level custom validation for phone format
// Simple Chinese phone number validation (11 digits starting with 13-19)
func validatePhoneFormat(fl validator.FieldLevel) bool {
	phone := fl.Field().String()
	return phoneRegex.MatchString(phone)
}

// ValidateValuer is a custom type function that handles sql.Null* types
func ValidateValuer(field reflect.Value) interface{} {
	if valuer, ok := field.Interface().(driver.Valuer); ok {
		val, err := valuer.Value()
		if err == nil {
			return val
		}
	}
	return nil
}

// UserStructValidation is a struct-level validation that ensures at least one name exists
func UserStructValidation(sl validator.StructLevel) {
	user := sl.Current().Interface().(User)

	// Validate that either FirstName or LastName must exist
	if len(user.FirstName) == 0 && len(user.LastName) == 0 {
		sl.ReportError(user.FirstName, "first_name", "FirstName", "require_name", "")
		sl.ReportError(user.LastName, "last_name", "LastName", "require_name", "")
	}
}
