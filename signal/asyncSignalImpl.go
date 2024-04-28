package signal

import (
	"log"

	"github.com/Aj002Th/BlockchainEmulator/application/supervisor/supervisor_log"
)

var log1 *log.Logger = supervisor_log.Log1

// Channel和FANOUT风格的异步信号机制。
// 只能支持在进程内通信。
// Emit可能阻塞。如果回调阻塞的话。但能够保持data调用的顺序，也能防止竞争。
// 所以不要让回调阻塞。

type ChanCancel = chan int

type Val[DATA any] struct { // TODO: 加一个调试信息，比如函数名字之类，方便调试。
	cd chan DATA
	cc ChanCancel
}

func NewVal[DATA any]() Val[DATA] {
	var cd chan DATA = make(chan DATA)
	var cc ChanCancel = make(ChanCancel)
	return Val[DATA]{cd: cd, cc: cc}
}

type AsyncSignalImpl[DATA any] struct { // SO说by value在小对象的时候开销更小。然后它是深拷贝，但遇到指针，map，chan之类就会停止往下拷贝。所以这样应该是可以的。
	name        string
	outChannels map[*func(data DATA)]Val[DATA] // 从Handler到Val（cd和cc）
}

func NewAsyncSignalImpl[DATA any](name string) AsyncSignalImpl[DATA] {
	s := AsyncSignalImpl[DATA]{
		name:        name,
		outChannels: make(map[*func(data DATA)]Val[DATA]),
	}
	RegisterSig(s) // 把它注册到全局的表里。方便稍后用名字查到。
	return s
}

func (self AsyncSignalImpl[DATA]) GetName() string {
	return self.name
}

// cb在一个goroutine上依次调用，没有数据竞争。
func (self AsyncSignalImpl[DATA]) Connect(cb func(data DATA)) bool { // 到时候只准传函数和lambda，不准传方法。

	val := NewVal[DATA]()
	self.outChannels[&cb] = val
	go func() { // 运行消息队列
		for {
			log1.Printf("I'm %v, now waiting for channel %v\n", self.name, val.cd)
			select {
			case <-val.cc:
				return
			case data := <-val.cd:
				cb(data)
			}
		}
	}()
	return true
}

// 返回值：操作是否成功
func (self AsyncSignalImpl[DATA]) Disconnect(cb func(data DATA)) bool {
	self.outChannels[&cb].cc <- 1 // 发送Cancel Token
	delete(self.outChannels, &cb)
	return true
}

// 可能阻塞。如果回调阻塞的话。但能够保持data调用的顺序，也能防止竞争。
func (self AsyncSignalImpl[DATA]) Emit(data DATA) {
	for _, val := range self.outChannels {
		log1.Printf("I'm %v, sending to channel %v\n", self.name, val.cd)
		val.cd <- data
	}
}
