package handler

import (
	"fmt"
	"net/http"

	"eventslk-notification-service/email"

	"github.com/gin-gonic/gin"
)

func Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "UP"})
}

// TestEmailHandler returns a gin handler that sends a real test email via Brevo
// and returns the full delivery result as JSON.
//
// Usage: GET /debug/send-test-email?to=you@gmail.com
//
// Response:
//
//	{ "to": "you@gmail.com", "httpStatus": 201, "messageId": "<id@smtp-brevo.com>", "queued": true }
//
// If messageId is empty, Brevo accepted the request but did not queue the email —
// check Brevo dashboard → Transactional → Logs for the reason.
func TestEmailHandler(sender *email.EmailSender) gin.HandlerFunc {
	return func(c *gin.Context) {
		to := c.Query("to")
		if to == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "query param ?to= is required"})
			return
		}

		subject := "EventsLK — Test Email"
		body := fmt.Sprintf(`<!DOCTYPE html><html><body style="font-family:Arial,sans-serif;padding:32px;">
<h2 style="color:#e94560;">EventsLK Test Email</h2>
<p>This is a test email sent from the notification service debug endpoint.</p>
<p><strong>Recipient:</strong> %s</p>
<p>If you received this, Brevo delivery is working correctly.</p>
</body></html>`, to)

		messageId, httpStatus, err := sender.SendRaw(to, subject, body)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"to":         to,
				"httpStatus": httpStatus,
				"error":      err.Error(),
				"queued":     false,
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"to":         to,
			"httpStatus": httpStatus,
			"messageId":  messageId,
			"queued":     messageId != "",
		})
	}
}
