package admin

import (
	"blog-server/global"
	"blog-server/internal/platform"
	"errors"
	"net/http"
	"net/mail"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var configurableSettings = map[string]struct{}{
	"site_name":          {},
	"site_url":           {},
	"site_description":   {},
	"timezone":           {},
	"allow_registration": {},
	"maintenance_mode":   {},
	"support_email":      {},
}

var allowedPermissions = map[string]struct{}{
	platform.PermissionContentManage:  {},
	platform.PermissionUsersManage:    {},
	platform.PermissionRolesManage:    {},
	platform.PermissionSettingsManage: {},
	platform.PermissionSitesManage:    {},
	platform.PermissionAuditRead:      {},
}

func (h Handler) registerPlatformRoutes(router *gin.Engine) {
	group := router.Group("/api/v1/admin/platform")
	group.Use(h.AuthRequired())
	group.GET("/dashboard", h.RequirePermission(platform.PermissionAuditRead), h.PlatformDashboard)
	group.GET("/users", h.RequirePermission(platform.PermissionUsersManage), h.ListUsers)
	group.POST("/users", h.RequirePermission(platform.PermissionUsersManage), h.CreateUser)
	group.PUT("/users/:id", h.RequirePermission(platform.PermissionUsersManage), h.UpdateUser)
	group.GET("/roles", h.RequireAnyPermission(platform.PermissionUsersManage, platform.PermissionRolesManage), h.ListRoles)
	group.POST("/roles", h.RequirePermission(platform.PermissionRolesManage), h.CreateRole)
	group.PUT("/roles/:id", h.RequirePermission(platform.PermissionRolesManage), h.UpdateRole)
	group.DELETE("/roles/:id", h.RequirePermission(platform.PermissionRolesManage), h.DeleteRole)
	group.GET("/settings", h.RequirePermission(platform.PermissionSettingsManage), h.GetSettings)
	group.PUT("/settings", h.RequirePermission(platform.PermissionSettingsManage), h.UpdateSettings)
	group.GET("/sites", h.RequirePermission(platform.PermissionSitesManage), h.ListSites)
	group.POST("/sites", h.RequirePermission(platform.PermissionSitesManage), h.CreateSite)
	group.PUT("/sites/:id", h.RequirePermission(platform.PermissionSitesManage), h.UpdateSite)
	group.DELETE("/sites/:id", h.RequirePermission(platform.PermissionSitesManage), h.DeleteSite)
	group.GET("/audit-logs", h.RequirePermission(platform.PermissionAuditRead), h.ListAuditLogs)
}

func (h Handler) RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		permissions, ok := c.Get("admin_permissions")
		list, isList := permissions.([]string)
		if !ok || !isList || hasPermission(list, "*") || hasPermission(list, permission) {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "permission denied"})
	}
}

func (h Handler) RequireAnyPermission(permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		value, ok := c.Get("admin_permissions")
		granted, isList := value.([]string)
		if ok && isList {
			for _, permission := range permissions {
				if hasPermission(granted, permission) {
					c.Next()
					return
				}
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "permission denied"})
	}
}

func (h Handler) PlatformDashboard(c *gin.Context) {
	db, ok := platformDB(c)
	if !ok {
		return
	}
	var result struct {
		Users       int64 `json:"users"`
		ActiveUsers int64 `json:"active_users"`
		Roles       int64 `json:"roles"`
		Sites       int64 `json:"sites"`
		Operations  int64 `json:"operations"`
		Failures    int64 `json:"failures"`
	}
	for _, query := range []struct {
		model any
		where string
		dest  *int64
	}{
		{&platform.User{}, "", &result.Users},
		{&platform.User{}, "state = ?", &result.ActiveUsers},
		{&platform.Role{}, "", &result.Roles},
		{&platform.Site{}, "state = ?", &result.Sites},
		{&platform.AuditLog{}, "created_on >= ?", &result.Operations},
		{&platform.AuditLog{}, "created_on >= ? AND status >= ?", &result.Failures},
	} {
		count := db.Model(query.model)
		switch query.where {
		case "state = ?":
			count = count.Where(query.where, 1)
		case "created_on >= ?":
			count = count.Where(query.where, time.Now().AddDate(0, 0, -7).Unix())
		case "created_on >= ? AND status >= ?":
			count = count.Where(query.where, time.Now().AddDate(0, 0, -7).Unix(), 400)
		}
		if err := count.Count(query.dest).Error; err != nil {
			writePlatformError(c, err)
			return
		}
	}
	c.JSON(http.StatusOK, result)
}

func (h Handler) ListUsers(c *gin.Context) {
	db, ok := platformDB(c)
	if !ok {
		return
	}
	var users []platform.User
	if err := db.Order("id DESC").Limit(500).Find(&users).Error; err != nil {
		writePlatformError(c, err)
		return
	}
	roles, err := roleMap(db)
	if err != nil {
		writePlatformError(c, err)
		return
	}
	type userResponse struct {
		ID          uint32 `json:"id"`
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
		RoleID      uint32 `json:"role_id"`
		RoleName    string `json:"role_name"`
		State       uint8  `json:"state"`
		CreatedOn   uint32 `json:"created_on"`
	}
	result := make([]userResponse, 0, len(users)+1)
	for _, user := range users {
		result = append(result, userResponse{
			ID: user.ID, Username: user.Username, DisplayName: user.DisplayName,
			RoleID: user.RoleID, RoleName: roles[user.RoleID].Name, State: user.State, CreatedOn: user.CreatedOn,
		})
	}
	c.JSON(http.StatusOK, gin.H{"list": result})
}

func (h Handler) CreateUser(c *gin.Context) {
	var input struct {
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
		Password    string `json:"password"`
		RoleID      uint32 `json:"role_id"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || !validUsername(input.Username) || len(input.DisplayName) > 100 || len(input.Password) < 12 || len(input.Password) > 72 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "username, display_name, password (12-72 characters), and role_id are required"})
		return
	}
	db, ok := platformDB(c)
	if !ok {
		return
	}
	role, err := getRole(db, input.RoleID)
	if err != nil {
		writePlatformError(c, err)
		return
	}
	if hasPermission(role.PermissionList(), "*") {
		c.JSON(http.StatusBadRequest, gin.H{"message": "managed users cannot be assigned unrestricted permissions"})
		return
	}
	hash, err := platform.HashPassword(input.Password)
	if err != nil {
		writePlatformError(c, err)
		return
	}
	user := platform.User{
		Username: input.Username, DisplayName: strings.TrimSpace(input.DisplayName),
		PasswordHash: hash, RoleID: input.RoleID, State: 1,
	}
	if err := db.Create(&user).Error; err != nil {
		if duplicateKey(err) {
			c.JSON(http.StatusConflict, gin.H{"message": "username already exists"})
			return
		}
		writePlatformError(c, err)
		return
	}
	h.audit(c, "user.create", user.Username, http.StatusCreated)
	c.JSON(http.StatusCreated, gin.H{"id": user.ID, "username": user.Username})
}

func (h Handler) UpdateUser(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var input struct {
		DisplayName string `json:"display_name"`
		RoleID      uint32 `json:"role_id"`
		State       uint8  `json:"state"`
		Password    string `json:"password"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || len(input.DisplayName) > 100 || input.State > 1 || (input.Password != "" && (len(input.Password) < 12 || len(input.Password) > 72)) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid user update"})
		return
	}
	if currentUserID(c) == id {
		c.JSON(http.StatusBadRequest, gin.H{"message": "you cannot change your own role or status"})
		return
	}
	db, ok := platformDB(c)
	if !ok {
		return
	}
	role, err := getRole(db, input.RoleID)
	if err != nil {
		writePlatformError(c, err)
		return
	}
	if hasPermission(role.PermissionList(), "*") {
		c.JSON(http.StatusBadRequest, gin.H{"message": "managed users cannot be assigned unrestricted permissions"})
		return
	}
	updates := map[string]any{"display_name": strings.TrimSpace(input.DisplayName), "role_id": input.RoleID, "state": input.State}
	if input.Password != "" {
		hash, err := platform.HashPassword(input.Password)
		if err != nil {
			writePlatformError(c, err)
			return
		}
		updates["password_hash"] = hash
	}
	var existing platform.User
	if err := db.Where("id = ? AND is_del = ?", id, 0).First(&existing).Error; err != nil {
		writePlatformError(c, err)
		return
	}
	result := db.Model(&platform.User{}).Where("id = ? AND is_del = ?", id, 0).Updates(updates)
	if result.Error != nil {
		writePlatformError(c, result.Error)
		return
	}
	h.audit(c, "user.update", "user:"+c.Param("id"), http.StatusOK)
	c.JSON(http.StatusOK, gin.H{"message": "user updated"})
}

func (h Handler) ListRoles(c *gin.Context) {
	db, ok := platformDB(c)
	if !ok {
		return
	}
	var roles []platform.Role
	if err := db.Where("is_del = ?", 0).Order("id ASC").Limit(200).Find(&roles).Error; err != nil {
		writePlatformError(c, err)
		return
	}
	type roleResponse struct {
		ID          uint32   `json:"id"`
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Permissions []string `json:"permissions"`
	}
	result := make([]roleResponse, 0, len(roles))
	for _, role := range roles {
		result = append(result, roleResponse{role.ID, role.Name, role.Description, role.PermissionList()})
	}
	c.JSON(http.StatusOK, gin.H{"list": result, "available_permissions": permissionOptions()})
}

func (h Handler) CreateRole(c *gin.Context) {
	var input struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Permissions []string `json:"permissions"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || !validRoleInput(input.Name, input.Description, input.Permissions) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid role name, description, or permissions"})
		return
	}
	db, ok := platformDB(c)
	if !ok {
		return
	}
	role := platform.Role{Name: strings.TrimSpace(input.Name), Description: strings.TrimSpace(input.Description)}
	if err := role.SetPermissions(input.Permissions); err != nil {
		writePlatformError(c, err)
		return
	}
	if err := db.Create(&role).Error; err != nil {
		if duplicateKey(err) {
			c.JSON(http.StatusConflict, gin.H{"message": "role name already exists"})
			return
		}
		writePlatformError(c, err)
		return
	}
	h.audit(c, "role.create", role.Name, http.StatusCreated)
	c.JSON(http.StatusCreated, gin.H{"id": role.ID})
}

func (h Handler) UpdateRole(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var input struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Permissions []string `json:"permissions"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || !validRoleInput(input.Name, input.Description, input.Permissions) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid role name, description, or permissions"})
		return
	}
	db, ok := platformDB(c)
	if !ok {
		return
	}
	role, err := getRole(db, id)
	if err != nil {
		writePlatformError(c, err)
		return
	}
	if hasPermission(role.PermissionList(), "*") || role.Name == "超级管理员" {
		c.JSON(http.StatusForbidden, gin.H{"message": "built-in unrestricted role cannot be modified"})
		return
	}
	role.Name = strings.TrimSpace(input.Name)
	role.Description = strings.TrimSpace(input.Description)
	if err := role.SetPermissions(input.Permissions); err != nil {
		writePlatformError(c, err)
		return
	}
	if err := db.Save(&role).Error; err != nil {
		if duplicateKey(err) {
			c.JSON(http.StatusConflict, gin.H{"message": "role name already exists"})
			return
		}
		writePlatformError(c, err)
		return
	}
	h.audit(c, "role.update", role.Name, http.StatusOK)
	c.JSON(http.StatusOK, gin.H{"message": "role updated"})
}

func (h Handler) DeleteRole(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	db, ok := platformDB(c)
	if !ok {
		return
	}
	role, err := getRole(db, id)
	if err != nil {
		writePlatformError(c, err)
		return
	}
	if hasPermission(role.PermissionList(), "*") {
		c.JSON(http.StatusForbidden, gin.H{"message": "built-in unrestricted role cannot be deleted"})
		return
	}
	var users int64
	if err := db.Model(&platform.User{}).Where("role_id = ?", id).Count(&users).Error; err != nil {
		writePlatformError(c, err)
		return
	}
	if users > 0 {
		c.JSON(http.StatusConflict, gin.H{"message": "role is assigned to users"})
		return
	}
	if err := db.Delete(&role).Error; err != nil {
		writePlatformError(c, err)
		return
	}
	h.audit(c, "role.delete", role.Name, http.StatusOK)
	c.JSON(http.StatusOK, gin.H{"message": "role deleted"})
}

func (h Handler) GetSettings(c *gin.Context) {
	db, ok := platformDB(c)
	if !ok {
		return
	}
	var settings []platform.Setting
	if err := db.Order("setting_key ASC").Find(&settings).Error; err != nil {
		writePlatformError(c, err)
		return
	}
	values := make(map[string]string, len(configurableSettings))
	for _, setting := range settings {
		values[setting.Key] = setting.Value
	}
	for key := range configurableSettings {
		if _, exists := values[key]; !exists {
			values[key] = ""
		}
	}
	c.JSON(http.StatusOK, gin.H{"settings": values})
}

func (h Handler) UpdateSettings(c *gin.Context) {
	var input map[string]string
	if err := c.ShouldBindJSON(&input); err != nil || len(input) == 0 || len(input) > len(configurableSettings) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid settings payload"})
		return
	}
	for key, value := range input {
		if _, ok := configurableSettings[key]; !ok || len(value) > 500 {
			c.JSON(http.StatusBadRequest, gin.H{"message": "unsupported setting or value too long"})
			return
		}
		if key == "site_url" && value != "" {
			parsed, err := url.ParseRequestURI(value)
			if err != nil || parsed.Host == "" || parsed.Scheme != "http" && parsed.Scheme != "https" {
				c.JSON(http.StatusBadRequest, gin.H{"message": "invalid site URL or support email"})
				return
			}
		}
		if key == "support_email" && value != "" {
			if _, err := mail.ParseAddress(value); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"message": "invalid support email"})
				return
			}
		}
		if (key == "allow_registration" || key == "maintenance_mode") && value != "true" && value != "false" {
			c.JSON(http.StatusBadRequest, gin.H{"message": "boolean settings must be true or false"})
			return
		}
	}
	db, ok := platformDB(c)
	if !ok {
		return
	}
	err := db.Transaction(func(tx *gorm.DB) error {
		for key, value := range input {
			setting := platform.Setting{Key: key}
			if err := tx.Where(platform.Setting{Key: key}).Assign(platform.Setting{Value: value}).FirstOrCreate(&setting).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		writePlatformError(c, err)
		return
	}
	h.audit(c, "settings.update", "platform settings", http.StatusOK)
	c.JSON(http.StatusOK, gin.H{"message": "settings updated"})
}

func (h Handler) ListSites(c *gin.Context) {
	db, ok := platformDB(c)
	if !ok {
		return
	}
	var sites []platform.Site
	if err := db.Order("id ASC").Limit(200).Find(&sites).Error; err != nil {
		writePlatformError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"list": sites})
}

func (h Handler) CreateSite(c *gin.Context) {
	var input struct {
		Name        string `json:"name"`
		Domain      string `json:"domain"`
		Description string `json:"description"`
		State       uint8  `json:"state"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || !validSite(input.Name, input.Domain, input.Description, input.State) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid site name, domain, description, or state"})
		return
	}
	db, ok := platformDB(c)
	if !ok {
		return
	}
	site := platform.Site{
		Name: strings.TrimSpace(input.Name), Domain: normalizeDomain(input.Domain),
		Description: strings.TrimSpace(input.Description), State: input.State,
	}
	if err := db.Create(&site).Error; err != nil {
		if duplicateKey(err) {
			c.JSON(http.StatusConflict, gin.H{"message": "site domain already exists"})
			return
		}
		writePlatformError(c, err)
		return
	}
	h.audit(c, "site.create", site.Domain, http.StatusCreated)
	c.JSON(http.StatusCreated, gin.H{"id": site.ID})
}

func (h Handler) UpdateSite(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var input struct {
		Name        string `json:"name"`
		Domain      string `json:"domain"`
		Description string `json:"description"`
		State       uint8  `json:"state"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || !validSite(input.Name, input.Domain, input.Description, input.State) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid site name, domain, description, or state"})
		return
	}
	db, ok := platformDB(c)
	if !ok {
		return
	}
	var existing platform.Site
	if err := db.Where("id = ? AND is_del = ?", id, 0).First(&existing).Error; err != nil {
		writePlatformError(c, err)
		return
	}
	result := db.Model(&platform.Site{}).Where("id = ? AND is_del = ?", id, 0).Updates(map[string]any{
		"name": strings.TrimSpace(input.Name), "domain": normalizeDomain(input.Domain),
		"description": strings.TrimSpace(input.Description), "state": input.State,
	})
	if result.Error != nil {
		if duplicateKey(result.Error) {
			c.JSON(http.StatusConflict, gin.H{"message": "site domain already exists"})
			return
		}
		writePlatformError(c, result.Error)
		return
	}
	h.audit(c, "site.update", "site:"+c.Param("id"), http.StatusOK)
	c.JSON(http.StatusOK, gin.H{"message": "site updated"})
}

func (h Handler) DeleteSite(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	db, ok := platformDB(c)
	if !ok {
		return
	}
	result := db.Delete(&platform.Site{}, id)
	if result.Error != nil {
		writePlatformError(c, result.Error)
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "site not found"})
		return
	}
	h.audit(c, "site.delete", "site:"+c.Param("id"), http.StatusOK)
	c.JSON(http.StatusOK, gin.H{"message": "site deleted"})
}

func (h Handler) ListAuditLogs(c *gin.Context) {
	db, ok := platformDB(c)
	if !ok {
		return
	}
	page, pageSize := boundedPage(c.Query("page"), c.Query("page_size"))
	query := db.Model(&platform.AuditLog{})
	if username := strings.TrimSpace(c.Query("username")); username != "" {
		query = query.Where("username = ?", username)
	}
	if action := strings.TrimSpace(c.Query("action")); action != "" {
		query = query.Where("action LIKE ?", "%"+action+"%")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		writePlatformError(c, err)
		return
	}
	var logs []platform.AuditLog
	if err := query.Order("created_on DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&logs).Error; err != nil {
		writePlatformError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"list":  logs,
		"pager": gin.H{"page": page, "page_size": pageSize, "total_rows": total},
	})
}

func (h Handler) audit(c *gin.Context, action, resource string, status int) {
	db := global.DBEngine
	if db == nil {
		return
	}
	if err := platform.RecordAudit(db, currentUsername(c), action, resource, c.Request.Method, c.ClientIP(), status); err != nil && global.Logger != nil {
		global.Logger.Errorf(c, "record platform audit log err: %v", err)
	}
}

func platformDB(c *gin.Context) (*gorm.DB, bool) {
	if global.DBEngine == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "database is not available"})
		return nil, false
	}
	return global.DBEngine.WithContext(c.Request.Context()), true
}

func writePlatformError(c *gin.Context, err error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"message": "resource not found"})
		return
	}
	if global.Logger != nil {
		global.Logger.Errorf(c, "platform management request err: %v", err)
	}
	c.JSON(http.StatusInternalServerError, gin.H{"message": "platform management request failed"})
}

func roleMap(db *gorm.DB) (map[uint32]platform.Role, error) {
	var roles []platform.Role
	if err := db.Find(&roles).Error; err != nil {
		return nil, err
	}
	result := make(map[uint32]platform.Role, len(roles))
	for _, role := range roles {
		result[role.ID] = role
	}
	return result, nil
}

func getRole(db *gorm.DB, id uint32) (platform.Role, error) {
	var role platform.Role
	err := db.Where("id = ? AND is_del = ?", id, 0).First(&role).Error
	return role, err
}

func pathID(c *gin.Context) (uint32, bool) {
	var input struct {
		ID uint32 `uri:"id" binding:"required,gte=1"`
	}
	if err := c.ShouldBindUri(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid resource id"})
		return 0, false
	}
	return input.ID, true
}

func currentUsername(c *gin.Context) string {
	username, _ := c.Get("admin_username")
	name, _ := username.(string)
	return name
}

func currentUserID(c *gin.Context) uint32 {
	id, _ := c.Get("admin_user_id")
	userID, _ := id.(uint32)
	return userID
}

func hasPermission(permissions []string, required string) bool {
	for _, permission := range permissions {
		if permission == required || permission == "*" {
			return true
		}
	}
	return false
}

func validUsername(username string) bool {
	username = strings.TrimSpace(username)
	if len(username) < 3 || len(username) > 100 {
		return false
	}
	for _, char := range username {
		if !(char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || char == '.' || char == '_' || char == '-') {
			return false
		}
	}
	return true
}

func validRoleInput(name, description string, permissions []string) bool {
	if strings.TrimSpace(name) == "" || len(name) > 80 || len(description) > 255 || len(permissions) == 0 {
		return false
	}
	for _, permission := range permissions {
		if _, ok := allowedPermissions[permission]; !ok {
			return false
		}
	}
	return true
}

func permissionOptions() []string {
	result := make([]string, 0, len(allowedPermissions))
	for permission := range allowedPermissions {
		result = append(result, permission)
	}
	sort.Strings(result)
	return result
}

func validSite(name, domain, description string, state uint8) bool {
	if strings.TrimSpace(name) == "" || len(name) > 120 || len(strings.TrimSpace(domain)) > 255 || len(description) > 255 || state > 1 {
		return false
	}
	normalized := normalizeDomain(domain)
	parsed, err := url.Parse("https://" + normalized)
	return err == nil && parsed.Host != "" && parsed.Path == "" && parsed.RawQuery == "" &&
		parsed.Fragment == "" && parsed.User == nil && !strings.ContainsAny(parsed.Host, " /\\")
}

func normalizeDomain(domain string) string {
	domain = strings.TrimSpace(strings.ToLower(domain))
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimPrefix(domain, "http://")
	return strings.TrimSuffix(domain, "/")
}

func boundedPage(pageValue, pageSizeValue string) (int, int) {
	page, pageSize := 1, 20
	if value, err := strconv.Atoi(pageValue); err == nil && value > 0 {
		page = value
	}
	if value, err := strconv.Atoi(pageSizeValue); err == nil && value > 0 {
		pageSize = value
	}
	if pageSize > 100 {
		pageSize = 100
	}
	if page > 1_000_000 {
		page = 1_000_000
	}
	return page, pageSize
}

func duplicateKey(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "duplicate") || strings.Contains(strings.ToLower(err.Error()), "unique constraint")
}
