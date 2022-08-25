package args

import (
	"net"
	"sync"
)

var lbBuilder = &builder{holder: lbHolder}

type builder struct {
	l sync.RWMutex
	*holder
}

func (b *builder) SetPort(port uint) *builder {
	b.l.Lock()
	defer b.l.Unlock()
	b.port = port
	return b
}

func (b *builder) SetBindAddress(bindAddress net.IP) *builder {
	b.l.Lock()
	defer b.l.Unlock()
	b.bindAddress = bindAddress
	return b
}

func (b *builder) SetKubeconfig(kubeconfig string) *builder {
	b.l.Lock()
	defer b.l.Unlock()
	b.kubeconfig = kubeconfig
	return b
}

func (b *builder) SetLogLevel(logLevel string) *builder {
	b.l.Lock()
	defer b.l.Unlock()
	b.logLevel = logLevel
	return b
}

func (b *builder) SetLogFormat(logFormat string) *builder {
	b.l.Lock()
	defer b.l.Unlock()
	b.logFormat = logFormat
	return b
}

func (b *builder) SetLogFile(logFile string) *builder {
	b.l.Lock()
	defer b.l.Unlock()
	b.logFile = logFile
	return b
}

func (b *builder) SetUpstream(upstream []string) *builder {
	b.l.Lock()
	defer b.l.Unlock()
	for _, host := range upstream {
		b.upstream = append(b.upstream, host)
	}
	return b
}

func (b *builder) SetNumWorker(numWorker uint) *builder {
	b.l.Lock()
	defer b.l.Unlock()
	b.numWorker = numWorker
	return b
}

func NewBuilder() *builder { return lbBuilder }
