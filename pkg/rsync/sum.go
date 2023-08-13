package rsync

import (
	"encoding/json"
)

type SumHead struct {
	ChecksumCount int32 `json:"checksum_count"`
	BlockLen      int32 `json:"block_len"`
	ChecksumLen   int32 `json:"checksum_len"`
	ReminderLen   int32 `json:"reminder_len"`
}

func ReadSumHead(r *Conn) (res SumHead, err error) {
	res.ChecksumCount, err = r.ReadInt()
	if err != nil {
		return
	}
	res.BlockLen, err = r.ReadInt()
	if err != nil {
		return
	}
	res.ChecksumLen, err = r.ReadInt()
	if err != nil {
		return
	}
	res.ReminderLen, err = r.ReadInt()
	if err != nil {
		return
	}
	return
}

func (s SumHead) String() string {
	bin, _ := json.Marshal(s)
	return string(bin)
}

func WriteSumHead(r *Conn, sh SumHead) error {
	err := r.WriteInt(sh.ChecksumCount)
	if err != nil {
		return err
	}
	err = r.WriteInt(sh.BlockLen)
	if err != nil {
		return err
	}
	err = r.WriteInt(sh.ChecksumLen)
	if err != nil {
		return err
	}
	err = r.WriteInt(sh.ReminderLen)
	if err != nil {
		return err
	}
	return nil
}
