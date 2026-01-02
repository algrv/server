package admin

type SetUseInTrainingRequest struct {
	UseInTraining bool `json:"use_in_training"`
}

type StrudelAdminResponse struct {
	ID            string   `json:"id"`
	UserID        string   `json:"user_id"`
	Title         string   `json:"title"`
	Code          string   `json:"code"`
	IsPublic      bool     `json:"is_public"`
	UseInTraining bool     `json:"use_in_training"`
	Description   string   `json:"description,omitempty"`
	Tags          []string `json:"tags,omitempty"`
}
