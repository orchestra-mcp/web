package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
)

// SendPush sends a Web Push notification to all registered subscriptions for a user.
// Runs asynchronously — returns immediately.
func SendPush(db *gorm.DB, userID uint, title, message, ntype string) {
	go sendPushSync(db, userID, title, message, ntype)
}

func sendPushSync(db *gorm.DB, userID uint, title, message, ntype string) {
	privateKey := os.Getenv("VAPID_PRIVATE_KEY")
	publicKey := os.Getenv("VAPID_PUBLIC_KEY")
	subject := os.Getenv("VAPID_SUBJECT")
	if privateKey == "" || publicKey == "" {
		return // Web Push not configured
	}
	if subject == "" {
		subject = "mailto:noreply@orchestra-mcp.com"
	}

	var subs []models.PushSubscription
	db.Where("user_id = ?", userID).Find(&subs)
	if len(subs) == 0 {
		return
	}

	payload, _ := json.Marshal(map[string]string{
		"title":   title,
		"message": message,
		"body":    message,
		"type":    ntype,
		"url":     "/",
	})

	for _, sub := range subs {
		s := &webpush.Subscription{
			Endpoint: sub.Endpoint,
			Keys: webpush.Keys{
				P256dh: sub.P256dh,
				Auth:   sub.Auth,
			},
		}

		resp, err := webpush.SendNotification(payload, s, &webpush.Options{
			Subscriber:      subject,
			VAPIDPublicKey:  publicKey,
			VAPIDPrivateKey: privateKey,
			TTL:             60,
		})
		if err != nil {
			fmt.Printf("[WebPush] Error sending to %s: %v\n", sub.Endpoint, err)
			continue
		}
		resp.Body.Close()

		// Remove expired/invalid subscriptions
		if resp.StatusCode == http.StatusGone || resp.StatusCode == http.StatusNotFound {
			db.Delete(&models.PushSubscription{}, sub.ID)
		}
	}
}
