package responses

import (
	imap "github.com/emersion/imap/common"
)

func getRespName(res *imap.Resp) (name string) {
	if len(res.Fields) == 0 {
		return
	}

	name, _ = res.Fields[0].(string)
	return
}
