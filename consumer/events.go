package consumer

type UserSignupEvent struct {
	Email             string `json:"email"`
	Username          string `json:"username"`
	VerificationToken string `json:"verificationToken"`
	VerificationLink  string `json:"verificationLink"`
}

type UserOtpEvent struct {
	Email         string `json:"email"`
	Username      string `json:"username"`
	OtpCode       string `json:"otpCode"`
	ExpiryMinutes int    `json:"expiryMinutes"`
}

type BookingConfirmedEvent struct {
	Email        string `json:"email"`
	Username     string `json:"username"`
	EventName    string `json:"eventName"`
	TicketNumber string `json:"ticketNumber"`
	EventDate    string `json:"eventDate"`
	BookingId    string `json:"bookingId"`
}

type BookingCancelledEvent struct {
	Email     string `json:"email"`
	Username  string `json:"username"`
	EventName string `json:"eventName"`
	BookingId string `json:"bookingId"`
}

type PromotionEvent struct {
	RecipientEmails []string `json:"recipientEmails"`
	Subject         string   `json:"subject"`
	MessageBody     string   `json:"messageBody"`
	CtaLink         string   `json:"ctaLink"`
}
