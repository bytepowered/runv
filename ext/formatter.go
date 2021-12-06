package ext

import (
	"bytes"
	"github.com/sirupsen/logrus"
	"strconv"
	"time"
)

var _ logrus.Formatter = new(UTCZoneFormatter)

type UTCZoneFormatter struct {
	name string
	zone time.Duration
	logrus.Formatter
}

func (f *UTCZoneFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	entry.Time = entry.Time.In(time.FixedZone(f.name, int((f.zone * time.Hour).Seconds())))
	return f.Formatter.Format(entry)
}

func NewUTCZoneFormatter(origin logrus.Formatter, name string, zone int) logrus.Formatter {
	return &UTCZoneFormatter{Formatter: origin, name: name, zone: time.Duration(zone)}
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
