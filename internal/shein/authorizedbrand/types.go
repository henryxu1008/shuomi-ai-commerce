package authorizedbrand

type Config struct {
	Enabled bool
	Code    string
	Name    string
}

type Resolved struct {
	Enabled bool
	Code    string
	Name    string
	NameEn  string
}
