package v1

type PrimeSubscriptionResponse struct {
	Amount            int    `json:"amount"`
	CancelAt          uint   `json:"cancel_at"`
	IsPrimeSim        bool   `json:"is_prime_sim"`
	NextChargeAt      uint   `json:"next_charge_at"`
	Plan              string `json:"plan"`
	RequiresMigration bool   `json:"requires_migration"`
	SubscribedAt      uint   `json:"subscribed_at"`
	TrialEnd          uint   `json:"trial_end"`
	UserID            string `json:"user_id"`
}
