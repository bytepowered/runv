package ext

import (
	"bytes"
	"github.com/sirupsen/logrus"
	"strconv"
	"time"
)

var _ logrus.Formatter = new(UTC8Formatter)

type UTC8Formatter struct {
	logrus.Formatter
}

func (f *UTC8Formatter) Format(entry *logrus.Entry) ([]byte, error) {
	entry.Time = entry.Time.In(time.FixedZone("UTC+8", int((8 * time.Hour).Seconds())))
	return f.Formatter.Format(entry)
}

func NewUTC8Formatter(origin logrus.Formatter) logrus.Formatter {
	return &UTC8Formatter{origin}
}

func LogShortCaller(caller string, line int) string {
	// github.com/bytepowered/webtrigger/impl/coding.(*WebsocketMessageAdapter).OnInit
	sbytes := []byte(caller)
	idx := bytes.LastIndexByte(sbytes, '(')
	if idx <= 0 {
		return caller
	}
	return string(sbytes[idx:]) + ":" + strconv.Itoa(line)
}
