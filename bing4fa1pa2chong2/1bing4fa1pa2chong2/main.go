package main

//管道作为一个配角，放在goroutine两头，来限制for循环(包括range遍历)中实际开启协程的个数
import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"sync"
	"time"
)

func myerr(where string, err error) {
	if err != nil {
		fmt.Println("%v报错==", where, err)
		os.Exit(1)
	}
}

//获取网页上所有图片的链接组成的切片
func pictureUrls(url string) []string {
	//向待爬取图片的网站发送请求
	response, err := http.Get(url)
	myerr("15行response, err := http.Get", err)
	defer response.Body.Close() //下面读取响应体后需调用者关闭，见官方文档response下的Body
	//读取响应体
	bodyBytes, err := ioutil.ReadAll(response.Body)
	myerr("19行bodySlice, err := ioutil.ReadAll", err)
	//与正则表达式匹配并获得双重切片结果
	regu := regexp.MustCompile(`<img[\s\S]+?src="(http[\s\S]+?)"`)
	resBytes := regu.FindAllStringSubmatch(string(bodyBytes), -1)
	//遍历双重切片，取出图片链接组成新切片
	var urls []string
	for _, v := range resBytes {
		urls = append(urls, v[1])
	}
	return urls
}

//下载图片，向所有图片链接发送请求，获得响应体并读取写进文件中
func downLoad(k int, url string) {
	response, err := http.Get(url)
	//myerr("37行",err)
	if err != nil {
		return
	}
	defer response.Body.Close() //下面读取响应体后需关闭
	//读取响应体
	byteSlice, err := ioutil.ReadAll(response.Body)
	myerr("41行", err)
	//将图片的字节切片写进文件中,需给图片命不同的路径名字
	err = ioutil.WriteFile("./picture/"+strconv.Itoa(k)+".png", byteSlice, 0644)
	myerr("45行", err)
}

//并发下载图片
var wg sync.WaitGroup              //全部变量的等待组，用于协程函数里的计数，主函数里等待
var chanStr = make(chan string, 5) //全局变量管道通过阻塞来控制全部协程一瞬间最多5个
func asyncDownload(k int, url string) {
	(&wg).Add(1) //每一个协程开启之前计数增加  这里也可以不用&符号
	go func() {
		chanStr <- "one" //全局变量管道通过阻塞来控制全部协程的一瞬间存在的最多数量,5个
		downLoad(k, url)
		<-chanStr
	}()
	wg.Done() //每个协程结束后计数减少//视频中等待组的放置位置不对，会导致效率比不开协程没增加多少
}
func main() {
	strSlice := pictureUrls("https://www.guancha.cn/")
	t1 := time.Now().UnixNano()
	//for k,v := range strSlice {
	//	downLoad(k,v)
	//}
	//下面用协程，效率提高1000多倍
	for k, v := range strSlice {
		asyncDownload(k, v) //遍历图片链接切片下载，每个下载操作都开协程过于占用服务器资源
		//且会被网站发现，因此通过管道容量来限制
	}
	wg.Wait() //把等待从上面的遍历里移到这里，运行效率会比在循环里高1.5倍
	t2 := time.Now().UnixNano()
	fmt.Println("耗时=", t2-t1)
}
