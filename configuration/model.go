package configuration

type Config struct {
	Servers   []*Server
	Endpoints []*Endpoint
	Services  []*Service
}

type Server struct {
	Protocol string
	Enabled  bool
	Addr     string
}

type Endpoint struct {
	Protocol  string
	Host      string
	To        string
	ToService *Service
}

type Service struct {
	Name     string
	Protocol string
	Url      string
}
