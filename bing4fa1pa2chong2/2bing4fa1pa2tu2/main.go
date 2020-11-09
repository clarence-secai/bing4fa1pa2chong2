package main

//并发爬取分页的图片，这是自己原创的，可以成功运行，但显然效率不如fang1fa3er4中
//视频老师的快，也不像goroutine的正宗用法
import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"
)

//设置访问超时，处理那种在爬取过程中某一个图片卡着下不下来的情况，本例中主体函数并未
//用到这个init，如果要用，则http.Get()要改成myClient.Get()
var myClient http.Client

func init() {
	myClient = http.Client{
		Transport: &http.Transport{
			Dial: func(network string, addr string) (net.Conn, error) {
				//设置发起连接的超时时间
				conn, err := net.DialTimeout(network, addr, time.Second*2)
				if err != nil {
					return nil, err
				}
				//设置成功连接后操作的超时时间
				deadline := time.Now().Add(2 * time.Second)
				conn.SetDeadline(deadline)
				//返回设置好超时属性的conn
				return conn, err
			},
		},
	}
}
func myerr(where string, err error) {
	if err != nil {
		log.Fatal(where, "报错=", err)
	}
}

//获取各个页面的图片链接
var chanBool = make(chan bool, 3)
var chanUrl = make(chan string, 5)
var chanSrc = make(chan string, 5)
var chanBody = make(chan string, 5)

//获得每个分页的网址
func GetUrls(url string, max int) { //max表示要下载的最大分页页码
	for i := 1; i <= max; i++ {
		if i > 1 {
			url = url + "/page/" + strconv.Itoa(i)
		}
		chanUrl <- url
		fmt.Println("向管道写入了分页网址")
	}
	close(chanUrl) //一定要close，否则后面从chanUrl弹出取数，会因本函数是协程写入，取完后还傻傻的一直等
	//wg.Done()
	fmt.Println("第一个协程结束Done")
	//close(chanUrl)
	//return chanUrl
}

//获得每个分页网址的响应体
func GetPagesBody() {
	for {
		url, ok := <-chanUrl //协程写入chanUrl又不close，则此处不会出现ok==false，而是一直傻傻等待
		if !ok {
			fmt.Println("41行取出chanUrl里的分页网址取尽")
			//wg.Done()
			close(chanBody)
			break
		}
		//借着for循环，接下来的获取各分页也可以开多个协程，通过辅助管道控制协程数量
		response, err := http.Get(url)
		myerr("31行", err)
		BodyBytes, err := ioutil.ReadAll(response.Body)
		myerr("33行", err)
		chanBody <- string(BodyBytes)
		fmt.Println("向管道写入了分页响应体内容")
	}
}

//获得每个分页网址响应体里图片的链接
func GetPictureSrc() {
	for {
		res, ok := <-chanBody
		if !ok {
			fmt.Println("41行取出各分页响应体取尽")
			//wg.Done()
			close(chanSrc)
			break
		}
		//借for循环，接下来的获取分页里图片链接也可开多个协程，通过辅助管道控制协程数量
		reg := regexp.MustCompile(`<img.+?src="(http.+?)"`)
		fmt.Println("完成正则匹配")
		doubleBytes := reg.FindAllStringSubmatch(res, -1)
		for _, v := range doubleBytes {
			chanSrc <- v[1]
			fmt.Println("向管道写入了图片链接")
			fmt.Println(v[1])
		}
	}
}

//根据图片链接下载图片
func DownLoadPicture() {
	k := 0
	for {
		url, ok := <-chanSrc
		k++
		if !ok {
			//wg.Done()
			fmt.Println("从管道取出图片链接取尽")
			break
		}
		//借着for循环，下面的下载图片的操作也可开多个协程，用辅助管道控制协程数量
		response, err := http.Get(url)
		//myerr("16行",err)
		if err != nil {
			continue //替代掉myerr的遇到错就panic，从而跳过某个下不下来的情况，避免一颗老鼠屎坏了一锅粥
		}
		BodyBytes, err := ioutil.ReadAll(response.Body)
		myerr("19行", err)
		err = ioutil.WriteFile("./tupian/"+strconv.Itoa(k)+".jpeg", BodyBytes, 0644)
		myerr("84行", err)
		fmt.Println("下载成功", k)
	}
}

var wg sync.WaitGroup

func main() {
	fmt.Println("开始")
	//wg.Add(4)
	go GetUrls("http://23hanfu.com/category/hanfutupian", 3)
	go GetPagesBody()
	go GetPictureSrc()
	//go DownLoadPicture()
	DownLoadPicture() //可以用这一行，替代上一行的go DownLoadPicture()和同时起作用的wg等待组的wg.Add(),wg.Done(),wg.Wait()
	//wg.Wait()
	fmt.Println("运行结束")
}
