package mahakam

type NetworkFramework string

const (
	HTTP    NetworkFramework = "net/http"
	NET     NetworkFramework = "net"
	NETPOLL NetworkFramework = "github.com/cloudwego/netpoll"
)

func (nf NetworkFramework) String() string {
	return string(nf)
}

func (nf NetworkFramework) IsValid() bool {
	switch nf {
	case HTTP, NET, NETPOLL:
		return true
	default:
		return false
	}
}
