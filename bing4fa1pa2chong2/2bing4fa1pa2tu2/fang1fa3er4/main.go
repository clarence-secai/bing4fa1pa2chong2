package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"sync"
)

//以下是参照视频老师的方法，并发以循环(包括range遍历)goroutine为主，管道、等待组为辅
func myerr(where string, err error) {
	if err != nil {
		log.Fatal(where, err)
	}
}

var chanSrc = make(chan string, 100)
var chanStrict = make(chan int, 3)
var vs string
var wg1, wg2 sync.WaitGroup

func main() {
	url := "http://23hanfu.com/category/hanfutupian"
	for i := 1; i <= 3; i++ {
		if i > 1 {
			url = url + "/page/" + strconv.Itoa(i)
		}
		//3个循环开启三个协程，都是获取图片链接的,放入管道chanSrc中
		wg1.Add(1)
		fmt.Printf("获取分页%v的图片\n", i)
		go func() {
			response, err := http.Get(url)
			myerr("错误1", err)
			defer response.Body.Close()
			bodyByte, err := ioutil.ReadAll(response.Body)
			myerr("错误2", err)
			reg := regexp.MustCompile(`<img.+?(data-original|src)="(http.+?)"`)
			//其实上面这一行可以抽出协程外面先执行，避免每个协程都重复执行一回正则的解析
			doubleBytes := reg.FindAllStringSubmatch(string(bodyByte), -1)
			for _, v := range doubleBytes {
				chanSrc <- v[2]
			}
			wg1.Done()
		}()
	}
	//todo:对等待组开协程，让关闭管道这个工作不再是卡在这里等，从而可以继续执行后面的代码，提高效率
	go func() {
		wg1.Wait()//todo:将wait放到一个协程中，妙哉！！！
		close(chanSrc) //等待着关闭管道，便于后面可以无所顾忌的遍历管道
		fmt.Println("chanSrc管道成功关闭")
	}()
	i := 0
	//todo:被取值的管道有等待协程吸入的特性
	for vs = range chanSrc { //注意管道的遍历左边没有键值对的键
		wg2.Add(1)
		i++
		chanStrict <- 1 //chanStrict管道容量为3，通过阻塞控制下载的协程数量为3
		go func() {
			//chanStrict <- 1 //不该写在这里，不然协程开了很多，只是阻塞着保证3个运行
			response, err := http.Get(vs)
			myerr("48行错误", err)
			defer response.Body.Close()
			bodyByte, err := ioutil.ReadAll(response.Body)
			myerr("51行报错", err)
			ioutil.WriteFile("./store/"+strconv.Itoa(i)+".jpg", bodyByte, 0644)
			fmt.Printf("图片成功存入文件夹%v\n", i)
			<-chanStrict
			wg2.Done()
		}()
	}
	wg2.Wait()
}
