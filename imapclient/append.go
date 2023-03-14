package imapclient

import (
	"io"
	"time"

	"github.com/emersion/go-imap/v2"
)

// AppendOptions contains options for the APPEND command.
type AppendOptions struct {
	Flags []imap.Flag
	Time  time.Time
}

// Append sends an APPEND command.
//
// The caller must call AppendCommand.Close.
//
// The options are optional.
func (c *Client) Append(mailbox string, size int64, options *AppendOptions) *AppendCommand {
	cmd := &AppendCommand{}
	cmd.enc = c.beginCommand("APPEND", cmd)
	cmd.enc.SP().Mailbox(mailbox).SP()
	if options != nil && len(options.Flags) > 0 {
		cmd.enc.List(len(options.Flags), func(i int) {
			cmd.enc.Atom(string(options.Flags[i]))
		}).SP()
	}
	if options != nil && !options.Time.IsZero() {
		cmd.enc.String(options.Time.Format(dateTimeLayout)).SP()
	}
	cmd.wc = cmd.enc.Literal(size)
	return cmd
}

// AppendCommand is an APPEND command.
//
// Callers must write the message contents, then call Close.
type AppendCommand struct {
	cmd
	enc  *commandEncoder
	wc   io.WriteCloser
	data AppendData
}

func (cmd *AppendCommand) Write(b []byte) (int, error) {
	return cmd.wc.Write(b)
}

func (cmd *AppendCommand) Close() error {
	err := cmd.wc.Close()
	if cmd.enc != nil {
		cmd.enc.end()
		cmd.enc = nil
	}
	return err
}

func (cmd *AppendCommand) Wait() (*AppendData, error) {
	return &cmd.data, cmd.cmd.Wait()
}

// AppendData is the data returned by an APPEND command.
type AppendData struct {
	UID, UIDValidity uint32 // requires UIDPLUS or IMAP4rev2
}
