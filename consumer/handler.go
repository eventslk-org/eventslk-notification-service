package consumer

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"eventslk-notification-service/email"

	"github.com/IBM/sarama"
)

type NotificationHandler struct {
	emailSender *email.EmailSender
}

func NewNotificationHandler(sender *email.EmailSender) *NotificationHandler {
	return &NotificationHandler{emailSender: sender}
}

func (h *NotificationHandler) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (h *NotificationHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (h *NotificationHandler) ConsumeClaim(
	session sarama.ConsumerGroupSession,
	claim sarama.ConsumerGroupClaim,
) error {
	for msg := range claim.Messages() {
		if err := h.dispatch(msg); err != nil {
			log.Printf("[consumer] dispatch error on topic %s offset %d: %v", msg.Topic, msg.Offset, err)
			if !errors.Is(err, email.ErrTemplateFailed) {
				// Recoverable error (e.g. Brevo API unreachable) — skip MarkMessage so
				// Kafka re-delivers this message after the next consumer group restart.
				continue
			}
			// Unrecoverable error (bad JSON or broken template) — commit to skip the
			// broken message and avoid an infinite retry loop.
		}
		session.MarkMessage(msg, "")
	}
	return nil
}

func (h *NotificationHandler) dispatch(msg *sarama.ConsumerMessage) error {
	switch msg.Topic {
	case "eventslk.user.signup":
		var event UserSignupEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			return fmt.Errorf("%w: unmarshal UserSignupEvent: %v", email.ErrTemplateFailed, err)
		}
		return h.emailSender.SendEmail(email.EmailMessage{
			To:           event.Email,
			Subject:      "Welcome to EventsLK — Verify Your Email",
			TemplateName: "user_signup.html",
			TemplateData: event,
		})

	case "eventslk.user.otp":
		var event UserOtpEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			return fmt.Errorf("%w: unmarshal UserOtpEvent: %v", email.ErrTemplateFailed, err)
		}
		return h.emailSender.SendEmail(email.EmailMessage{
			To:           event.Email,
			Subject:      "Your EventsLK One-Time Code",
			TemplateName: "user_otp.html",
			TemplateData: event,
		})

	case "eventslk.booking.confirmed":
		var event BookingConfirmedEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			return fmt.Errorf("%w: unmarshal BookingConfirmedEvent: %v", email.ErrTemplateFailed, err)
		}
		return h.emailSender.SendEmail(email.EmailMessage{
			To:           event.Email,
			Subject:      "Booking Confirmed — " + event.EventName,
			TemplateName: "booking_confirmed.html",
			TemplateData: event,
		})

	case "eventslk.booking.cancelled":
		var event BookingCancelledEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			return fmt.Errorf("%w: unmarshal BookingCancelledEvent: %v", email.ErrTemplateFailed, err)
		}
		return h.emailSender.SendEmail(email.EmailMessage{
			To:           event.Email,
			Subject:      "Booking Cancelled — " + event.EventName,
			TemplateName: "booking_cancelled.html",
			TemplateData: event,
		})

	case "eventslk.promotion":
		var event PromotionEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			return fmt.Errorf("%w: unmarshal PromotionEvent: %v", email.ErrTemplateFailed, err)
		}
		// Fan-out: send to every recipient, log individual failures, always commit.
		// Promotion delivery is best-effort — partial failure does not block the offset.
		for _, recipient := range event.RecipientEmails {
			if err := h.emailSender.SendEmail(email.EmailMessage{
				To:           recipient,
				Subject:      event.Subject,
				TemplateName: "promotion.html",
				TemplateData: event,
			}); err != nil {
				log.Printf("[consumer] promotion email failed for %s: %v", recipient, err)
			}
		}
		return nil

	default:
		log.Printf("[consumer] unhandled topic: %s", msg.Topic)
		return nil
	}
}
