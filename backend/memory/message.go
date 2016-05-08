package memory

import (
	"github.com/emersion/imap/common"
)

type Message struct {
	uid uint32
	metadata *common.Message
}
