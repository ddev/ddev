// Non-interactive exec through the moby Go client, mirroring DDEV's
// dockerutil.Exec without stdin. Usage: execrepro <container> [cmd] ;
// set CLOSEWRITE=1 to half-close like the docker CLI does.
package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/client"
)

func main() {
	ctx := context.Background()
	c, err := client.New(client.FromEnv)
	if err != nil {
		panic(err)
	}
	cmd := "echo hi"
	if len(os.Args) > 2 {
		cmd = os.Args[2]
	}
	ec, err := c.ExecCreate(ctx, os.Args[1], client.ExecCreateOptions{
		Cmd:          []string{"sh", "-c", cmd},
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		panic(err)
	}
	start := time.Now()
	att, err := c.ExecAttach(ctx, ec.ID, client.ExecAttachOptions{})
	if err != nil {
		panic(err)
	}
	defer att.Close()
	if os.Getenv("CLOSEWRITE") == "1" {
		_ = att.CloseWrite()
	}
	var out, errb bytes.Buffer
	_, err = stdcopy.StdCopy(&out, &errb, att.Reader)
	elapsed := time.Since(start)
	info, ierr := c.ExecInspect(ctx, ec.ID, client.ExecInspectOptions{})
	code := -999
	if ierr == nil {
		code = info.ExitCode
	}
	s := out.String()
	if len(s) > 40 {
		s = s[:40] + "..."
	}
	fmt.Printf("stdout_bytes=%d stderr_bytes=%d exit=%d err=%v elapsed=%s stdout=%q\n", out.Len(), errb.Len(), code, err, elapsed.Round(time.Millisecond), s)
}
