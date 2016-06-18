package main

import (
	"errors"
	"net"
	"strconv"
	"strings"
	"time"
)

// 客户端操作翻译器
type OperationTranslator interface {
	// Type 返回操作类型
	Type() uint32
	// Translate 将传入data翻译为指定类型
	//  data:二进制数据
	//  return:(已使用的字节数,翻译后的类型,错误)
	Translate(data []byte) (int, interface{}, error)
}

// 服务端状态生成器
type StatusGenerator interface {
	// Type 返回状态类别
	Type() uint32
	// Generate 将指定状态的类型转换为字节数组
	Generate(interface{}) ([]byte, error)
}

// UDP客户端
type UDPClient struct {
	Client     *net.UDPAddr //客户端地址
	Conn       *net.UDPConn //UDP连接
	LastRecv   time.Time    //上次接收到数据的时间
	LastSend   time.Time    //上次发送数据的时间
	Buffer     []byte       //发送缓冲区
	BufferSize int          //缓冲区已使用大小
}

// NewUDPClient 创建UDP客户端
func NewUDPClient(addr *net.UDPAddr, conn *net.UDPConn, bufferSize int) *UDPClient {
	return &UDPClient{
		Client:     addr,
		Conn:       conn,
		LastRecv:   time.Now(),
		LastSend:   time.Now(),
		Buffer:     make([]byte, bufferSize),
		BufferSize: 0,
	}
}

// UpdateRecvTime 更新接收包的时间
func (this *UDPClient) UpdateRecvTime() {
	this.LastRecv = time.Now()
}

// SendPackage 将数据以UDP包的形式直接发送出去
func (this *UDPClient) SendPackage(data []byte) error {
	var sent = 0
	for sent < len(data) {
		var c, err = this.Conn.WriteToUDP(data[sent:], this.Client)
		if err != nil {
			return err
		}
		sent += c
	}
	this.LastSend = time.Now()
	return nil
}

// Send 先将data存入发送缓冲区,若缓冲区满则直接发送,否则一直等待缓冲区填充
func (this *UDPClient) Send(data []byte) error {
	if len(data)+this.BufferSize > len(this.Buffer) {
		var err = this.Flush()
		if err != nil {
			return err
		}
	}
	copy(this.Buffer[this.BufferSize:], data)
	this.BufferSize += len(data)
	if this.BufferSize >= len(this.Buffer) {
		return this.Flush()
	}
	return nil
}

// Flush 无论发送缓冲区是否满,都发送数据并清空缓冲区
func (this *UDPClient) Flush() error {
	if this.BufferSize > 0 {
		var err = this.SendPackage(this.Buffer[:this.BufferSize])
		this.BufferSize = 0
		return err
	}
	return nil
}

// 操作信息
type Operation struct {
	Type   uint32      //操作类型
	Object interface{} //操作对象
}

// NewOperation 创建操作
func NewOperation(t uint32, obj interface{}) *Operation {
	return &Operation{t, obj}
}

// 状态信息
type Status struct {
	Type   uint32      //状态类型
	Object interface{} //状态对象
}

// NewStatus 创建操作
func NewStatus(t uint32, obj interface{}) *Status {
	return &Status{t, obj}
}

// UDP管理器事件
type UDPManagerEvent interface {
	// Connect 在有新的客户端连接的时候触发,返回值决定是否处理该连接的数据
	Connect(manager *UDPManager, client *UDPClient) bool
	// RecvOperation 接收到信息时触发
	RecvOperation(manager *UDPManager, client *UDPClient, op *Operation)
	// Error 处理UDP包的过程中出现错误时触发
	Error(manager *UDPManager, client *UDPClient, err error)
}

// UDP管理器,大端字节序
type UDPManager struct {
	Conn        *net.UDPConn                   //监听连接
	Addr        *net.UDPAddr                   //监听地址
	Buffer      []byte                         //接收缓冲区
	ClientsMap  map[string]int                 //客户端名称:位置映射
	Clients     []*UDPClient                   //客户端数组
	Translators map[uint32]OperationTranslator //翻译器映射,没有映射翻译器的操作信息将被抛弃
	Generators  map[uint32]StatusGenerator     //生成器映射,没有映射生成器的状态信息将被抛弃
	Event       UDPManagerEvent                //UDP管理器事件
}

// NewUDPManager 创建新的UDP管理器
//  addr:本地监听地址
//  bufferSize:UDP缓冲区大小,同时是最大UDP包大小
func NewUDPManager(addr string, bufferSize int, event UDPManagerEvent) (*UDPManager, error) {
	if event == nil {
		return nil, errors.New("event不能为空")
	}
	var pos = strings.LastIndexByte(addr, ':')
	var ip string
	var port int
	if pos >= 0 {
		ip = addr[:pos]
		var p, err = strconv.Atoi(addr[pos+1:])
		if err != nil {
			return nil, err
		}
		port = p
	} else {
		return nil, errors.New("addr必须符合ip:port的形式")
	}
	var udpAddr = &net.UDPAddr{
		IP:   net.ParseIP(ip),
		Port: port,
	}
	var udp, err = net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}
	var m = new(UDPManager)
	m.Addr = udpAddr
	m.Conn = udp
	m.Buffer = make([]byte, bufferSize)
	m.Translators = make(map[uint32]OperationTranslator, 0)
	m.Generators = make(map[uint32]StatusGenerator, 0)
	m.Clients = make([]*UDPClient, 0, 10)
	m.ClientsMap = make(map[string]int, 10)
	m.Event = event
	return m, nil
}

// AddOperationTranslator 添加操作翻译器
func (this *UDPManager) AddOperationTranslator(op OperationTranslator) {
	this.Translators[op.Type()] = op
}

// AddStatusGenerator 添加状态生成器
func (this *UDPManager) AddStatusGenerator(st StatusGenerator) {
	this.Generators[st.Type()] = st
}

// Broadcast 将状态数据广播给所有客户端
//  status:状态信息数组
func (this *UDPManager) Broadcast(status []*Status) error {
	return this.Send(this.Clients, status)
}

// Send 将状态数据发送给指定的客户端
//  clients:指定的客户端数组
//  status:状态信息数组
func (this *UDPManager) Send(clients []*UDPClient, status []*Status) error {
	var errInfo = ""
	for _, s := range status {
		var g, ok = this.Generators[s.Type]
		// 忽略没有生成器的状态信息
		if ok {
			var d, err = g.Generate(s.Object)
			if err != nil {
				errInfo += err.Error()
			} else {
				var w = NewNetworkWriter(4 + len(d))
				w.WriteUint32(s.Type)
				w.WriteBytes(d)
				d = w.Buffer()
				for _, c := range clients {
					err = c.Send(d)
					if err != nil {
						errInfo += err.Error()
					}
				}
			}
		}
	}
	for _, c := range clients {
		var err = c.Flush()
		if err != nil {
			errInfo += err.Error()
		}
	}
	if errInfo != "" {
		return errors.New(errInfo)
	}
	return nil
}

// Run 运行并开始监听UDP请求
func (this *UDPManager) Run() error {
	for {
		var c, addr, err = this.Conn.ReadFromUDP(this.Buffer)
		if err == nil {
			var name = addr.String()
			var index, ok = this.ClientsMap[name]
			var client *UDPClient
			if ok {
				client = this.Clients[index]
			} else {
				client = NewUDPClient(addr, this.Conn, len(this.Buffer))
				var index = len(this.Clients)
				this.Clients = append(this.Clients, client)
				this.ClientsMap[addr.String()] = index
				this.Event.Connect(this, client)
			}
			//读取操作数据
			client.UpdateRecvTime()
			var buf = this.Buffer[:c]
			var r = NewNetworkReader(buf)
			for r.Length() > 0 {
				// 读取类型
				var t, e = r.ReadUint32()
				if e != nil {
					this.Event.Error(this, client, err)
					break
				}
				var trans, ok = this.Translators[t]
				if !ok {
					this.Event.Error(this, client, errors.New("指定类型的解释器不存在"))
					break
				}
				var count, obj, err = trans.Translate(r.Buffer())
				if err != nil {
					this.Event.Error(this, client, err)
					break
				}
				r.Seek(count)
				this.Event.RecvOperation(this, client, NewOperation(t, obj))
			}
		} else {
			this.Event.Error(this, nil, err)
		}
	}
	return nil
}
