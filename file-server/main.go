package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
)

const (
	HeaderContentLength    = "Content-Length"
	HeaderAcceptRange      = "Accept-Range"
	HeaderRange      = "Range"
	HeaderAcceptRangeValue = "bytes"
)

var RootFileDir string

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("need agrs RootFileDir, not %v", os.Args)
	}
	RootFileDir = os.Args[1]
	fileInfo, err := os.Stat(RootFileDir)
	if err != nil {
		log.Fatalf("stat RootFileDir failed:%v", err)
	}
	if !fileInfo.IsDir() {
		log.Fatalf("need RootFileDir not file:%s", RootFileDir)
	}

	router := gin.Default()
	router.HEAD("/data/*file_path", headFile)
	router.GET("/data/*file_path", getFile)
	if err := router.Run(":13000"); err != nil {
		log.Errorf("router start failed:%v", err)
		return
	}
}

func headFile(c *gin.Context) {
	filePath := c.Param("file_path")
	log.Infof("headFile %s", filePath)
	filePath = fmt.Sprintf("%s/%s", RootFileDir, c.Param("file_path"))
	fileInfo, err := os.Stat(filePath)
	if err != nil && !os.IsNotExist(err) {
		c.String(http.StatusInternalServerError, err.Error())
		return
	} else if os.IsNotExist(err) {
		c.String(http.StatusNotFound, "")
		return
	}
	if fileInfo.IsDir() {
		c.String(http.StatusNotFound, "")
		return
	}
	c.Header(HeaderContentLength, fmt.Sprintf("%d", fileInfo.Size()))
	c.Header(HeaderAcceptRange, HeaderAcceptRangeValue)
	c.Status(http.StatusOK)
}

func getFile(c *gin.Context) {
	filePath := c.Param("file_path")
	log.Infof("getFile %s", filePath)
	filePath = fmt.Sprintf("%s/%s", RootFileDir, c.Param("file_path"))
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		err, ok := r.(error)
		if !ok {
			c.JSON(http.StatusInternalServerError, r)
		}
		if os.IsNotExist(err) {
			c.String(http.StatusNotFound, err.Error())
			return
		}
	}()
	var fileSize int64
	var dataCode int
	fo, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer fo.Close()

	if fileInfo, err := fo.Stat(); err != nil {
		panic(err)
	} else {
		fileSize = fileInfo.Size()
	}

	bytesRange := c.GetHeader(HeaderRange)
	var data io.Reader
	if bytesRange == ""{
		data = fo
		dataCode = http.StatusOK
	}else{
		start, end, err := getRange(bytesRange)
		if err != nil {
			c.String(http.StatusBadRequest, err.Error())
			return
		}
		log.Infof("HeaderAcceptRange %d-%d", start, end)
		if start > end {
			c.Status(http.StatusBadRequest)
			return
		}
		if start > fileSize {
			c.Status(http.StatusRequestedRangeNotSatisfiable)
			return
		}
		fileSize = minInt64(fileSize-start, end-start+1)
		rf, err := NewRangeFile(fo, start, end)
		if err != nil{
			panic(err)
		}
		data = rf
		dataCode = http.StatusPartialContent
	}
	c.Status(dataCode)
	c.Header(HeaderContentLength, fmt.Sprintf("%d", fileSize))
	c.Header(HeaderAcceptRange, HeaderAcceptRangeValue)
	n, err := io.Copy(c.Writer, data)
	if err != nil {
		panic(err)
	}
	log.Infof("getFile send file %s:%d", filePath, n)

}

func getRange(bytesRange string) (start, end int64, err error) {
	ranges := strings.Split(strings.Trim(bytesRange, "bytes="), "-")
	if len(ranges) != 2 {
		err = fmt.Errorf("wrong %s:%s", HeaderRange, bytesRange)
		return
	}
	start, err = strconv.ParseInt(ranges[0], 10, 64)
	if err != nil {
		return
	}
	end, err = strconv.ParseInt(ranges[1], 10, 64)
	if err != nil {
		return
	}
	return
}

type RangeFile struct {
	fo    *os.File
	start int64
	end   int64
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func NewRangeFile(fo *os.File, start, end int64) (*RangeFile, error) {
	rf := &RangeFile{
		fo:    fo,
		start: start,
		end:   end,
	}
	return rf, nil
}

func (rf *RangeFile) Read(buf []byte) (int, error) {
	var dataSize int64
	dataSize = minInt64(rf.end-rf.start+1, int64(len(buf)))
	if dataSize <= 0 {
		return 0, io.EOF
	}
	n, err := rf.fo.ReadAt(buf[:dataSize], rf.start)
	rf.start += dataSize
	return n, err
}