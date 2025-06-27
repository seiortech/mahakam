package mahakam

type NetworkFramework string

const (
	HTTP    NetworkFramework = "net/http"                     // Standard HTTP package
	NET     NetworkFramework = "net"                          // Standard net package
	NETPOLL NetworkFramework = "github.com/cloudwego/netpoll" // Netpoll package, a high-performance network library (https://github.com/cloudwego/netpoll)
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
