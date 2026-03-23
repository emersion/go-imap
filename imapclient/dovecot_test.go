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
	cfg := `dovecot_config_version  = 2.4.0
dovecot_storage_version = 2.4.0

log_path      = "` + tempDir + `/dovecot.log"
ssl           = no
mail_home     = "` + tempDir + `/%{user}"
mail_driver   = maildir
mail_path     = "~/Mail"

namespace inbox {
	separator = /
	prefix =
	inbox = yes
}

mail_plugins {
	acl = yes
}

protocol imap {
	mail_plugins {
		imap_acl = yes
	}
}

acl_driver = vfile
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
