package consumer

// Topics is the canonical list of notification topics this service subscribes
// to. It is consumed by the broker layer when wiring up the consumer group.
var Topics = []string{
	"eventslk.user.signup",
	"eventslk.user.otp",
	"eventslk.booking.confirmed",
	"eventslk.booking.cancelled",
	"eventslk.promotion",
}
