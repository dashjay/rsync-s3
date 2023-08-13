package main

import (
	"bytes"
	"flag"
	"net/http"
	_ "net/http/pprof"
	"sort"
	"time"

	"github.com/schollz/progressbar/v3"
	"github.com/sirupsen/logrus"

	"github.com/dashjay/rsync-s3/pkg/config"
	"github.com/dashjay/rsync-s3/pkg/core"
)

var (
	flagConfigFile = flag.String("config-file", "", "config for rsync oss")

	flagS3Accesskey = flag.String("s3.accesskey", "", "accesskey of aws s3")
	flagS3Secretkey = flag.String("s3.secretkey", "", "secretkey of aws s3")
	flagS3Endpoint  = flag.String("s3.endpoint", "http://localhost:9000", "endpoint of s3")
	flagS3Bucket    = flag.String("s3.bucket", "test-bucket", "bucket of s3")
	flagS3Prefix    = flag.String("s3.prefix", "ubuntu", "prefix")

	flagRsyncEndpoint = flag.String("rsync.endpoint", "rsync://rsync.mirrors.ustc.edu.cn/ubuntu", "rsync endpoint")
	flagLogLevel      = flag.String("log-level", "info", "log level")
	flagPProfPort     = flag.String("pprof-port", ":6161", "port for pprof")
)

func main() {
	flag.Parse()
	var cfg *config.Config
	if *flagConfigFile != "" {
		cfg = config.FromFile(*flagConfigFile)
	} else {
		cfg = new(config.Config)
		cfg.S3Accesskey = *flagS3Accesskey
		cfg.S3Secretkey = *flagS3Secretkey
		cfg.S3Endpoint = *flagS3Endpoint
		cfg.S3Bucket = *flagS3Bucket
		cfg.S3Prefix = *flagS3Prefix
		cfg.RsyncEndpoint = *flagRsyncEndpoint
		cfg.LogLevel = *flagLogLevel
		cfg.PProfPort = *flagPProfPort
	}

	lvl, err := logrus.ParseLevel(cfg.LogLevel)
	if err == nil {
		logrus.SetLevel(lvl)
		logrus.Infoln("running rsync-oss in level ", logrus.GetLevel())
	}

	if cfg.PProfPort != "" {
		go http.ListenAndServe(cfg.PProfPort, nil)
	}

	cli, err := core.NewRsyncClient(cfg)
	if err != nil {
		panic(err)
	}
	defer cli.Shutdown()
	fileList, err := cli.ListFiles()
	if err != nil {
		panic(err)
	}
	// all file from rsync will get relative path after module/path
	logrus.WithField("len(fileList)", len(fileList)).Infoln("rsync list files")
	logrus.WithField("first_key", string(fileList[0].Path)).Infoln("rsync first entry")

	err = cli.ReadIOError()
	if err != nil {
		panic(err)
	}

	s3Cli := core.NewS3Client(cfg)
	s3FileList, err := s3Cli.ListObjects()
	if err != nil {
		panic(err)
	}
	// all file from aws get absolute full path from the prefix
	// so we need to trim the left module(rsync) and add prefix before the list
	logrus.WithField("len(s3FileList)", len(s3FileList)).WithField("moduleName", cli.ModuleName()).Infoln("s3 list files")
	for i := range s3FileList {
		s3FileList[i].Path = bytes.TrimPrefix(s3FileList[i].Path, []byte(cfg.S3Prefix))
	}
	sort.Sort(s3FileList)
	if len(s3FileList) != 0 {
		logrus.WithField("first_key", string(s3FileList[0].Path)).Infoln("s3 list first entry")
	}
	newItems, oldItems := s3FileList.Diff(fileList)
	logrus.WithField("len(newItems)", len(newItems)).Infoln("files need to be update")
	logrus.WithField("len(oldItems)", len(oldItems)).Infoln("files need to be deleted")

	go func() {
		var pb *progressbar.ProgressBar
		if logrus.GetLevel() < logrus.DebugLevel {
			pb = progressbar.Default(int64(len(newItems)), "generate files")
		}
		logrus.WithField("len(newItems)", len(newItems)).Infoln("generator start(let rsync server know what we need)")

		err = cli.Generator(fileList, newItems, s3Cli, cfg.S3Bucket, pb)
		if err != nil {
			panic(err)
		}
		if pb != nil {
			pb.Reset()
		}
		logrus.Infoln("generator finished")
	}()

	time.Sleep(1 * time.Second)
	err = cli.FileDownloader(fileList[:], cfg.S3Bucket, s3Cli, len(newItems))
	if err != nil {
		panic(err)
	}
	//err = cli.HandleSymlinks(fileList, newItems, s3Cli, cfg.S3Bucket, nil)
	//if err != nil {
	//	panic(err)
	//}
	//err = cli.FileCleaner(s3FileList[:], oldItems, cfg.S3Bucket, s3Cli, len(oldItems))
	//if err != nil {
	//	panic(err)
	//}
}
