package catalog

type Service struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	DockerImage string `json:"docker_image"`
	ExposedPort string `json:"exposed_port"`
	Scheme      string `json:"scheme"`
	DefaultUser string `json:"default_user"`
}
