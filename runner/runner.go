package runner

import (
	"bufio"
	"fmt"
	"github.com/nats-io/nats.go"
	"io"
	"os"
	"os/exec"
)

type Runner struct {
	name string
	cmd  *exec.Cmd
	con  *nats.Conn

	stdErrPipe io.ReadCloser
	stdOutPipe io.ReadCloser
	stdInPipe  io.WriteCloser
	stdInSub   *nats.Subscription
}

func New(name string, cmd *exec.Cmd, con *nats.Conn) *Runner {
	return &Runner{name: name, cmd: cmd, con: con}
}

func (r *Runner) publisher(pipeName string, pipe io.ReadCloser) {
	defer func(pipe io.ReadCloser) { _ = pipe.Close() }(pipe)
	publish := func(text string) {
		if err := r.con.Publish(r.name+"."+pipeName, []byte(text)); err != nil {
			fmt.Printf(pipeName+" publish error: %v\n", err)
			return
		}
		if err := r.con.Flush(); err != nil {
			fmt.Printf(pipeName+" flush error: %v\n", err)
		}
	}
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		publish(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		fmt.Printf(pipeName+" scanner error: %v\n", err)
	}
}

func (r *Runner) subscriber(pipeName string, pipe io.WriteCloser) (err error) {
	subName := r.name + "." + pipeName
	r.stdInSub, err = r.con.Subscribe(subName, func(msg *nats.Msg) {
		if _, writeErr := pipe.Write(msg.Data); writeErr != nil {
			fmt.Printf("subscriber write error: %v\n", writeErr)
			return
		}
	})
	return err
}

func (r *Runner) Signal(signal os.Signal) (err error) {
	if r.cmd == nil || r.cmd.Process == nil {
		return fmt.Errorf("process not started")
	}
	return r.cmd.Process.Signal(signal)
}

func (r *Runner) Wait() (err error) {

	logError := func(name string, err error) {
		if err != nil {
			fmt.Printf("%v error: %v\n", name, err)
		}
	}
	err = r.cmd.Wait()
	if r.stdInSub != nil {
		logError("stdInSub.Unsubscribe()", r.stdInSub.Unsubscribe())
		r.stdInSub = nil
	}
	return err
}

func (r *Runner) Status() string {
	if r.cmd == nil || r.cmd.Process == nil {
		return "not started"
	}

	if r.cmd.ProcessState == nil {
		return "started"
	}

	if r.cmd.ProcessState.Exited() {
		return "exited"
	}

	return "unknown"
}

func (r *Runner) Start() (err error) {

	if r.cmd.Process != nil && r.cmd.ProcessState == nil {
		return fmt.Errorf("process started and not yet exited")
	}

	// if it ran and exited, then re-create the command
	if r.cmd.Process != nil && r.cmd.ProcessState.Exited() {
		dir := r.cmd.Dir
		r.cmd = exec.Command(r.cmd.Path, r.cmd.Args[1:]...)
		r.cmd.Dir = dir
	}

	fmt.Println("runner.Start()", r.cmd.Path, r.cmd.Args[1:])

	if r.stdErrPipe, err = r.cmd.StderrPipe(); err != nil {
		return err
	}

	if r.stdOutPipe, err = r.cmd.StdoutPipe(); err != nil {
		return err
	}

	if r.stdInPipe, err = r.cmd.StdinPipe(); err != nil {
		return err
	}

	go r.publisher("stderr", r.stdErrPipe)
	go r.publisher("stdout", r.stdOutPipe)
	if err = r.subscriber("stdin", r.stdInPipe); err != nil {
		return err
	}

	return r.cmd.Start()
}
