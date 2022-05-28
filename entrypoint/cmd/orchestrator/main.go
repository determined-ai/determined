package main

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
	"strings"

	"fmt"
)

type Line = []byte

func encodeBytes(byts []byte) string {
	out := ""
	for _, b := range byts {
		if b == '\n' {
			out = out + "\\n"
		} else {
			out = out + string(b)
		}
	}
	return out
}

func readLines(r *bufio.Reader, out chan<-Line) {
	defer func(){out<-nil}()
	for {
		// err is set IFF the delimiter was not found
		byts, err := r.ReadBytes('\n')
		// ReadBytes will return partial reads, even if it returns an error
		if len(byts) > 0 {
			out <- byts
		}
		if err != nil {
			// EIO means the child process is done.
			// fs.PathError("file already closed") means the underlying fd is closed.
			switch {
			case err == io.EOF:
				// EOF is how we expect to finish every stream
			case err == syscall.EIO:
				// EIO means the child process is done.
			case strings.Contains(err.Error(), "file already closed"):
				// fs.PathError("file already closed") means the underlying fd is closed.
			default:
				// XXX: report this error in logs
				log.Fatal(fmt.Sprintf("%T: %s\n", err, err))
			}
			return
		}
	}
}

func prepLines(buffer []Line) []byte {
	return bytes.Join(buffer, []byte(""))
}

func postLogs(logs []byte) error {
	print(string(logs))
	return nil
}

// bufferLines could write lines to disk to keep them out of RAM if we had too much to write,
// but for now it just passes lines through FIFO-style
func bufferLines(in <-chan Line, out chan<- Line) {
	defer func(){out<-nil}()
	// we expect an EOF on both stdout and on stderr
	wantExits := 2
	for wantExits > 0 {
		line := <-in
		if line == nil {
			wantExits--
		} else {
			out <- line
		}
	}
}

// group lines into large submissions, either on a tick or when too many have built up
func submitLines(lines <-chan Line, giveUp <-chan time.Time, done chan<- bool){
	defer func(){done <- true}()

	timeout := 100 * time.Millisecond
	tick := time.NewTimer(timeout)
	defer tick.Stop()

	maxLines := 1000
	buffer := make([]Line, 0, 1000)
	var giveUpTime time.Time

	keepGoing := func() bool {
		if giveUpTime == (time.Time{}) {
			// giveUpTime not yet set
			return true
		}
		return time.Now().Before(giveUpTime)
	}

	for keepGoing() {
		// either fill our buffer or wait for a timeout
		for len(buffer) < maxLines {
			select {
			case line := <- lines:
				if line == nil {
					goto noMoreLines
				}
				buffer = append(buffer, line)
			case <-tick.C:
				tick.Reset(timeout)
				if len(buffer) > 0 {
					goto submit
				}
			case giveUpTime = <-giveUp:
			}
		}

	submit:
		logs := prepLines(buffer)
		for keepGoing() {
			err := postLogs(logs)
			if err != nil {
				continue
			}
			// preserve the memory of our buffer
			buffer = buffer[:0]
			break
		}
	}

noMoreLines:
	logs := prepLines(buffer)
	for keepGoing() {
		err := postLogs(logs)
		if err != nil {
			continue
		}
		break
	}
}

func handleSignals(p *os.Process, dead <-chan bool, done chan<- bool){
	defer func(){done<-true}()
	sigs := make(chan os.Signal, 1)
	// see https://www-uxsup.csx.cam.ac.uk/courses/moved.Building/signals.pdf
	signal.Notify(
		sigs,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGQUIT,
		syscall.SIGTRAP,
		syscall.SIGUSR1,
		syscall.SIGUSR2,
	)
	select {
	case sig := <-sigs:
		err := p.Signal(sig)
		if err != nil {
			// XXX: what can we do here?
		}
	case <-dead:
		return
	}
}

func main() {
	if len(os.Args) < 2 {
		os.Exit(44)
	}

	// Create the command with stdout and stderr attached to pipes.
	cmd := exec.Command(os.Args[1], os.Args[2:]...)
	so, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	stdout := bufio.NewReader(so)

	se, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}
	stderr := bufio.NewReader(se)

	/* Log line flow:
		readLines(stderr)
		                 \ via lines               via buffered
		                  -----------> bufferLines() ------------> submitLines()
		                 /
		readLines(stderr)
	*/
	lines := make(chan Line)
	buffered := make(chan Line)
	giveUp := make(chan time.Time, 1)
	done := make(chan bool, 2)
	go readLines(stdout, lines)
	go readLines(stderr, lines)
	go bufferLines(lines, buffered)
	go submitLines(buffered, giveUp, done)

	// start command
	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
	}

	// Pass signals to child.
	dead := make(chan bool, 1)
	go handleSignals(cmd.Process, dead, done)

	// wait for child to exit
	cmd.Wait()
	exitCode := 0
	if cmd.ProcessState == nil {
		// Somehow, .Wait() failed.
		exitCode = 45
	} else {
		// convert ProcessState.Sys() to a Unix-specific WaitStatus to get exit status.
		waitStatus := cmd.ProcessState.Sys().(syscall.WaitStatus)
		if waitStatus.Exited() {
			// Normal exit, preserve exit code.
			exitCode = waitStatus.ExitStatus()
		} else if waitStatus.Signaled() {
			// Exit due to a signal.  This is how shells traditionally encode signal exits into
			// 8-bit exit codes.  See https://tldp.org/LDP/abs/html/exitcodes.html.
			exitCode = 128 + int(waitStatus.Signal())
		} else {
			// Process was stopped (this should never happen to us, see man 2 wait) or linux broke.
			exitCode = 46
		}
	}

	// tell the signal handler to stop catching signals
	dead <- true

	// tell submitLines to stop trying if it's not done in 30 seconds
	giveUpTime := time.Now().Add(30 * time.Second)
	giveUp <- giveUpTime

	// wait for submitLines and handleSignals to both finish
	<-done
	<-done

	os.Exit(exitCode)
}
