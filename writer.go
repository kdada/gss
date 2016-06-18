package main

import (
	"math"
)

// 网络数据写入器(大端子节序)
type NetworkWriter struct {
	Data []byte //数据
	Len  int    //数据长度
}

// NewNetworkWriter 创建网络数据写入器
func NewNetworkWriter(bufferSize int) *NetworkWriter {
	return &NetworkWriter{
		make([]byte, bufferSize),
		0,
	}
}

// Buffer 返回当前缓冲区
func (this *NetworkWriter) Buffer() []byte {
	return this.Data[:this.Length()]
}

// Length 返回当前缓冲区大小
func (this *NetworkWriter) Length() int {
	return this.Len
}

// extend 将当前缓冲区扩大为2倍大小
func (this *NetworkWriter) extend() {
	var d = this.Data
	this.Data = make([]byte, len(d)*2)
	copy(this.Data, d[:this.Len])
}

// WriteBytes 写入字节数组
func (this *NetworkWriter) WriteBytes(data []byte) {
	if (len(this.Data) - this.Len) < len(data) {
		this.extend()
	}
	copy(this.Data[this.Len:], data)
	this.Len += len(data)
}

// WriteByte 写入字节
func (this *NetworkWriter) WriteByte(data byte) {
	this.WriteBytes([]byte{data})
}

// WriteUint32 写入4字节整数
func (this *NetworkWriter) WriteUint32(data uint32) {
	var b = make([]byte, 4)
	for i := uint(0); i < 4; i++ {
		b[3-i] = byte(data >> (i * 8))
	}
	this.WriteBytes(b)
}

// WriteFloat32 写入4字节浮点数
func (this *NetworkWriter) WriteFloat32(data float32) {
	this.WriteUint32(math.Float32bits(data))
}
