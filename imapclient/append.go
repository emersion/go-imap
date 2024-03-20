package imapclient

import (
	"io"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal"
)

// Append sends an APPEND command.
//
// The caller must call AppendCommand.Close.
//
// The options are optional.
func (c *Client) Append(mailbox string, size int64, options *imap.AppendOptions) *AppendCommand {
	cmd := &AppendCommand{}
	cmd.enc = c.beginCommand("APPEND", cmd)
	cmd.enc.SP().Mailbox(mailbox).SP()
	if options != nil && len(options.Flags) > 0 {
		cmd.enc.List(len(options.Flags), func(i int) {
			cmd.enc.Flag(options.Flags[i])
		}).SP()
	}
	if options != nil && !options.Time.IsZero() {
		cmd.enc.String(options.Time.Format(internal.DateTimeLayout)).SP()
	}
	// TODO: literal8 for BINARY
	// TODO: UTF8 data ext for UTF8=ACCEPT, with literal8
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
	data imap.AppendData
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

func (cmd *AppendCommand) Wait() (*imap.AppendData, error) {
	return &cmd.data, cmd.cmd.Wait()
}

// MultiAppend sends an APPEND command with multiple messages.
//
// The caller must call MultiAppendCommand.Close.
//
// This command requires support for the MULTIAPPEND extension.
func (c *Client) MultiAppend(mailbox string) *MultiAppendCommand {
	cmd := &MultiAppendCommand{}
	cmd.enc = c.beginCommand("APPEND", cmd)
	cmd.enc.SP().Mailbox(mailbox)
	return cmd
}

// MultiAppendCommand is an APPEND command with multiple messages.
type MultiAppendCommand struct {
	cmd
	enc *commandEncoder
	wc  io.WriteCloser
}

// CreateMessage appends a new message.
func (cmd *MultiAppendCommand) CreateMessage(size int64, options *imap.AppendOptions) io.Writer {
	if cmd.wc != nil {
		// TODO: handle error
		cmd.wc.Close()
		cmd.wc = nil
	}

	cmd.enc.SP()
	if options != nil && len(options.Flags) > 0 {
		cmd.enc.List(len(options.Flags), func(i int) {
			cmd.enc.Flag(options.Flags[i])
		}).SP()
	}
	if options != nil && !options.Time.IsZero() {
		cmd.enc.String(options.Time.Format(internal.DateTimeLayout)).SP()
	}
	cmd.wc = cmd.enc.Literal(size)
	return cmd.wc
}

// Close ends the APPEND command.
func (cmd *MultiAppendCommand) Close() error {
	err := cmd.wc.Close()
	if cmd.enc != nil {
		cmd.enc.end()
		cmd.enc = nil
	}
	return err
}

func (cmd *MultiAppendCommand) Wait() error {
	return cmd.cmd.Wait()
}
