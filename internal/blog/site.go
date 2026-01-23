package blog

type SiteProfile struct {
	Title        string   `json:"title"`
	Tagline      string   `json:"tagline"`
	Intro        string   `json:"intro"`
	Location     string   `json:"location"`
	Email        string   `json:"email"`
	Newsletter   string   `json:"newsletter"`
	CurrentFocus []string `json:"current_focus"`
}
