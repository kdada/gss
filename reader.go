package main

import (
	"errors"
	"math"
)

// 网络数据读取器(大端子节序)
type NetworkReader struct {
	Data []byte //数据
	Pos  int    //位置
}

// NewNetworkReader 创建网络数据读取器
func NewNetworkReader(data []byte) *NetworkReader {
	return &NetworkReader{data, 0}
}

// Buffer 返回当前可读取缓冲区
func (this *NetworkReader) Buffer() []byte {
	return this.Data[this.Pos:]
}

// Length 返回当前可读取数据长度
func (this *NetworkReader) Length() int {
	return len(this.Data) - this.Pos
}

// Seek 移动位置,length为正数则向数据尾部移动,否则向数据头部移动
func (this *NetworkReader) Seek(length int) {
	this.Pos += length
	if this.Pos < 0 {
		this.Pos = 0
	}
	if this.Pos > len(this.Data) {
		this.Pos = len(this.Data)
	}
}

// ReadBytes 读取指定数量的字节
func (this *NetworkReader) ReadBytes(count int) ([]byte, error) {
	if this.Length() >= count {
		this.Pos += count
		return this.Data[this.Pos-count : this.Pos], nil
	}
	return nil, errors.New("NetworkReader:可读取数据长度不足")
}

// ReadByte 读取一个字节的数据
func (this *NetworkReader) ReadByte() (byte, error) {
	var b, err = this.ReadBytes(1)
	if err == nil {
		return b[0], nil
	}
	return 0, err
}

// ReadUint32 读取4字节数据并转换为uint32
func (this *NetworkReader) ReadUint32() (uint32, error) {
	var b, err = this.ReadBytes(4)
	if err != nil {
		return 0, err
	}
	return uint32(b[0])<<24 + uint32(b[1])<<16 + uint32(b[2])<<8 + uint32(b[3]), nil
}

// ReadFloat32 读取4字节数据并转换为float32
func (this *NetworkReader) ReadFloat32() (float32, error) {
	var c, err = this.ReadUint32()
	if err != nil {
		return 0, err
	}
	return math.Float32frombits(c), nil
}
