package main

import (
	"fmt"
	"runtime"
	"context"
	"os/signal"
	"os"
	"syscall"
	"net/http"
	"strconv"
	"strings"
	"net/url"
	"io/ioutil"
	"time"
	"errors"

	"github.com/PuerkitoBio/goquery"
	iconv "github.com/djimenez/iconv-go"
)

const(
	REQUEST_HOST = "http://lisea.blog.51cto.com"
)

func incrPV(u string) error {
	URL, err := url.Parse(u)
	if nil != err {
		return err
	}

	paths := strings.Split(URL.Path[1:], "/")
	if 2 > len(paths) {
		return errors.New("request false url: " + u)
	}

	body := "uid="+paths[0]+"&tid="+paths[1]
	// start request
	c := &http.Client{}
	req, err := http.NewRequest(http.MethodPost, "http://lisea.blog.51cto.com/js/header.php", strings.NewReader(body))
	if nil != err {
		return err
	}
	//set request header
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := c.Do(req)
	if nil != err {
		return err
	}
	defer res.Body.Close()
	_, err = ioutil.ReadAll(res.Body)

	return err
}

func requestPV(ctx context.Context, title, url string){
	var (
		request_success int64
		request_failure int64
	)

	for {
		select {
		case <- ctx.Done():
			fmt.Printf("request url: %s\ttitle: %s\tsuccess count: %d\tfailure count: %d\n", url, title, request_success, request_failure)
			break
		default:
		}

		// request page
		err := incrPV(url)

		// Statistics for success and failure
		if nil != err {
			request_failure += 1
		}else{
			request_success += 1
		}
		
		// sleep time
		time.Sleep(time.Millisecond * 500)
	}
}


// contents analysis
func contentsAnalysis(ctx context.Context, uri string) {
	res, err := http.Get(uri)
	if nil != err {
		fmt.Println("url: ", uri, " request failure, err:", err)
		return
	}
	defer res.Body.Close()

	utfbody, err := iconv.NewReader(res.Body, "gbk", "utf-8")
	if nil != err {
		fmt.Println("url: ", uri, " html ivov failure, err:", err)
		return
	}

	doc, err := goquery.NewDocumentFromReader(utfbody)
	if nil != err {
		fmt.Println("url: ", uri, " goqeury failure, err:", err)
		return
	}

	// body analysis
	doc.Find(".blogMain .blogRight .art .modCon .blogList").Each(func(i int, s * goquery.Selection){
		art := s.Find(".artHead div .artTitle a")
		title := art.Text()

		path, _ := art.Attr("href")
		u, _ := url.Parse(uri)
		//fmt.Printf("title: %s, url: %s\n", title, u.Scheme + "://" + u.Host + path)
		go requestPV(ctx, title, u.Scheme + "://" + u.Host + path)
	})
}

func homePage(ctx context.Context, hostUrl string) {
	doc, err := goquery.NewDocument(hostUrl)
	if nil != err {
		fmt.Println("url: ", hostUrl, " request failure, err:", err)
		return
	}

	var (
		pathPre string
		maxPage int
		currPage int
		url string = hostUrl
	)

	if val, ok := doc.Find(".blogMain .blogRight .art .modCon .pages a").Last().Attr("href"); ok {
		paths := strings.Split(val, "-")
		pathPre = paths[0]
		maxPage, _  = strconv.Atoi(paths[len(paths) - 1])
	}

	fmt.Println("pathPre:", pathPre, " maxPage:", maxPage, " currPage:", currPage, " url:", url)

	for {
		select {
		case <- ctx.Done():
			break
		default:
		}

		go contentsAnalysis(ctx, url)

		if currPage >= maxPage {
			break
		}
		currPage += 1
		url = hostUrl + pathPre + "-" + strconv.Itoa(currPage)
	}

}

func main(){
	// set go max runtine
	runtime.GOMAXPROCS(runtime.NumCPU())

	// start log output
	fmt.Println("server start...")

	// create context
	ctx, cancel := context.WithCancel(context.Background())

	// start content
	go homePage(ctx, REQUEST_HOST)

	c := make(chan os.Signal)
	// register signal
	signal.Notify(c, syscall.SIGINT, syscall.SIGHUP)
	
	// listen signal
	<- c
	// send context cancel
	cancel()
	// info
	fmt.Printf("\nserver 3 second stop")
	// sleep runtime output info
	//time.Sleep(time.Second * 3)
	for i := 0; i < 3; i++ {
		time.Sleep(time.Second)
		fmt.Print(".")
	}
	fmt.Printf("\n")
	
	// stop log output
	fmt.Println("server stop...")
}


