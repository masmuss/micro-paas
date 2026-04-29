package res

type Instance struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Subdomain string `json:"subdomain"`
	Status    string `json:"status"`
}

type Instances []*Instance
