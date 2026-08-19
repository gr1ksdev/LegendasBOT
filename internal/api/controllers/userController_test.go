package controllers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/internal/core/services"
	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"github.com/leirbagxis/FreddyBot/internal/database/repositories"
	"gorm.io/gorm"
)

func newUserInfoTestController(t *testing.T) (*UserController, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:user-info-controller?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Fatalf("migrate users: %v", err)
	}
	return NewUserController(&container.AppContainer{
		UserService: services.NewUserService(repositories.NewUserRepository(db)),
	}), db
}

func TestGetUserInfoReturnsOnlyLocalPublicProfile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	controller, db := newUserInfoTestController(t)
	if err := db.Create(&models.User{UserId: 42421, FirstName: "Ana", Username: "ana", IsAdmin: true, IsBlacklisted: true}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	router := gin.New()
	router.GET("/user/info/:userParams", controller.GetUserInfo)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/user/info/42421", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	body := response.Body.String()
	if !strings.Contains(body, `"id":42421`) || strings.Contains(body, "is_admin") || strings.Contains(body, "is_blacklisted") {
		t.Fatalf("unexpected public profile response: %s", body)
	}
}
