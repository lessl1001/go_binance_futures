package middlewares

import (
	"encoding/json"
	"go_binance_futures/models"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web/context"
)

// APILoggingMiddleware API访问日志中间件
func APILoggingMiddleware(ctx *context.Context) {
	startTime := time.Now()
	
	// 记录API访问日志
	duration := time.Since(startTime)
	
	// 只记录API相关的访问日志
	if ctx.Request.URL.Path != "" && len(ctx.Request.URL.Path) > 4 && ctx.Request.URL.Path[:4] == "/api" {
		go logAPIAccess(ctx, duration)
	}
}

// 记录API访问日志
func logAPIAccess(ctx *context.Context, duration time.Duration) {
	defer func() {
		if r := recover(); r != nil {
			logs.Error("API logging middleware panic:", r)
		}
	}()
	
	// 获取请求信息
	method := ctx.Request.Method
	url := ctx.Request.URL.Path
	userAgent := ctx.Request.UserAgent()
	ip := ctx.Input.IP()
	
	// 构建日志详情
	details := map[string]interface{}{
		"method":      method,
		"url":         url,
		"user_agent":  userAgent,
		"duration_ms": duration.Milliseconds(),
	}
	
	// 记录请求体（仅对POST、PUT请求）
	if method == "POST" || method == "PUT" {
		if ctx.Input.RequestBody != nil && len(ctx.Input.RequestBody) > 0 {
			// 限制记录的请求体大小
			if len(ctx.Input.RequestBody) < 1024 {
				details["request_body"] = string(ctx.Input.RequestBody)
			} else {
				details["request_body"] = "REQUEST_BODY_TOO_LARGE"
			}
		}
	}
	
	// 获取用户信息（如果有认证信息）
	userID := "anonymous"
	if userData := ctx.Input.GetData("user"); userData != nil {
		userID = "admin" // 简化处理
	}
	
	// 记录到数据库
	o := orm.NewOrm()
	detailsBytes, _ := json.Marshal(details)
	
	log := models.OperationLog{
		UserID:       userID,
		Operation:    "API_ACCESS",
		ResourceType: "api_endpoint",
		ResourceID:   url,
		Details:      string(detailsBytes),
		IPAddress:    ip,
		UserAgent:    userAgent,
		CreateTime:   time.Now().Unix(),
	}
	
	if _, err := o.Insert(&log); err != nil {
		logs.Error("Failed to log API access:", err)
	}
}

// PermissionMiddleware 权限校验中间件
func PermissionMiddleware(ctx *context.Context) {
	// 获取请求路径
	url := ctx.Request.URL.Path
	method := ctx.Request.Method
	
	// 检查是否需要特殊权限的API
	if needsPermission(url, method) {
		if !checkUserPermission(ctx, url, method) {
			ctx.Output.SetStatus(403)
			ctx.Output.JSON(map[string]interface{}{
				"status":  "error",
				"message": "Permission denied: insufficient privileges",
			}, true, false)
			return
		}
	}
}

// 检查是否需要特殊权限
func needsPermission(url, method string) bool {
	// 需要特殊权限的API路径
	restrictedPaths := []string{
		"/api/deploy_strategy",
		"/api/operation_logs",
	}
	
	for _, path := range restrictedPaths {
		if len(url) >= len(path) && url[:len(path)] == path {
			return true
		}
	}
	
	return false
}

// 检查用户权限
func checkUserPermission(ctx *context.Context, url, method string) bool {
	// 获取用户信息
	userID := "anonymous"
	if userData := ctx.Input.GetData("user"); userData != nil {
		userID = "admin" // 简化处理
	}
	
	// 简化权限检查：admin用户有所有权限
	if userID == "admin" {
		return true
	}
	
	// 其他用户权限检查逻辑
	// 这里可以根据实际需求实现更复杂的权限检查
	// 例如：基于角色的权限控制(RBAC)
	
	return false
}

// RateLimitMiddleware 速率限制中间件
func RateLimitMiddleware(ctx *context.Context) {
	ip := ctx.Input.IP()
	
	// 检查是否超过速率限制
	if isRateLimited(ip) {
		ctx.Output.SetStatus(429)
		ctx.Output.JSON(map[string]interface{}{
			"status":  "error",
			"message": "Rate limit exceeded. Please try again later.",
		}, true, false)
		return
	}
}

// 检查速率限制
func isRateLimited(ip string) bool {
	// 这里可以实现基于内存或Redis的速率限制
	// 简化实现：暂时返回false
	return false
}