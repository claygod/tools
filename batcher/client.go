package batcher

// Batcher
// Client
// Copyright © 2018 Eduard Sesigin. All rights reserved. Contacts: <claygod@yandex.ru>

import (
	// "fmt"
	"log"
	"os"
)

/*
Client - for ease of use of the batcher in the typis case.
*/
type Client struct {
	b *Batcher
	f *os.File
	//chIn chan []byte
}

/*
Open - client creation and batcher.
*/
func Open(filePath string, batchSize int) (*Client, error) {
	f, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}

	chIn := make(chan []byte, batchSize)
	nb := NewBatcher(newWriter(f), alarm, chIn, batchSize)
	nb.Start()

	return &Client{
		b: nb,
		f: f,
		//chIn: chIn,
	}, nil
}

/*
Write -
*/
func (c *Client) Write(in []byte) {

	//fmt.Println("step 1")
	c.b.chInput <- in
	//fmt.Println("step 2")

	//fmt.Println("step 3")
	ch := c.b.GetChan()
	<-ch
	//fmt.Println("step 4")
}

func (c *Client) Close() {
	c.b.Stop()
}

/*
alarm - errors log.
*/
func alarm(err error) {
	log.Print(err)
}

/*
writer - intermediate structure for writing to file.
*/
type writer struct {
	f *os.File
}

/*
newWriter - create new filewriter.
*/
func newWriter(f *os.File) *writer {
	return &writer{
		f: f,
	}
}

/*
Write - write data to a file with synchronization
*/
func (w *writer) Write(in []byte) (int, error) {
	i, err := w.f.Write(in)
	i, err = w.f.Write([]byte{99})
	if err == nil {
		err = w.f.Sync()
	}

	return i, err
}
