package notifications

type server struct{}

func NewServer() *server {
	return &server{}
}
