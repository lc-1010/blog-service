package membership

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"blog-server/global"
	"blog-server/internal/platform"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestMemberRegistrationLoginProfileAndRevocation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:membership-%p?mode=memory&cache=shared", t)), &gorm.Config{})
	if err != nil {
		t.Fatalf("open SQLite database: %v", err)
	}
	if err := db.AutoMigrate(&Member{}, &Profile{}, &Session{}, &platform.Setting{}, &EmailVerificationCode{}); err != nil {
		t.Fatalf("migrate member tables: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get SQL database: %v", err)
	}
	oldDB := global.DBEngine
	global.DBEngine = db
	t.Cleanup(func() {
		global.DBEngine = oldDB
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close SQLite database: %v", err)
		}
	})

	router := gin.New()
	NewHandler().RegisterRoutes(router)
	register := memberRequest(router, http.MethodPost, "/api/v1/public/members/register",
		`{"username":"writer01","display_name":"作者一","password":"correct-long-password"}`, nil)
	if register.Code != http.StatusForbidden {
		t.Fatalf("registration while disabled status = %d, want %d", register.Code, http.StatusForbidden)
	}

	if err := db.Create(&platform.Setting{Key: "allow_registration", Value: "true"}).Error; err != nil {
		t.Fatalf("enable registration: %v", err)
	}
	register = memberRequest(router, http.MethodPost, "/api/v1/public/members/register",
		`{"username":"writer01","display_name":"作者一","password":"correct-long-password"}`, nil)
	if register.Code != http.StatusCreated {
		t.Fatalf("registration status = %d, body = %s", register.Code, register.Body.String())
	}
	cookie := memberCookie(t, register)
	var member Member
	if err := db.Where("username = ?", "writer01").First(&member).Error; err != nil {
		t.Fatalf("load registered member: %v", err)
	}
	if !platform.VerifyPassword(member.PasswordHash, "correct-long-password") {
		t.Fatal("stored member password is not a valid bcrypt hash")
	}
	var session Session
	if err := db.Where("member_id = ?", member.ID).First(&session).Error; err != nil {
		t.Fatalf("load registered member session: %v", err)
	}
	if session.TokenHash == cookie.Value {
		t.Fatal("raw session token must not be stored in the database")
	}

	me := memberRequest(router, http.MethodGet, "/api/v1/public/members/me", "", cookie)
	if me.Code != http.StatusOK || !bytes.Contains(me.Body.Bytes(), []byte("writer01")) {
		t.Fatalf("member me status/body = %d/%s", me.Code, me.Body.String())
	}
	update := memberRequest(router, http.MethodPatch, "/api/v1/public/members/me/profile",
		`{"display_name":"诗人","bio":"分享文字","avatar_url":"https://example.com/avatar.png"}`, cookie)
	if update.Code != http.StatusOK {
		t.Fatalf("profile update status/body = %d/%s", update.Code, update.Body.String())
	}
	publicProfile := memberRequest(router, http.MethodGet, "/api/v1/public/members/writer01", "", nil)
	if publicProfile.Code != http.StatusOK || !bytes.Contains(publicProfile.Body.Bytes(), []byte("分享文字")) {
		t.Fatalf("public profile status/body = %d/%s", publicProfile.Code, publicProfile.Body.String())
	}

	logout := memberRequest(router, http.MethodPost, "/api/v1/public/members/logout", "", cookie)
	if logout.Code != http.StatusOK {
		t.Fatalf("logout status/body = %d/%s", logout.Code, logout.Body.String())
	}
	var revoked Session
	if err := db.First(&revoked, session.ID).Error; err != nil || revoked.RevokedAt == nil {
		t.Fatalf("logout did not revoke the session: session=%+v err=%v", revoked, err)
	}
	me = memberRequest(router, http.MethodGet, "/api/v1/public/members/me", "", cookie)
	if me.Code != http.StatusUnauthorized {
		t.Fatalf("request with revoked session status = %d, want %d", me.Code, http.StatusUnauthorized)
	}

	login := memberRequest(router, http.MethodPost, "/api/v1/public/members/login",
		`{"username":"writer01","password":"correct-long-password"}`, nil)
	if login.Code != http.StatusOK {
		t.Fatalf("login status/body = %d/%s", login.Code, login.Body.String())
	}
	loginCookie := memberCookie(t, login)
	changePassword := memberRequest(router, http.MethodPatch, "/api/v1/public/members/me/password",
		`{"current_password":"correct-long-password","new_password":"another-correct-password"}`, loginCookie)
	if changePassword.Code != http.StatusOK {
		t.Fatalf("password change status/body = %d/%s", changePassword.Code, changePassword.Body.String())
	}
	me = memberRequest(router, http.MethodGet, "/api/v1/public/members/me", "", loginCookie)
	if me.Code != http.StatusUnauthorized {
		t.Fatalf("password change did not revoke existing session: status = %d", me.Code)
	}
	oldPasswordLogin := memberRequest(router, http.MethodPost, "/api/v1/public/members/login",
		`{"username":"writer01","password":"correct-long-password"}`, nil)
	if oldPasswordLogin.Code != http.StatusUnauthorized {
		t.Fatalf("old password login status = %d, want %d", oldPasswordLogin.Code, http.StatusUnauthorized)
	}
	newPasswordLogin := memberRequest(router, http.MethodPost, "/api/v1/public/members/login",
		`{"username":"writer01","password":"another-correct-password"}`, nil)
	if newPasswordLogin.Code != http.StatusOK {
		t.Fatalf("new password login status/body = %d/%s", newPasswordLogin.Code, newPasswordLogin.Body.String())
	}
	duplicate := memberRequest(router, http.MethodPost, "/api/v1/public/members/register",
		`{"username":"writer01","display_name":"作者一","password":"correct-long-password"}`, nil)
	if duplicate.Code != http.StatusConflict {
		t.Fatalf("duplicate registration status = %d, want %d", duplicate.Code, http.StatusConflict)
	}
}

func TestMemberRegistrationValidationAndUnknownAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:membership-validation-%p?mode=memory&cache=shared", t)), &gorm.Config{})
	if err != nil {
		t.Fatalf("open SQLite database: %v", err)
	}
	if err := db.AutoMigrate(&Member{}, &Profile{}, &Session{}, &platform.Setting{}, &EmailVerificationCode{}); err != nil {
		t.Fatalf("migrate member tables: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get SQL database: %v", err)
	}
	oldDB := global.DBEngine
	global.DBEngine = db
	t.Cleanup(func() {
		global.DBEngine = oldDB
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close SQLite database: %v", err)
		}
	})
	if err := db.Create(&platform.Setting{Key: "allow_registration", Value: "true"}).Error; err != nil {
		t.Fatalf("enable registration: %v", err)
	}

	router := gin.New()
	NewHandler().RegisterRoutes(router)
	invalid := memberRequest(router, http.MethodPost, "/api/v1/public/members/register",
		`{"username":"bad name","display_name":"A","password":"short"}`, nil)
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid registration status = %d, want %d", invalid.Code, http.StatusBadRequest)
	}
	login := memberRequest(router, http.MethodPost, "/api/v1/public/members/login",
		`{"username":"missing","password":"correct-long-password"}`, nil)
	if login.Code != http.StatusUnauthorized {
		t.Fatalf("unknown member login status = %d, want %d", login.Code, http.StatusUnauthorized)
	}
}

func memberRequest(router http.Handler, method, path, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		request.AddCookie(cookie)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func memberCookie(t *testing.T, response *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == sessionCookieName {
			if !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode || cookie.MaxAge <= 0 {
				t.Fatalf("unexpected member session cookie attributes: %+v", cookie)
			}
			return cookie
		}
	}
	t.Fatal("response did not set member session cookie")
	return nil
}

func TestMemberResponseDoesNotExposePasswordHash(t *testing.T) {
	result, err := json.Marshal(Member{Username: "reader", PasswordHash: "sensitive"})
	if err != nil {
		t.Fatalf("marshal member: %v", err)
	}
	if bytes.Contains(result, []byte("sensitive")) || bytes.Contains(result, []byte("password_hash")) {
		t.Fatalf("member response exposed password data: %s", result)
	}
}

func TestEmailVerificationFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:email-verification-%p?mode=memory&cache=shared", t)), &gorm.Config{})
	if err != nil {
		t.Fatalf("open SQLite database: %v", err)
	}
	if err := db.AutoMigrate(&Member{}, &Profile{}, &Session{}, &platform.Setting{}, &EmailVerificationCode{}); err != nil {
		t.Fatalf("migrate member tables: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get SQL database: %v", err)
	}
	oldDB := global.DBEngine
	global.DBEngine = db
	t.Cleanup(func() {
		global.DBEngine = oldDB
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close SQLite database: %v", err)
		}
	})
	if err := db.Create(&platform.Setting{Key: "allow_registration", Value: "true"}).Error; err != nil {
		t.Fatalf("enable registration: %v", err)
	}

	router := gin.New()
	NewHandler().RegisterRoutes(router)

	// Register a member with email
	register := memberRequest(router, http.MethodPost, "/api/v1/public/members/register",
		`{"username":"emailuser","display_name":"Email User","password":"correct-long-password","email":"email@test.com"}`, nil)
	if register.Code != http.StatusCreated {
		t.Fatalf("registration status = %d, body = %s", register.Code, register.Body.String())
	}
	cookie := memberCookie(t, register)

	// Request verification code
	sendVerification := memberRequest(router, http.MethodPost, "/api/v1/public/members/send-verification", "", cookie)
	if sendVerification.Code != http.StatusOK {
		t.Fatalf("send verification status = %d, body = %s", sendVerification.Code, sendVerification.Body.String())
	}

	// Get the verification code from the database
	var member Member
	if err := db.Where("username = ?", "emailuser").First(&member).Error; err != nil {
		t.Fatalf("load member: %v", err)
	}
	var verification EmailVerificationCode
	if err := db.Where("member_id = ?", member.ID).Order("created_at DESC").First(&verification).Error; err != nil {
		t.Fatalf("load verification code: %v", err)
	}

	// Verify with the correct code
	verify := memberRequest(router, http.MethodPost, "/api/v1/public/members/verify-email",
		fmt.Sprintf(`{"code":"%s"}`, verification.Code), cookie)
	if verify.Code != http.StatusOK {
		t.Fatalf("verify email status = %d, body = %s", verify.Code, verify.Body.String())
	}

	// Check that the member's email is now verified
	var updatedMember Member
	if err := db.Where("id = ?", member.ID).First(&updatedMember).Error; err != nil {
		t.Fatalf("load updated member: %v", err)
	}
	if updatedMember.EmailVerified != 1 {
		t.Fatal("member email should be verified")
	}

	// Try to verify again with the same code (should fail - code already used)
	verifyAgain := memberRequest(router, http.MethodPost, "/api/v1/public/members/verify-email",
		fmt.Sprintf(`{"code":"%s"}`, verification.Code), cookie)
	if verifyAgain.Code != http.StatusUnauthorized {
		t.Fatalf("re-verify with used code status = %d, want %d", verifyAgain.Code, http.StatusUnauthorized)
	}

	// Try to verify with an invalid code
	verifyInvalid := memberRequest(router, http.MethodPost, "/api/v1/public/members/verify-email",
		`{"code":"000000"}`, cookie)
	if verifyInvalid.Code != http.StatusUnauthorized {
		t.Fatalf("verify with invalid code status = %d, want %d", verifyInvalid.Code, http.StatusUnauthorized)
	}
}
