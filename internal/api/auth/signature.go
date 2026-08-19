package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/leirbagxis/FreddyBot/pkg/config"
)

func GenerateSignature(ownerID, channelID, secreteKey string) string {
	sum := ownerID + channelID
	data := fmt.Sprintf("signaturePayload:%s", sum)
	mac := hmac.New(sha256.New, []byte(secreteKey))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}

func ValidateSignature(ownerID, channelID, receivedSig, secreteKey string) bool {
	expectedSig := GenerateSignature(ownerID, channelID, secreteKey)
	return hmac.Equal([]byte(expectedSig), []byte(receivedSig))
}

func GenerateMiniAppUrl(userID, channelID string) string {
	//hash := GenerateSignature(userID, channelID, config.SecreteKey)
	return fmt.Sprintf("%s/dashboard/%s", config.WebAppURL, channelID)
}
