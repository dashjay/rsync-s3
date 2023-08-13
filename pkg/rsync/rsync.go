package rsync

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"path/filepath"
	"sort"
	"strings"

	"github.com/schollz/progressbar/v3"
	"github.com/sirupsen/logrus"

	// nolint:staticcheck // md4 is used by rsync
	"golang.org/x/crypto/md4"
)

const (
	RsyncVersion = "27.0"
	RsyncOK      = "@RSYNCD: OK"
)

type Client struct {
	ver    string
	module string
	path   string
	conn   *Conn
	seed   int32
}

func parseModuleAndPath(rsyncEndpoint string) (host, module, path string) {
	rsyncEndpoint = strings.TrimPrefix(rsyncEndpoint, "rsync://")
	firstSlash := strings.Index(rsyncEndpoint, "/")
	if firstSlash == -1 {
		logrus.Panicf("invalid rsync endpoint: %s", rsyncEndpoint)
	}
	host = rsyncEndpoint[:firstSlash]
	if !strings.Contains(host, ":") {
		host += ":873"
	}
	moduleAndPath := rsyncEndpoint[firstSlash+1:]
	if strings.Contains(moduleAndPath, "/") {
		temp := strings.SplitN(moduleAndPath, "/", 2)
		module, path = temp[0], temp[1]
		if path == "" {
			path = "/"
		}
	} else {
		module = moduleAndPath
		path = "/"
	}
	return
}

type ClientConfig struct {
	RsyncEndpoint string
}

func NewClient(cfg *ClientConfig) (*Client, error) {
	host, module, path := parseModuleAndPath(cfg.RsyncEndpoint)
	conn, err := net.Dial("tcp", host)
	if err != nil {
		return nil, err
	}
	rc := &Client{
		ver:    RsyncVersion,
		conn:   &Conn{writer: conn, reader: conn},
		module: module,
		path:   path,
	}
	err = rc.doHandShake()
	if err != nil {
		return nil, err
	}
	err = rc.changeToDemuxReader()
	return rc, err
}

func readLine(r io.Reader) []byte {
	var out []byte
	var buf [1]byte
	for {
		_, err := r.Read(buf[:])
		if err != nil {
			return out
		}
		out = append(out, buf[0])
		if out[len(out)-1] == '\n' {
			return out
		}
	}
}

func (rc *Client) ModuleName() string {
	return rc.module
}

func (rc *Client) doHandShake() error {
	logrus.Infoln("do handshake")
	_, err := fmt.Fprintf(rc.conn, "@RSYNCD: %s\n", rc.ver)
	if err != nil {
		return err
	}
	version := readLine(rc.conn)
	logrus.WithField("server_version", version).WithField("client_version", rc.ver).Infoln("negotiate version")

	// send module name
	rc.conn.Write([]byte(rc.module))
	rc.conn.Write([]byte("\n"))

	// read motd
	for {
		line := readLine(rc.conn)
		if bytes.HasPrefix(line, []byte(RsyncOK)) {
			break
		}
		// motd
		fmt.Print(string(line))
	}

	args := [][]byte{
		[]byte("--server"),
		[]byte("--sender"),
		[]byte("-ltpr"),
		[]byte("."),
		[]byte(fmt.Sprintf("%s/%s", rc.module, rc.path)),
	}
	for i := range args {
		_, err = rc.conn.Write(args[i])
		if err != nil {
			return err
		}
		_, err = rc.conn.Write([]byte("\n"))
		if err != nil {
			return err
		}
	}
	_, err = rc.conn.Write([]byte("\n"))
	if err != nil {
		return err
	}
	var seed int32
	err = binary.Read(rc.conn, binary.LittleEndian, &seed)
	if err != nil {
		return err
	}
	rc.seed = seed
	logrus.WithField("seed", seed).Infoln("read seed")
	logrus.Infoln("handshake completed")
	return nil
}

func (rc *Client) changeToDemuxReader() error {
	rc.conn.reader = NewMuxReader(rc.conn.reader)
	return rc.conn.WriteInt(EMPTY_EXCLUSION)
}

func (rc *Client) ListFiles() (InnerFileList, error) {
	logrus.Infoln("list object from rsync server")
	fileList := make(InnerFileList, 0)
	var buf = make([]byte, 1)
	first := true
	for {
		_, err := io.ReadFull(rc.conn, buf)
		if err != nil {
			return nil, err
		}
		flag := buf[0]
		if flag == 0 {
			break
		}
		var prev *InnerFileInfo
		if first {
			prev = nil
		} else {
			prev = &fileList[len(fileList)-1]
		}
		fi, err := rc.readFileInfo(int(flag), prev)
		if err != nil {
			return nil, err
		}
		first = false
		fileList = append(fileList, fi)
		fmt.Printf("\r%d files listed", len(fileList))
	}
	sort.Sort(fileList)
	return fileList, nil
}

func (rc *Client) readFileInfo(flag int, prev *InnerFileInfo) (fi InnerFileInfo, err error) {
	var buf = make([]byte, 1)
	var partial, pathLen uint32 = 0, 0
	if (flag & FlistNameSame) != 0 {
		_, err = io.ReadFull(rc.conn, buf)
		if err != nil {
			return
		}
		partial = uint32(buf[0])
	}
	if (flag & FlistNameLong) != 0 {
		err = binary.Read(rc.conn, binary.LittleEndian, &pathLen)
		if err != nil {
			return
		}
	} else {
		_, err = io.ReadFull(rc.conn, buf)
		if err != nil {
			return
		}
		pathLen = uint32(buf[0])
	}
	buf = make([]byte, pathLen)
	_, err = io.ReadFull(rc.conn, buf)
	if err != nil {
		return
	}
	path := make([]byte, 0, partial+pathLen)
	if (flag & FlistNameSame) != 0 {
		path = append(path, prev.Path[0:partial]...)
	}
	path = append(path, buf...)

	// try read size
	size, err := rc.conn.ReadVarInt()
	if err != nil {
		return
	}
	var mtime int32
	if (flag & FlistTimeSame) == 0 {
		mtime, err = rc.conn.ReadInt()
		if err != nil {
			return
		}
	} else {
		mtime = prev.Mtime
	}
	var mode FileMode
	if (flag & FlistModeSame) == 0 {
		var val int32
		val, err = rc.conn.ReadInt()
		if err != nil {
			return
		}
		mode = FileMode(val)
	} else {
		mode = prev.Mode
	}
	var slink []byte
	if (mode&32768) != 0 && mode&8192 != 0 {
		var sLen int32
		sLen, err = rc.conn.ReadInt()
		if err != nil {
			return
		}
		slink = make([]byte, sLen)
		_, err = io.ReadFull(rc.conn, slink)
		if err != nil {
			err = fmt.Errorf("failed to read symlink, err: %s", err)
			return
		}
	}
	return InnerFileInfo{
		Path:       path,
		Size:       size,
		Mtime:      mtime,
		Mode:       mode,
		TargetLink: slink,
	}, nil
}

func (rc *Client) ReadIOError() error {
	ioErr, err := rc.conn.ReadInt()
	if err != nil {
		return err
	}
	if ioErr != 0 {
		return fmt.Errorf("return io error, NO: %d", ioErr)
	}
	return nil
}

func (rc *Client) Shutdown() error {
	return rc.conn.Close()
}

func (rc *Client) Generator(remoteList InnerFileList, downloadList []int, pb *progressbar.ProgressBar) error {
	for _, v := range downloadList {
		if pb != nil {
			pb.Add(1)
		}
		if remoteList[v].Mode.IsREG() {
			logrus.WithField("path", string(remoteList[v].Path)).WithField("idx", v).
				Debugln("tell rsync server to prepare for this file")
			if err := rc.conn.WriteInt(int32(v)); err != nil {
				return fmt.Errorf("failed to send index: %s", err)
			}
			var sh SumHead
			err := WriteSumHead(rc.conn, sh)
			if err != nil {
				return err
			}
		} else {
			logrus.WithField("path", string(remoteList[v].Path)).Debugln("skip none regular file")
		}
	}

	if err := rc.conn.WriteInt(IndexEnd); err != nil {
		return fmt.Errorf("write index end error: %s", err)
	}
	return nil
}

type SymlinkItem struct {
	Source string
	Target string
}

func (rc *Client) HandleSymlinks(remoteList InnerFileList, downloadList []int, bucket string, pb *progressbar.ProgressBar) []SymlinkItem {
	var out = make([]SymlinkItem, 0)
	cnt := 0
	for _, idx := range downloadList {
		if !remoteList[idx].Mode.IsLNK() {
			continue
		}
		cleanSource := filepath.Clean(filepath.Join(bucket, rc.module, rc.path, string(remoteList[idx].TargetLink)))
		key := filepath.Join(rc.module, rc.path, string(remoteList[idx].Path))

		out = append(out, SymlinkItem{
			Source: cleanSource,
			Target: key,
		})
		cnt += 1
	}
	return out
}

type DownloadEntry struct {
	io.ReadCloser
	error
}

func (rc *Client) FileDownloadList(ctx context.Context, localList InnerFileList, bucket string, total int) chan *DownloadEntry {
	cnt := 0
	lmd4 := md4.New()
	c := make(chan *DownloadEntry)
	go func() {
		for {
			select {
			case <-ctx.Done():
				c <- nil
				close(c)
				return
			default:
				index, err := rc.conn.ReadInt()
				if err != nil {
					c <- &DownloadEntry{
						ReadCloser: nil,
						error:      err,
					}
					close(c)
					return
				}
				if index == IndexEnd {
					break
				}
				cnt++

				sh, err := ReadSumHead(rc.conn)
				if err != nil {
					c <- &DownloadEntry{
						ReadCloser: nil,
						error:      err,
					}
					close(c)
					return
				}
				path := localList[index].Path
				strPath := string(path)
				if logrus.GetLevel() >= logrus.DebugLevel {
					logrus.WithField("path", strPath).
						WithField("sum_head", sh.String()).
						WithField("index", index).
						WithField("size", localList[index].Size).Debugln("try to download file")
				}

				downloadSize := 0
				if err := binary.Write(lmd4, binary.LittleEndian, rc.seed); err != nil {
					c <- &DownloadEntry{
						ReadCloser: nil,
						error:      err,
					}
					close(c)
					return
				}
				pr, pw := io.Pipe()

				c <- &DownloadEntry{
					ReadCloser: pr,
					error:      nil,
				}

				var filePb *progressbar.ProgressBar
				if logrus.GetLevel() < logrus.DebugLevel {
					filePb = progressbar.DefaultBytes(localList[index].Size)
					filePb.Describe(fmt.Sprintf("[%d/%d]: %s", cnt, total, strPath))
				}
				for {
					token, err := rc.conn.ReadInt()
					if err != nil {
						pw.CloseWithError(err)
						close(c)
						return
					}
					logrus.WithField("token", token).Debugln("downloading file part")
					if token == 0 {
						err = pw.Close()
						if err != nil {
							logrus.WithError(err).Errorln("close pipe error")
						}
						break
					} else if token < 0 {
						pw.CloseWithError(errors.New("not support block checksum"))
						close(c)
						return
					} else {
						_, err := io.CopyN(io.MultiWriter(pw, lmd4), rc.conn, int64(token))
						if err != nil {
							pw.CloseWithError(err)
							close(c)
							return
						}
						if filePb != nil {
							filePb.Add(int(token))
						}
						downloadSize += int(token)
						logrus.WithField("path", string(localList[index].Path)).
							WithField("download_size", downloadSize).Debugln("downloading")
					}
				}

				localMd4Result := lmd4.Sum(nil)
				rmd4 := make([]byte, len(localMd4Result))
				_, err = io.ReadFull(rc.conn, rmd4)
				if err != nil {
					pw.CloseWithError(err)
					close(c)
					return
				}
				if logrus.GetLevel() >= logrus.DebugLevel {
					logrus.WithField("rmd4", hex.EncodeToString(rmd4)).
						WithField("lmd4", hex.EncodeToString(localMd4Result)).
						WithField("path", string(localList[index].Path)).Debugln("compare md4")
				}

				lmd4.Reset()
				if !bytes.Equal(rmd4, localMd4Result) {
					logrus.WithField("path", string(localList[index].Path)).Warnln("md4 mismatched")
				}
			}
		}
	}()
	return c
}
