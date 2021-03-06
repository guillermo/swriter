// Package swriter implements a buffered writer that dumps Writes with a maxium frequence.
// SlowWriter will save the content of the Writes and dump after it timesout.
//
//     w := swriter.New(os.Stdout, time.Second / 60)
//     w.Write("hello")
//     w.Write("world")
//     // After 1/60 seconds it will dump to os.Stdout all the previous writes.
package swriter

import (
	"bytes"
	"io"
	"time"
)

type SlowWriter struct {
	w     io.Writer
	d     time.Duration
	t     *time.Timer
	in    chan ([]byte)
	e     error
	flush chan (struct{})
}

// New returns a new SlowWriter
func New(w io.Writer, d time.Duration) *SlowWriter {
	sw := &SlowWriter{
		w,
		d,
		nil,
		make(chan ([]byte)),
		nil,
		make(chan (struct{})),
	}
	go sw.loop()

	return sw
}

func (w *SlowWriter) loop() {
	buf := &bytes.Buffer{}

For:
	for {
		if w.t == nil {
			select {
			case data, ok := <-w.in:
				if !ok {
					break For
				}
				buf.Write(data)
				w.t = time.NewTimer(w.d)
			case <-w.flush:
			}
			continue
		}
		select {
		case data, ok := <-w.in:
			if !ok {
				w.t.Stop()
				break For
			}
			buf.Write(data)
		case <-w.flush:
			w.t.Stop()
			_, err := buf.WriteTo(w.w)
			if err != nil {
				w.e = err
			}
			buf.Reset()
			w.t = nil
		case <-w.t.C:
			_, err := buf.WriteTo(w.w)
			if err != nil {
				w.e = err
			}
			buf.Reset()
			w.t = nil
		}
	}
}

// Close stops the internal gorutines and return any error that could happen during Writes
func (w *SlowWriter) Close() error {
	if w.t != nil {
		w.t.Stop()
	}
	close(w.in)
	return w.e
}

// Write will queue the data to be send and wait a max of the time specify during creation
func (w *SlowWriter) Write(data []byte) (int, error) {
	w.in <- data
	return len(data), w.e
}

// Flush will inmediatly flush all the data
func (w *SlowWriter) Flush() {
	w.flush <- struct{}{}
}
