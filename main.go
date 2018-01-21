package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/neovim/go-client/nvim"
)

func main() {
	os.Exit(run())
}

func run() int {
	addr := os.Getenv("NVIM_LISTEN_ADDRESS")
	if addr == "" {
		dir, err := ioutil.TempDir("", "nvl")
		if err != nil {
			fmt.Println(err)
			return 1
		}
		defer os.RemoveAll(dir)

		addr = filepath.Join(dir, "nvim")
		cmd := exec.Command("nvim")
		cmd.Env = append(os.Environ(), "NVIM_LISTEN_ADDRESS="+addr)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Start()
		if err != nil {
			fmt.Println(err)
			return 1
		}
		defer cmd.Wait()

		for i := 0; i < 100; i++ {
			if _, err := os.Stat(addr); err == nil {
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
	}

	nv, err := nvim.Dial(addr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer nv.Close()

	err = nv.Command("ene")
	if err != nil {
		fmt.Println(err)
		return 1
	}

	buf, err := nv.CurrentBuffer()
	if err != nil {
		fmt.Println(err)
		return 1
	}

	nv.SetBufferOption(buf, "buftype", "nofile")
	nv.Command(`nnoremap <buffer> q :bd<CR>
nnoremap <buffer> j <C-e>
nnoremap <buffer> k <C-y>
setl nonumber
echo "Loading..."`)

	win, err := nv.CurrentWindow()
	if err != nil {
		fmt.Println(err)
		return 1
	}

	w := NewNvimWriter(nv, win, buf)
	in := w.start()

	for {
		buf := make([]byte, 512*1024)
		n, err := os.Stdin.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println(err)
			return 1
		}

		in <- bytes.Split(buf[0:n], []byte{'\n'})
	}

	close(in)

	w.Wait()

	nv.Command("echo 'Read Complete.'")
	return 0
}

type NvimWriter struct {
	nv           *nvim.Nvim
	win          nvim.Window
	buf          nvim.Buffer
	iLines       int
	wait         chan bool
	isFirstWrite bool
}

func NewNvimWriter(nv *nvim.Nvim, win nvim.Window, buf nvim.Buffer) NvimWriter {
	return NvimWriter{
		nv:           nv,
		win:          win,
		buf:          buf,
		iLines:       0,
		isFirstWrite: true,
	}
}

func (w *NvimWriter) start() chan [][]byte {
	in := make(chan [][]byte, 100)
	w.wait = make(chan bool) // Create new channel. Not do in go routine.

	go func() {
		var buf [][]byte
		writeend := make(chan bool)

		writable := true
	L:
		for {
			select {
			case lines, ok := <-in:
				if !ok {
					break L
				}

				buf = append(buf, lines...)

				if writable {
					w.write(writeend, buf)
					buf = [][]byte{}
					writable = false
				}

			case <-writeend:
				writable = true
			}
		}

		if !writable {
			<-writeend
		}

		w.write(writeend, buf) // Left
		w.wait <- <-writeend
	}()

	return in
}

func (w NvimWriter) Wait() {
	<-w.wait
}

func (w *NvimWriter) write(end chan<- bool, lines [][]byte) {
	s := -1
	if w.isFirstWrite {
		s = 0
	}
	w.isFirstWrite = false

	go func() {
		defer func() {
			end <- true
		}()

		buf, err := w.nv.WindowBuffer(w.win)
		if err != nil {
			fmt.Println(err)
			return
		}

		tail := false
		var pos [2]int
		if buf == w.buf {
			var err error
			pos, err = w.nv.WindowCursor(w.win)
			if err != nil {
				fmt.Println(err)
				return
			}

			if pos[0] == w.iLines {
				tail = true
			}
		}

		err = w.nv.SetBufferLines(w.buf, s, -1, true, lines)
		if err != nil {
			fmt.Println(err)
			return
		}

		if tail {
			w.nv.SetWindowCursor(w.win, [2]int{pos[0] + len(lines), pos[1]})
		}

		w.iLines += len(lines)
	}()
}
