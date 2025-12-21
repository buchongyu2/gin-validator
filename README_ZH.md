# 自定义验证器指南

本文档介绍如何基于 validator 库创建自定义验证器。

## 目录

- [概述](#概述)
- [方法一：注册字段级别的自定义验证函数](#方法一注册字段级别的自定义验证函数)
- [方法二：注册自定义类型函数](#方法二注册自定义类型函数)
- [方法三：注册结构体级别验证](#方法三注册结构体级别验证)
- [完整示例](#完整示例)

## 概述

validator 提供了三种主要方式来实现自定义验证：

1. **字段级别自定义验证** - 使用 `RegisterValidation` 注册自定义验证标签
2. **自定义类型处理** - 使用 `RegisterCustomTypeFunc` 处理特殊类型
3. **结构体级别验证** - 使用 `RegisterStructValidation` 实现跨字段验证

## 方法一：注册字段级别的自定义验证函数

这是最常用的方法，允许你创建自定义验证标签。

### 基本用法

```go
package main

import (
    "fmt"
    "github.com/go-playground/validator/v10"
)

type User struct {
    Username string `validate:"required,is-awesome"`
}

func main() {
    validate := validator.New()
    
    // 注册自定义验证函数
    validate.RegisterValidation("is-awesome", validateIsAwesome)
    
    user := User{Username: "awesome"}
    err := validate.Struct(user)
    if err != nil {
        fmt.Printf("验证失败: %v\n", err)
    } else {
        fmt.Println("验证成功！")
    }
}

// 自定义验证函数
func validateIsAwesome(fl validator.FieldLevel) bool {
    return fl.Field().String() == "awesome"
}
```

### 带参数的自定义验证

```go
import "strconv"

type Product struct {
    Name  string `validate:"required"`
    Price int    `validate:"required,min_price=100"`
}

func main() {
    validate := validator.New()
    
    // 注册带参数的验证函数
    validate.RegisterValidation("min_price", validateMinPrice)
    
    product := Product{Name: "商品", Price: 50}
    err := validate.Struct(product)
    if err != nil {
        fmt.Printf("验证失败: %v\n", err)
    }
}

func validateMinPrice(fl validator.FieldLevel) bool {
    price := fl.Field().Int()
    minPriceStr := fl.Param() // 获取参数值，例如 "100"
    
    // 将参数转换为整数
    minPrice, err := strconv.ParseInt(minPriceStr, 10, 64)
    if err != nil {
        return false // 参数格式错误
    }
    
    return price >= minPrice
}
```

### 使用上下文的验证函数

```go
import "context"

func main() {
    validate := validator.New()
    
    // 注册支持 context 的验证函数
    validate.RegisterValidationCtx("custom_with_ctx", validateWithContext)
    
    // 使用带 context 的验证
    ctx := context.Background()
    err := validate.StructCtx(ctx, myStruct)
}

func validateWithContext(ctx context.Context, fl validator.FieldLevel) bool {
    // 可以从 context 中获取额外信息
    // 例如：数据库连接、配置等
    return true
}
```

## 方法二：注册自定义类型函数

用于处理特殊类型，如 `sql.NullString`、自定义枚举等。

### 处理数据库空值类型

```go
import (
    "database/sql"
    "database/sql/driver"
    "reflect"
)

type User struct {
    Name sql.NullString `validate:"required"`
    Age  sql.NullInt64  `validate:"required,gte=0"`
}

func main() {
    validate := validator.New()
    
    // 注册自定义类型函数
    validate.RegisterCustomTypeFunc(ValidateValuer, 
        sql.NullString{}, 
        sql.NullInt64{}, 
        sql.NullBool{},
    )
    
    user := User{
        Name: sql.NullString{String: "张三", Valid: true},
        Age:  sql.NullInt64{Int64: 25, Valid: true},
    }
    
    err := validate.Struct(user)
    if err != nil {
        fmt.Printf("验证失败: %v\n", err)
    }
}

// 自定义类型函数
func ValidateValuer(field reflect.Value) interface{} {
    if valuer, ok := field.Interface().(driver.Valuer); ok {
        val, err := valuer.Value()
        if err == nil {
            return val
        }
    }
    return nil
}
```

### 处理自定义枚举类型

```go
type Status int

const (
    StatusPending Status = iota
    StatusActive
    StatusInactive
)

type Order struct {
    ID     int    `validate:"required"`
    Status Status `validate:"required,oneof=0 1 2"`
}

func main() {
    validate := validator.New()
    
    // 注册自定义类型函数处理枚举
    validate.RegisterCustomTypeFunc(ValidateStatus, Status(0))
    
    order := Order{ID: 1, Status: StatusActive}
    err := validate.Struct(order)
    if err != nil {
        fmt.Printf("验证失败: %v\n", err)
    }
}

func ValidateStatus(field reflect.Value) interface{} {
    if status, ok := field.Interface().(Status); ok {
        return int(status)
    }
    return nil
}
```

## 方法三：注册结构体级别验证

用于实现跨字段验证逻辑。

### 基本用法

```go
type User struct {
    FirstName string `json:"first_name"`
    LastName  string `json:"last_name"`
    Email     string `validate:"required,email"`
}

func main() {
    validate := validator.New()
    
    // 注册结构体级别验证
    validate.RegisterStructValidation(UserStructLevelValidation, User{})
    
    user := User{
        FirstName: "",
        LastName:  "",
        Email:     "test@example.com",
    }
    
    err := validate.Struct(user)
    if err != nil {
        fmt.Printf("验证失败: %v\n", err)
    }
}

// 结构体级别验证函数
func UserStructLevelValidation(sl validator.StructLevel) {
    user := sl.Current().Interface().(User)
    
    // 验证至少有一个名字存在
    if len(user.FirstName) == 0 && len(user.LastName) == 0 {
        sl.ReportError(user.FirstName, "first_name", "FirstName", "fname_or_lname", "")
        sl.ReportError(user.LastName, "last_name", "LastName", "fname_or_lname", "")
    }
}
```

### 跨字段验证

```go
type DateRange struct {
    StartDate time.Time `validate:"required"`
    EndDate   time.Time `validate:"required"`
}

func main() {
    validate := validator.New()
    
    validate.RegisterStructValidation(DateRangeValidation, DateRange{})
    
    dateRange := DateRange{
        StartDate: time.Now(),
        EndDate:   time.Now().Add(-24 * time.Hour), // 结束日期早于开始日期
    }
    
    err := validate.Struct(dateRange)
    if err != nil {
        fmt.Printf("验证失败: %v\n", err)
    }
}

func DateRangeValidation(sl validator.StructLevel) {
    dateRange := sl.Current().Interface().(DateRange)
    
    // 验证结束日期必须晚于开始日期
    if dateRange.EndDate.Before(dateRange.StartDate) {
        sl.ReportError(dateRange.EndDate, "end_date", "EndDate", "gtefield_start", "")
    }
}
```

## 完整示例

下面是一个综合使用所有三种方法的完整示例：

```go
package main

import (
    "database/sql"
    "database/sql/driver"
    "fmt"
    "reflect"
    "regexp"
    
    "github.com/go-playground/validator/v10"
)

// 预编译正则表达式以提高性能
var (
    usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
    phoneRegex    = regexp.MustCompile(`^1[3-9]\d{9}$`)
)

// 用户状态枚举
type UserStatus int

const (
    UserStatusInactive UserStatus = 0
    UserStatusActive   UserStatus = 1
    UserStatusBanned   UserStatus = 2
)

// 用户结构体
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
    validate.RegisterValidation("username_format", validateUsernameFormat)
    validate.RegisterValidation("phone_format", validatePhoneFormat)
    
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
        FirstName: "三",
        LastName:  "张",
    }
    
    fmt.Println("=== 测试有效用户 ===")
    if err := validate.Struct(validUser); err != nil {
        fmt.Printf("验证失败: %v\n", err)
    } else {
        fmt.Println("✓ 验证成功")
    }
    
    // 测试用例 2: 无效的用户名格式
    invalidUser := User{
        Username:  "张三",  // 包含中文，不符合格式
        Email:     "zhangsan@example.com",
        Age:       25,
        Status:    UserStatusActive,
        Phone:     "13800138000",
        FirstName: "三",
        LastName:  "张",
    }
    
    fmt.Println("\n=== 测试无效用户名 ===")
    if err := validate.Struct(invalidUser); err != nil {
        fmt.Printf("✓ 预期的验证失败: %v\n", err)
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
    
    fmt.Println("\n=== 测试缺少姓名 ===")
    if err := validate.Struct(noNameUser); err != nil {
        fmt.Printf("✓ 预期的验证失败: %v\n", err)
    }
}

// 1. 字段级别自定义验证：用户名格式
// 用户名只能包含字母、数字和下划线
func validateUsernameFormat(fl validator.FieldLevel) bool {
    username := fl.Field().String()
    return usernameRegex.MatchString(username)
}

// 1. 字段级别自定义验证：手机号格式
// 简单的中国手机号验证（以 13-19 开头的 11 位数字）
func validatePhoneFormat(fl validator.FieldLevel) bool {
    phone := fl.Field().String()
    return phoneRegex.MatchString(phone)
}

// 2. 自定义类型函数：处理 sql.Null* 类型
func ValidateValuer(field reflect.Value) interface{} {
    if valuer, ok := field.Interface().(driver.Valuer); ok {
        val, err := valuer.Value()
        if err == nil {
            return val
        }
    }
    return nil
}

// 3. 结构体级别验证：确保至少有一个名字
func UserStructValidation(sl validator.StructLevel) {
    user := sl.Current().Interface().(User)
    
    // 验证必须有 FirstName 或 LastName 其中之一
    if len(user.FirstName) == 0 && len(user.LastName) == 0 {
        sl.ReportError(user.FirstName, "first_name", "FirstName", "require_name", "")
        sl.ReportError(user.LastName, "last_name", "LastName", "require_name", "")
    }
}
```

## 最佳实践

1. **使用单例模式**: 创建一个全局的 `validator.Validate` 实例，它会缓存结构体信息以提高性能。

2. **在初始化时注册**: 在应用启动时注册所有自定义验证，而不是在运行时注册。

3. **验证函数要简单**: 验证函数应该专注于单一职责，复杂的业务逻辑应该放在其他地方。

4. **错误处理**: 总是检查验证返回的错误，并提供用户友好的错误信息。

5. **性能考虑**: 
   - 避免在验证函数中进行耗时操作（如数据库查询）
   - 如果需要外部资源，考虑使用带 context 的验证函数

6. **测试**: 为自定义验证编写单元测试，确保边界情况得到处理。

## 常见问题

### Q: 如何在验证函数中访问其他字段？

A: 使用 `fl.Parent()` 可以访问父结构体：

```go
func validateField(fl validator.FieldLevel) bool {
    parent := fl.Parent()
    // 访问其他字段
    otherField := parent.FieldByName("OtherField")
    return true
}
```

### Q: 如何让验证在字段为 nil 时也执行？

A: 使用 `RegisterValidationCtx` 的第三个参数：

```go
validate.RegisterValidationCtx("custom", myFunc, true) // true 表示即使为 nil 也执行
```

### Q: 如何组合多个验证？

A: 在标签中使用逗号分隔：

```go
type User struct {
    Email string `validate:"required,email,custom_email"`
}
```

### Q: 自定义验证和内置验证的执行顺序？

A: 按照标签中定义的顺序从左到右执行。

## 参考资源

- [官方文档](https://pkg.go.dev/github.com/go-playground/validator/v10)
- [更多示例](https://github.com/go-playground/validator/tree/master/_examples)
- [英文 README](./README.md)

## 许可证

本项目使用 MIT 许可证，详见 [LICENSE](./LICENSE) 文件。
