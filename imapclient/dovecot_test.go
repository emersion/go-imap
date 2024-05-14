package imapclient_test

import (
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func newDovecotClientServerPair(t *testing.T) (net.Conn, io.Closer) {
	tempDir := t.TempDir()

	cfgFilename := filepath.Join(tempDir, "dovecot.conf")
	cfg := `log_path      = "` + tempDir + `/dovecot.log"
ssl           = no
mail_home     = "` + tempDir + `/%u"
mail_location = maildir:~/Mail

namespace inbox {
	separator = /
	prefix =
	inbox = yes
}

mail_plugins = $mail_plugins acl
protocol imap {
	mail_plugins = $mail_plugins imap_acl
}
plugin {
  acl = vfile
}
`
	if err := os.WriteFile(cfgFilename, []byte(cfg), 0666); err != nil {
		t.Fatalf("failed to write Dovecot config: %v", err)
	}

	clientConn, serverConn := net.Pipe()

	cmd := exec.Command("doveadm", "-c", cfgFilename, "exec", "imap")
	cmd.Env = []string{"USER=" + testUsername, "PATH=" + os.Getenv("PATH")}
	cmd.Dir = tempDir
	cmd.Stdin = serverConn
	cmd.Stdout = serverConn
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start Dovecot: %v", err)
	}

	return clientConn, &dovecotServer{cmd, serverConn}
}

type dovecotServer struct {
	cmd  *exec.Cmd
	conn net.Conn
}

func (srv *dovecotServer) Close() error {
	if err := srv.conn.Close(); err != nil {
		return err
	}
	return srv.cmd.Wait()
}
