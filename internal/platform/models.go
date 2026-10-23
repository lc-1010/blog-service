package platform

import (
	"blog-server/internal/model"
	"encoding/json"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	PermissionContentManage  = "content.manage"
	PermissionUsersManage    = "users.manage"
	PermissionRolesManage    = "roles.manage"
	PermissionSettingsManage = "settings.manage"
	PermissionSitesManage    = "sites.manage"
	PermissionAuditRead      = "audit.read"
)

type Role struct {
	model.Model
	Name        string `json:"name" gorm:"size:80;uniqueIndex"`
	Description string `json:"description" gorm:"size:255"`
	Permissions string `json:"-" gorm:"type:text"`
}

func (Role) TableName() string { return "blog_admin_roles" }

func (r Role) PermissionList() []string {
	var permissions []string
	if err := json.Unmarshal([]byte(r.Permissions), &permissions); err != nil {
		return nil
	}
	return permissions
}

func (r *Role) SetPermissions(permissions []string) error {
	data, err := json.Marshal(permissions)
	if err != nil {
		return err
	}
	r.Permissions = string(data)
	return nil
}

type User struct {
	model.Model
	Username     string `json:"username" gorm:"size:100;uniqueIndex"`
	DisplayName  string `json:"display_name" gorm:"size:100"`
	PasswordHash string `json:"-" gorm:"size:100"`
	RoleID       uint32 `json:"role_id" gorm:"index"`
	State        uint8  `json:"state" gorm:"default:1"`
}

func (User) TableName() string { return "blog_admin_users" }

type Setting struct {
	model.Model
	Key   string `json:"key" gorm:"column:setting_key;size:100;uniqueIndex"`
	Value string `json:"value" gorm:"type:text"`
}

func (Setting) TableName() string { return "blog_platform_settings" }

type Site struct {
	model.Model
	Name        string `json:"name" gorm:"size:120"`
	Domain      string `json:"domain" gorm:"size:255;uniqueIndex"`
	Description string `json:"description" gorm:"size:255"`
	State       uint8  `json:"state" gorm:"default:1"`
}

func (Site) TableName() string { return "blog_platform_sites" }

type AuditLog struct {
	model.Model
	Username string `json:"username" gorm:"size:100;index"`
	Action   string `json:"action" gorm:"size:120;index"`
	Resource string `json:"resource" gorm:"size:255"`
	Method   string `json:"method" gorm:"size:10"`
	Status   int    `json:"status"`
	RemoteIP string `json:"remote_ip" gorm:"size:64"`
}

func (AuditLog) TableName() string { return "blog_admin_audit_logs" }

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func VerifyPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func EnsureDefaultRoles(db *gorm.DB) error {
	defaults := []struct {
		name, description string
		permissions       []string
	}{
		{"超级管理员", "拥有全部平台权限", []string{"*"}},
		{"平台管理员", "管理平台用户、配置与站点", []string{PermissionUsersManage, PermissionRolesManage, PermissionSettingsManage, PermissionSitesManage, PermissionAuditRead, PermissionContentManage}},
		{"内容编辑", "管理文章与标签", []string{PermissionContentManage}},
		{"审计员", "查看平台操作记录", []string{PermissionAuditRead}},
	}
	for _, item := range defaults {
		role := Role{Name: item.name, Description: item.description}
		if err := role.SetPermissions(item.permissions); err != nil {
			return err
		}
		if err := db.Where("name = ?", role.Name).FirstOrCreate(&role).Error; err != nil {
			return err
		}
	}
	return nil
}

func RecordAudit(db *gorm.DB, username, action, resource, method, remoteIP string, status int) error {
	return db.Create(&AuditLog{
		Model:    model.Model{CreatedBy: username, CreatedOn: uint32(time.Now().Unix())},
		Username: username,
		Action:   action,
		Resource: resource,
		Method:   method,
		Status:   status,
		RemoteIP: remoteIP,
	}).Error
}
