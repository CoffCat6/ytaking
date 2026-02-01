package blog

type SiteProfile struct {
	Title        string       `json:"title"`
	Tagline      string       `json:"tagline"`
	Intro        string       `json:"intro"`
	Positioning  string       `json:"positioning"`
	Skills       []string     `json:"skills"`
	Avatar       string       `json:"avatar"`
	AvatarPosX   float64      `json:"avatar_pos_x"`
	AvatarPosY   float64      `json:"avatar_pos_y"`
	AvatarScale  float64      `json:"avatar_scale"`
	Location     string       `json:"location"`
	Email        string       `json:"email"`
	Newsletter   string       `json:"newsletter"`
	CurrentFocus []string     `json:"current_focus"`
	SocialLinks  []SocialLink `json:"social_links"`
}

type SocialLink struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}
