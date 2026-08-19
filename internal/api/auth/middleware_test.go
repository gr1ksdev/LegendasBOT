package auth

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/leirbagxis/FreddyBot/internal/container"
)

func TestAuthorizeChannel_AdminOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		role           Role
		userID         int64
		channelIDParam string
		expectedStatus int
		expectChanID   bool
	}{
		{
			name:           "Admin should get channelID in context",
			role:           RoleAdmin,
			userID:         1,
			channelIDParam: "123",
			expectedStatus: http.StatusOK,
			expectChanID:   true,
		},
		{
			name:           "Owner should get channelID in context",
			role:           RoleOwner,
			userID:         2,
			channelIDParam: "456",
			expectedStatus: http.StatusOK,
			expectChanID:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Request, _ = http.NewRequest(http.MethodGet, "/test/"+tt.channelIDParam, nil)
			c.Params = []gin.Param{{Key: "channelId", Value: tt.channelIDParam}}
			c.Set("role", tt.role)
			c.Set("userID", tt.userID)

			// Mock container - for Admin/Owner it doesn't reach Repo anymore before c.Set
			v := &container.AppContainer{}

			handler := AuthorizeChannel(v)
			handler(c)

			val, exists := c.Get("channelID")
			if tt.expectChanID {
				if !exists {
					t.Errorf("expected channelID in context, but not found")
				} else {
					expectedID, _ := strconv.ParseInt(tt.channelIDParam, 10, 64)
					if val.(int64) != expectedID {
						t.Errorf("expected channelID %d, got %v", expectedID, val)
					}
				}
			}
		})
	}
}
