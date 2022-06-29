p2p下载——分片下载服务器Gin实现

p2p下载有很多种，简单的一种就是文件下好后，整个文件提供给别人下载，这种做法需要等文件下载完成后才可作为源。
之前看过滴滴内部分享的文件传输系统就是这样，一台机器下载好作为源给其他几台机器下载，然后2台机器再给其他机器下载，指数级扩散。
但是该方法基本要等待几个完整的下载周期，一台机器下载需要10秒，1024台机器就需要10个下载周期才能全部下好，也就是100秒

而使用分片维度去共享给其他机器，每个机器下载不同的分片后就可以立刻给后来的机器下载，1000台机器下载时整体机器就都处于一起下载文件的情况了。
那么最后几乎接近10秒就可以完成1000台机器的下载。

文件的分片下载主要是使用http请求中header指定range范围获取文件的指定内容。
使用nginx托管目录来做的服务器就支持下载请求指定文件的范围。
我们现在的服务就是阿里开源的[Dragonfly][1]
其中的supernode及管理客户端下载分片且作为源站的超级节点，是将下载后的文件目录交给nginx托管，然后作为源种子提供给客户端作为初始分片去下载所需要的部分。
使用中nginx跟我们的服务是分离的，上线前需要在机器上配置启动nginx，还有些其他问题，因此考虑自身的服务实现一个文件下载服务器，减少额外依赖。

整体功能也比较简单主要就是，可以看[HTTP请求范围][2]
1. 返回头中需要标记Accept-Range: bytes表明该服务器支持指定请求范围
2. 支持解析请求中的Range:bytes=0-100来指定获取的文件范围
3. 支持返回码206 416等分片下载中需要返回码

实现点也就2个
头解析
获取请求头是否有Range，有的话解析获取范围，从而去读取文件的该部分内容
```
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
```
获取后简单判断下，如果请求范围起始值超过文件结束值，返回个416表示超过范围了，否则返回206表示为文件的分片同时范围该部分内容
```
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
```

第二点就是如何返回文件的部分内容了
构造一个RangeFile结构体，该实现一个Read接口，在使用io.Copy()范围内容给请求时能够读取指定范围的内容

```
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

```

最后就是直接返回结果，还要记得返回对方该读取的参数
```
c.Status(dataCode)
c.Header(HeaderContentLength, fmt.Sprintf("%d", fileSize))
c.Header(HeaderAcceptRange, HeaderAcceptRangeValue)
n, err := io.Copy(c.Writer, data)
if err != nil {
    xxxxx
}
```

详细代码见https://github.com/yarenhere/daily_learn/blob/master/file-server/main.go

```
// 在启动目录先生成文件
mkfile -n 101m test.file
// 启动服务
go run main.go ./
另起一个shell
// 请求0-10的 返回码206及部分内容
wget -O /dev/null  http://127.0.0.1:13000/data/test.file --head "Range:bytes=0-10"
// 请求超出文件大小，返回码416
wget -O /dev/null  http://127.0.0.1:13000/data/test.file --head "Range:bytes=10000000000-100000009000"
// 直接返回全部文件，返回码200
wget -O /dev/null  http://127.0.0.1:13000/data/test.file
```

[1]: https://github.com/dragonflyoss/Dragonfly
[2]: https://developer.mozilla.org/zh-CN/docs/Web/HTTP/Range_requests
