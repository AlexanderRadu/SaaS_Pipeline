package status

type Instance struct {
	ID            string
	UserID        int64
	ServiceName   string
	Status        string
	ConnectionURL *string
	CreatedAt     string
}
