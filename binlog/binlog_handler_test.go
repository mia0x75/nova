package binlog

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/toolkits/file"
)

// GetCurrentPath get current path
func GetCurrentPath() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Errorf("[E] %s", err.Error())
		return ""
	}
	return strings.Replace(dir, "\\", "/", -1)
}

// CurrentPath current path
var CurrentPath = GetCurrentPath()

func init() {
	log.SetFormatter(&log.TextFormatter{
		TimestampFormat:  "2006-01-02 15:04:05",
		ForceColors:      true,
		QuoteEmptyFields: true,
		FullTimestamp:    true,
	})
	log.SetLevel(log.Level(5))
}

// test SaveBinlogPostionCache api
// 基础的保存pos到cache和读取cache相关api测试
func TestBinlogHandler_SaveBinlogPostionCache(t *testing.T) {
	h := &Binlog{
		statusLock: new(sync.Mutex),
		status:     cacheHandlerIsOpened,
	}
	var err error
	flag := os.O_RDWR | os.O_CREATE | os.O_SYNC
	h.cacheHandler, err = os.OpenFile(CurrentPath+"/cache_test.pos", flag, 0755)
	if err != nil {
		t.Errorf("binlog open cache file error: %+v", err)
	}
	defer file.Remove(CurrentPath + "/cache_test.pos")

	binfile := "mysql-bin.000059"
	pos := int64(123456)
	eventIndex := int64(20)

	r := packPos(binfile, pos, eventIndex)
	h.saveBinlogPositionCache(r)
	s, p, e := h.getBinlogPositionCache()
	if s != binfile {
		t.Errorf("getBinlogPositionCache binfile error")
	}
	if pos != p {
		t.Errorf("getBinlogPositionCache pos error")
	}
	if e != eventIndex {
		t.Errorf("getBinlogPositionCache eventIndex error")
	}

	binfile = "mysql-bin.00005"
	pos = int64(12345)
	eventIndex = int64(2)
	r = packPos(binfile, pos, eventIndex)
	h.saveBinlogPositionCache(r)
	s, p, e = h.getBinlogPositionCache()
	if s != binfile {
		t.Errorf("getBinlogPositionCache binfile error")
	}
	if pos != p {
		t.Errorf("getBinlogPositionCache pos error")
	}
	if e != eventIndex {
		t.Errorf("getBinlogPositionCache eventIndex error")
	}
}
