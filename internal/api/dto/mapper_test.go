package dto

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/leirbagxis/FreddyBot/internal/database/models"
)

func TestUserLookupDTOExcludesSensitiveFlags(t *testing.T) {
	data, err := json.Marshal(ToUserLookupDTO(&models.User{UserId: 1, FirstName: "Ana", IsAdmin: true, IsBlacklisted: true}))
	if err != nil {
		t.Fatalf("marshal lookup DTO: %v", err)
	}
	serialized := string(data)
	if strings.Contains(serialized, "is_admin") || strings.Contains(serialized, "is_blacklisted") {
		t.Fatalf("lookup DTO leaked sensitive flags: %s", serialized)
	}
}
