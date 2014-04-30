package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	//"fmt"
	"github.com/davecgh/go-spew/spew"
	"net"
	"time"
)

type MeterValue interface {
	Time() time.Time
	Unit() string
	Values() []float64
	Print()
}

type Meter interface {
	Read() (MeterValue, error)
}

const (
	Addr       = "meter2518-1.seas.ucla.edu:4660"
	PowerReg   = 1000
	RegNum     = 27
)

type VerisValue struct {
	V []float64
	T time.Time
}

func (this *VerisValue) Unit() string {
	return "kW"
}

func (this *VerisValue) Values() []float64 {
	return this.V
}

func (this *VerisValue) Time() time.Time {
	return this.T
}

func (this *VerisValue) Print() {
	spew.Dump(*this)
}

type Veris struct{}

func (this *Veris) Read() (value MeterValue, err error) {
	conn, err := net.Dial("tcp", Addr)
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
		err = errors.New("bad response")
		return
	}
	binary.Read(conn, binary.BigEndian, &b) //length
	if b != RegNum*4 {
		err = errors.New("bad reg num")
		return
	}
	v := new(VerisValue)
	v.T = time.Now()
	for i := 0; i < RegNum; i++ {
		var f float32
		binary.Read(conn, binary.BigEndian, &f)
		v.V = append(v.V, float64(f))
	}
	// ignore crc16
	value = v
	return
}
