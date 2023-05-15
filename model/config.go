package model

type NrpType string

const (
	NrpTypeTCP  = "tcp"
	NrpTypeHTTP = "http"
)

type NrpC struct {
	ServerIp string
	LocalIp  string
	Type     NrpType
}

type NrpS struct {
	ServerPort string
	ProxyPort  string
}
