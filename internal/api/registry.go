package api

// AppRegistryResponse represents a registry from the TrueNAS API.
type AppRegistryResponse struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Username    string  `json:"username"`
	Password    string  `json:"password"`
	URI         string  `json:"uri"`
}
