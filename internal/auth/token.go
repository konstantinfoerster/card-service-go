package auth

type JWT struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	Scope        string `json:"scope"`
	Type         string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	Provider     string `json:"provider"`
}

type Claim struct {
	UserID string
	Email  string
}

func NewClaim(userID, email string) Claim {
	return Claim{UserID: userID, Email: email}
}
