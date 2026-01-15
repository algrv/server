package users

type UsageResponse struct {
	Tier      string       `json:"tier"`      // "free", "payg", "byok"
	Today     int          `json:"today"`     // Generations used today
	Limit     int          `json:"limit"`     // Daily limit (-1 for unlimited)
	Remaining int          `json:"remaining"` // Remaining generations today (-1 for unlimited)
	History   []DailyUsage `json:"history"`   // Last 30 days
}

type DailyUsage struct {
	Date  string `json:"date"`  // Format: "2006-01-02"
	Count int    `json:"count"` // Number of generations
}

type TrainingConsentRequest struct {
	TrainingConsent bool `json:"training_consent"`
}

type AIFeaturesEnabledRequest struct {
	AIFeaturesEnabled bool `json:"ai_features_enabled"`
}

type UpdateDisplayNameRequest struct {
	DisplayName string `json:"display_name" binding:"required,min=1,max=50"`
}
