package RWManager

import (
	"bytes"
	"errors"
	"io"
	"os"
)

// file manager
type LineManager struct {
	file  *os.File // file
	bufSl [][]byte // line text
	index int      // current line index
}

// create manager
func NewLineManager(file *os.File) (*LineManager, error) {
	// check if is nil
	if file == nil {
		return nil, errors.New("file is nil")
	}
	// read file
	buf := make([]byte, 512)
	_, err := file.Read(buf)
	if err != nil {
		return nil, errors.New("read file failed")
	}
	// seek back
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return nil, err
	}
	// read line to buf slice
	bufSl := bytes.Split(buf, []byte("\n"))
	return &LineManager{
		file:  file,
		bufSl: bufSl,
		index: 0,
	}, nil
}

// read line
func (f *LineManager) ReadLine() ([]byte, error) {
	// check if length is 0
	if len(f.bufSl) == 0 {
		return nil, io.EOF
	}
	// index equal or bigger than length, means file end
	if f.index >= len(f.bufSl) {
		return nil, io.EOF
	}
	// seek file
	buf := f.bufSl[f.index]
	cs := int64(len(buf) + len([]byte("\n")))
	_, err := f.file.Seek(cs, io.SeekCurrent)
	if err != nil {
		return nil, err
	}
	// return buf
	return buf, nil
}

func (f *LineManager) DelLine() error {
	// check if length is 0
	if len(f.bufSl) == 0 {
		return io.EOF
	}
	// index equal or bigger than length, means file end
	if f.index >= len(f.bufSl) {
		return io.EOF
	}
	// if index is at last place, delete the last elem
	if f.index == len(f.bufSl)-1 {
		f.bufSl = f.bufSl[:f.index-1]
	} else {
		f.bufSl = append(f.bufSl[:f.index], f.bufSl[f.index+1:]...)
	}
	// truncate
	return nil
}
