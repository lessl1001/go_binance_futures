package utils

import (
	"encoding/json"
	"go_binance_futures/models"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web/context"
	"github.com/golang-jwt/jwt/v5"
)

// 操作日志服务
type OperationLogService struct {
	Orm orm.Ormer
}

func NewOperationLogService() *OperationLogService {
	return &OperationLogService{
		Orm: orm.NewOrm(),
	}
}

// 记录操作日志
func (ols *OperationLogService) LogOperation(ctx *context.Context, operation, resource, resourceID string, details interface{}) error {
	// 获取用户信息
	userName := "system"
	if user, ok := ctx.Input.GetData("user").(jwt.MapClaims); ok {
		if username, exists := user["Username"].(string); exists {
			userName = username
		}
	}
	
	// 序列化详情
	detailsJson, err := json.Marshal(details)
	if err != nil {
		logs.Error("序列化操作详情失败:", err)
		detailsJson = []byte("{}")
	}
	
	// 获取客户端IP
	clientIP := ctx.Input.IP()
	
	// 获取用户代理
	userAgent := ctx.Request.UserAgent()
	
	// 创建操作日志
	log := &models.OperationLog{
		UserName:   userName,
		Operation:  operation,
		Resource:   resource,
		ResourceID: resourceID,
		Details:    string(detailsJson),
		IPAddress:  clientIP,
		UserAgent:  userAgent,
		CreatedAt:  time.Now().Unix(),
	}
	
	_, err = ols.Orm.Insert(log)
	if err != nil {
		logs.Error("记录操作日志失败:", err)
		return err
	}
	
	logs.Info("操作日志记录成功: 用户=%s, 操作=%s, 资源=%s, 资源ID=%s", userName, operation, resource, resourceID)
	return nil
}

// 获取操作日志列表
func (ols *OperationLogService) GetOperationLogs(page, pageSize int, resource, userName string) ([]models.OperationLog, int64, error) {
	var logs []models.OperationLog
	qs := ols.Orm.QueryTable("operation_logs")
	
	// 过滤条件
	if resource != "" {
		qs = qs.Filter("resource", resource)
	}
	if userName != "" {
		qs = qs.Filter("user_name__icontains", userName)
	}
	
	// 获取总数
	total, err := qs.Count()
	if err != nil {
		return nil, 0, err
	}
	
	// 分页查询
	offset := (page - 1) * pageSize
	_, err = qs.OrderBy("-created_at").Limit(pageSize, offset).All(&logs)
	if err != nil {
		return nil, 0, err
	}
	
	return logs, total, nil
}

// 权限验证辅助函数
func (ols *OperationLogService) HasPermission(ctx *context.Context, operation string) bool {
	// 检查用户是否已认证
	user, ok := ctx.Input.GetData("user").(jwt.MapClaims)
	if !ok {
		return false
	}
	
	// 获取用户名
	username, exists := user["Username"].(string)
	if !exists {
		return false
	}
	
	// 简单的权限检查 - 这里可以根据实际需求扩展更复杂的权限系统
	// 对于策略风控的敏感操作，需要admin权限
	sensitiveOperations := []string{"unfreeze", "reset_loss", "delete"}
	for _, op := range sensitiveOperations {
		if operation == op {
			// 只有admin用户才能执行敏感操作
			return username == "admin"
		}
	}
	
	// 其他操作允许所有已认证用户执行
	return true
}