package runv

import (
	"fmt"
	"os"
	"runtime"
	"syscall"
)

var (
	_CrashErrFileHandler *os.File
)

func InitCrashLogFile(errfile string) error {
	file, err := os.OpenFile(errfile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println(err)
		return err
	}
	_CrashErrFileHandler = file
	if err = syscall.Dup2(int(file.Fd()), int(os.Stderr.Fd())); err != nil {
		fmt.Println(err)
		return err
	}
	// 内存回收前关闭文件描述符
	runtime.SetFinalizer(_CrashErrFileHandler, func(fd *os.File) {
		_ = fd.Close()
	})
	return nil
}
