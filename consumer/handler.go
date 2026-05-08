package consumer

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/IBM/sarama"
	"eventslk-notification-service/email"
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
			return fmt.Errorf("unmarshal UserSignupEvent: %w", err)
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
			return fmt.Errorf("unmarshal UserOtpEvent: %w", err)
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
			return fmt.Errorf("unmarshal BookingConfirmedEvent: %w", err)
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
			return fmt.Errorf("unmarshal BookingCancelledEvent: %w", err)
		}
		return h.emailSender.SendEmail(email.EmailMessage{
			To:           event.Email,
			Subject:      "Booking Cancelled — " + event.EventName,
			TemplateName: "booking_cancelled.html",
			TemplateData: event,
		})

	default:
		log.Printf("[consumer] unhandled topic: %s", msg.Topic)
		return nil
	}
}
