package consumer

type UserSignupEvent struct {
	Email             string `json:"email"`
	Username          string `json:"username"`
	VerificationToken string `json:"verificationToken"`
	VerificationLink  string `json:"verificationLink"`
}

type UserOtpEvent struct {
	Email     string `json:"email"`
	Username  string `json:"username"`
	OtpCode   string `json:"otpCode"`
	OtpExpiry string `json:"otpExpiry"`
}

type BookingConfirmedEvent struct {
	Email        string `json:"email"`
	Username     string `json:"username"`
	EventName    string `json:"eventName"`
	TicketNumber string `json:"ticketNumber"`
	EventDate    string `json:"eventDate"`
}

type BookingCancelledEvent struct {
	Email     string `json:"email"`
	Username  string `json:"username"`
	EventName string `json:"eventName"`
	BookingId string `json:"bookingId"`
}
