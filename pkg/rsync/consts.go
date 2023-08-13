package rsync

const (
	EMPTY_EXCLUSION = int32(0)

	MuxBase      = 7
	MsgData      = 0
	MsgErrorXfer = 1
	MsgInfo      = 2
	MsgError     = 3
	MsgWarning   = 4
	MsgIoError   = 22
	MsgNoop      = 42
	MsgSuccess   = 100
	MsgDeleted   = 101
	MsgNoSend    = 102

	// For FILE LIST(1 byte)
	FlistEnd      = 0x00
	FlistTopLevel = 0x01 /* needed for remote --delete */
	FlistModeSame = 0x02 /* mode is repeat */
	FlistRdevSame = 0x04 /* rdev is repeat */
	FlistUIDSame  = 0x08 /* uid is repeat */
	FlistGIDSame  = 0x10 /* gid is repeat */
	FlistNameSame = 0x20 /* name is repeat */
	FlistNameLong = 0x40 /* name >255 bytes */
	FlistTimeSame = 0x80 /* time is repeat */

	IndexEnd = int32(-1)

	// File type
	SIfmt   = 0170000 /* Type of file */
	SIfreg  = 0100000 /* Regular file.  */
	SIfdir  = 0040000 /* Directory.  */
	SIflnk  = 0120000 /* Symbolic link.  */
	SIfchr  = 0020000 /* Character device.  */
	SIfblk  = 0060000 /* Block device.  */
	SIfifo  = 0010000 /* FIFO.  */
	SIfsock = 0140000 /* Socket.  */
)
