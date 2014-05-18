package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"net"
	"time"
)

const (
	PowerReg = 1000
	RegNum   = 27
)

type EatonValue struct {
	V []float64 `json:"v"`
	T time.Time `json:"t"`
}

func (this *EatonValue) Unit() string {
	return "kW"
}

func (this *EatonValue) Values() []float64 {
	return this.V
}

func (this *EatonValue) Time() time.Time {
	return this.T
}

func (this *EatonValue) Print() {
	spew.Dump(*this)
}

type Eaton struct {
	Addr string
}

func (this *Eaton) Read() (value EatonValue, err error) {
	conn, err := net.Dial("tcp", this.Addr)
	if err != nil {
		return
	}
	defer conn.Close()
	// send req
	buf := new(bytes.Buffer)
	buf.WriteByte(0x1) // addr
	buf.WriteByte(0x3) // func
	binary.Write(buf, binary.BigEndian, uint16(PowerReg-1))
	binary.Write(buf, binary.BigEndian, uint16(RegNum*2))
	binary.Write(buf, binary.LittleEndian, crc16(buf.Bytes()))

	buf.WriteTo(conn)
	conn.SetWriteDeadline(time.Now())
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	// read resp
	var b byte
	binary.Read(conn, binary.BigEndian, &b) //addr
	binary.Read(conn, binary.BigEndian, &b) //func
	if b != 0x3 {
		err = errors.New(fmt.Sprintf("[%v] expected %v but got %v", this.Addr, 0x3, b))
		return
	}
	binary.Read(conn, binary.BigEndian, &b) //length
	if b != RegNum*4 {
		err = errors.New(fmt.Sprintf("[%v] expected %v but got %v", this.Addr, RegNum*4, b))
		return
	}
	value.T = time.Now()
	for i := 0; i < RegNum; i++ {
		var f float32
		binary.Read(conn, binary.BigEndian, &f)
		value.V = append(value.V, float64(f))
	}
	// ignore crc16
	return
}
