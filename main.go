package main

import (
	"fmt"
	"log"

	"github.com/fsouza/go-dockerclient"
	"github.com/gliderlabs/ssh"
)

func main() {
	ssh.Handle(func(sess ssh.Session) {
		status, err := dockerExec(sess)
		if err != nil {
			fmt.Fprintln(sess, err)
			log.Println(err)
		}
		sess.Exit(int(status))
	})

	log.Println("starting ssh server on port 2222...")
	log.Println("you can try `$CONTAINER_ID@localhost -p 2222`")
	log.Fatal(ssh.ListenAndServe(":2222", nil))
}

func dockerExec(sess ssh.Session) (status int, err error) {
	_, _, isTty := sess.Pty()
	status = 255
	client, err := docker.NewClientFromEnv()
	if err != nil {
		panic(err)
	}
	cmd := sess.Command()
	if len(cmd) == 0 {
		cmd = []string{"bin/sh"}
	}
	// runs properly. Using bash does not seem like an elegant solution,
	// but this is the best so far.
	de := docker.CreateExecOptions{
		AttachStderr: true,
		AttachStdin:  true,
		AttachStdout: true,
		Tty:          isTty,
		Cmd:          cmd,
		Container:    sess.User(),
	}
	dExec, err := client.CreateExec(de)
	if err != nil {
		status = 1
		return
	}
	execID := dExec.ID

	opts := docker.StartExecOptions{
		OutputStream: sess,
		ErrorStream:  sess.Stderr(),
		InputStream:  sess,
		RawTerminal:  isTty,
	}
	cw, err := client.StartExecNonBlocking(execID, opts)
	if err != nil {
		status = 1
		return
	}

	cw.Wait()
	return
}
