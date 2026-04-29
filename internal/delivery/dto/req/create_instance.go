package req

type CreateInstance struct {
	Name      string `json:"name"      validate:"required,min=3,max=50"`
	Image     string `json:"image"     validate:"required"`
	Subdomain string `json:"subdomain" validate:"required,min=3,max=50"`
}
