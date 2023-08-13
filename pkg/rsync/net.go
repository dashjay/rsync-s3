package rsync

import (
	"encoding/binary"
	"fmt"
	"io"
)

type Conn struct {
	writer io.WriteCloser // Write only
	reader io.ReadCloser  // Read only
}

func (conn *Conn) Write(p []byte) (n int, err error) {
	return conn.writer.Write(p)
}

func (conn *Conn) Read(p []byte) (n int, err error) {
	return conn.reader.Read(p)
}

func (conn *Conn) ReadByte() (byte, error) {
	var buf [1]byte
	n, err := io.ReadFull(conn, buf[:])
	if err != nil {
		return 0, err
	}
	if n != 1 {
		return buf[0], fmt.Errorf("should read %d but read %d", 1, n)
	}
	return buf[0], nil
}

func (conn *Conn) ReadShort() (int16, error) {
	var i int16
	err := binary.Read(conn, binary.LittleEndian, &i)
	return i, err
}

func (conn *Conn) ReadInt() (int32, error) {
	var i int32
	err := binary.Read(conn, binary.LittleEndian, &i)
	return i, err
}

func (conn *Conn) ReadLong() (int64, error) {
	var i int64
	err := binary.Read(conn, binary.LittleEndian, &i)
	return i, err
}

func (conn *Conn) ReadVarInt() (int64, error) {
	sVal, err := conn.ReadInt()
	if err != nil {
		return 0, err
	}
	if sVal != -1 {
		return int64(sVal), nil
	}
	return conn.ReadLong()
}

func (conn *Conn) WriteByte(data byte) error {
	return binary.Write(conn.writer, binary.LittleEndian, data)
}

func (conn *Conn) WriteShort(data int16) error {
	return binary.Write(conn.writer, binary.LittleEndian, data)
}

func (conn *Conn) WriteInt(data int32) error {
	return binary.Write(conn.writer, binary.LittleEndian, data)
}

func (conn *Conn) WriteLong(data int64) error {
	return binary.Write(conn.writer, binary.LittleEndian, data)
}

func (conn *Conn) Close() error {
	_ = conn.writer.Close()
	_ = conn.reader.Close()
	return nil
}
