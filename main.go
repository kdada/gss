package main

import (
	"errors"
	"fmt"
)

var game *Game

type TestEvent struct {
}

// Connect 在有新的客户端连接的时候触发,返回值决定是否处理该连接的数据
func (this *TestEvent) Connect(manager *UDPManager, client *UDPClient) bool {
	fmt.Println("新客户端连接:", client.Client.String())
	game = NewGame(manager, client)
	go game.Run()
	return true
}

// RecvOperation 接收到信息时触发
func (this *TestEvent) RecvOperation(manager *UDPManager, client *UDPClient, op *Operation) {
	if game != nil && op.Type == 1 {
		game.AddOperation(client, op)
	}
}

// Error 处理UDP包的过程中出现错误时触发
func (this *TestEvent) Error(manager *UDPManager, client *UDPClient, err error) {
	fmt.Println(err)
}

// 客户端操作翻译器
type TestOperationTranslator struct {
}

// Type 返回操作类型
func (this *TestOperationTranslator) Type() uint32 {
	return 0
}

// Translate 将传入data翻译为指定类型
//  data:二进制数据
//  return:(已使用的字节数,翻译后的类型,错误)
func (this *TestOperationTranslator) Translate(data []byte) (int, interface{}, error) {
	if len(data) < 4 {
		return 0, nil, errors.New("解析失败，类型0的data需要4个字节")
	}
	return 4, uint32(data[0])<<24 + uint32(data[1])<<16 + uint32(data[2])<<8 + uint32(data[3]), nil
}

// 客户端操作翻译器
type MoveOperationTranslator struct {
}

// Type 返回操作类型
func (this *MoveOperationTranslator) Type() uint32 {
	return 1
}

// Translate 将传入data翻译为指定类型
//  data:二进制数据
//  return:(已使用的字节数,翻译后的类型,错误)
func (this *MoveOperationTranslator) Translate(data []byte) (int, interface{}, error) {
	var d = Direction{}
	var r = NewNetworkReader(data)
	d.x, _ = r.ReadFloat32()
	d.y, _ = r.ReadFloat32()
	return r.Pos, d, nil
}

// 客户端操作翻译器
type MoveStatusGenerator struct {
}

// Type 返回操作类型
func (this *MoveStatusGenerator) Type() uint32 {
	return 1
}

func (this *MoveStatusGenerator) Generate(a interface{}) ([]byte, error) {
	var p = a.(Player)
	var w = NewNetworkWriter(8)
	w.WriteFloat32(p.x)
	w.WriteFloat32(p.y)
	return w.Buffer(), nil
}

func main() {
	fmt.Println("服务端启动")
	var manager, err = NewUDPManager(":10086", 1024, new(TestEvent))
	if err != nil {
		fmt.Println(err)
	} else {
		manager.AddOperationTranslator(new(TestOperationTranslator))
		manager.AddOperationTranslator(new(MoveOperationTranslator))
		manager.AddStatusGenerator(new(MoveStatusGenerator))
		manager.Run()
	}
	fmt.Println("服务端停止")
}
