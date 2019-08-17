package main

import (
	"fmt"
	"log"

	"github.com/fsouza/go-dockerclient"
	"github.com/gliderlabs/ssh"
)

func main() {
	server := ssh.Server{
		Addr: ":2222",
		LocalPortForwardingCallback: ssh.LocalPortForwardingCallback(func(ctx ssh.Context, host string, port uint32) bool {
			ctx.SetValue("type", "local-port-forwarding")
			log.Println("attempt to bind", host, port, "granted")
			return true
		}),
		ReversePortForwardingCallback: ssh.ReversePortForwardingCallback(func(ctx ssh.Context, host string, port uint32) bool {
			log.Println("attempt to bind", host, port, "granted")
			ctx.SetValue("type", "remote-port-forwarding")
			return true
		}),
		Handler: func(sess ssh.Session) {
			log.Println(sess.Context().Value("type"))
			if str, ok := sess.Context().Value("type").(string); ok {
				if str == "local-port-forwarding" || str == "remote-port-forwarding" {
					ctx := sess.Context()
					for {
						select {
						case <-ctx.Done():
							if err := ctx.Err(); err != nil {
								log.Println(err)
								sess.Exit(1)
								return
							}
							sess.Close()
							return
						}
					}

				}
			}
			log.Println("remote local", sess.RemoteAddr(), sess.LocalAddr())
			status, err := dockerExec(sess)
			if err != nil {
				fmt.Fprintln(sess, err)
				log.Println(err)
			}
			sess.Exit(int(status))
		},
	}
	log.Println("starting ssh server on port 2222...")
	log.Println("you can try `$CONTAINER_ID@localhost -p 2222`")
	log.Fatal(server.ListenAndServe())
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
