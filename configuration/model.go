package configuration

type Config struct {
	EntryPoint []*EntryPoint
	Frontend   []*Frontend
	Backend    []*Backend
}

type EntryPoint struct {
	Protocol string
	Enabled  bool
	Addr     string
}

type Frontend struct {
	Protocol  string
	Host      string
	To        string
	ToBackend *Backend
}

type Backend struct {
	Name          string
	Protocol      string
	Url           string
	DiscoveryType string
}
