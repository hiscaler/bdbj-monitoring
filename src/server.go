package main

// 监控百度百家帐号最后发文时间

import (
	"os"
	"log"
	"fmt"
	"strings"
	"bufio"
	"encoding/csv"
	"sync"
	"net/http"
	"io/ioutil"
	"encoding/json"
	"regexp"
	"time"
)

type BaiDuBaiJiaItem struct {
	CategoryName string
	Name         string
	Url          string
	ModifyTime   string
}

type BaiDuBaiJiaResponse struct {
	Container string
	ReqID     int
	Id        string
	Html      string
}

var wg sync.WaitGroup

func get(item BaiDuBaiJiaItem, ch chan BaiDuBaiJiaItem, wg *sync.WaitGroup, client *http.Client) {
	wg.Add(1)
	url := item.Url
	appId := strings.Replace(url, "https://baijiahao.baidu.com/u?app_id=", "", 1)
	apiUrl := strings.Replace("https://author.baidu.com/pipe?tab=2&app_id={appId}&num=6&pagelets[]=article&reqID=1&ispeed=1", "{appId}", appId, 1)
	fmt.Println(apiUrl)
	// 发起请求
	req, err := http.NewRequest("GET", apiUrl, nil)
	//defer req.Body.Close()
	if err != nil {
		log.Fatalln(err)
	} else {
		req.Close = true
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/70.0.3538.110 Safari/537.36")
		req.Header.Set("Referer", url)
		req.Header.Set("Host", "author.baidu.com")
		req.Header.Set("Cookie", "PSTM=1493687267; BIDUPSID=8AD0DF0F2DBFDA345617B6A86D49F575; BAIDUID=6B8F061B7B14E2B7B7791909C97EF073:SL=0:NR=10:FG=1; BDUSS=kt4WWVaY2dmWDFDNnZwMEpHTDRJa3FJOGlHbDVTYWRRLWd2R0pBSzZKWDNMVnhiQVFBQUFBJCQAAAAAAAAAAAEAAAB6CvtkAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAPegNFv3oDRbQ; SIGNIN_UC=70a2711cf1d3d9b1a82d2f87d633bd8a02814595666; H_WISE_SIDS=125822_125302_122159_126245_125663_126305_126121_125915_120219_123018_118892_118876_118853_118818_118801_107318_126026_126145_117328_117428_125776_126442_124619_126380_126163_125926_126200_126094_125853_126055_124030_125058_110085_125451; delPer=0; PSINO=7; BDRCVFR[feWj1Vr5u3D]=I67x6TjHwwYf0; locale=zh; cflag=15%3A3; BCLID=8492456839767200788; BDSFRCVID=Rw-OJeC62ZuQjVJ9WLsdMFd0Em5jAG6TH6aIfv_PQCvUdnZhdpaHEG0Pef8g0KubYanIogKKLmOTHpKF_2uxOjjg8UtVJeC6EG0P3J; H_BDCLCKID_SF=tJFHVI_MJD83D-blqRrHh4-hMMr-J5_XKKOLVMo5Hl7keq8CDl6h0tIZyGrUWnJbBmcbop5EJRcNsIQ2y5jHytKBBn3I3b5P5IozBnbMKlnpsIJMbtDWbT8U5f5k546AaKviahREBMb1qhvDBT5h2M4qMxtOLR3pWDTm_q5TtUt5OCcnK4-Xj5oQeHoP; H_PS_PSSID=26524_1460_21108_18560_28019_22160; BDORZ=B490B5EBF6F3CD402E515D22BCDA1598")

		response, err1 := client.Do(req)
		if err1 != nil {
			log.Fatalln(item.Url, err1)
		} else {
			content, err := ioutil.ReadAll(response.Body)
			if err != nil {
				log.Fatalln(err)
			} else {
				respBody := string(content)
				respBody = strings.Replace(respBody, "BigPipe.onPageletArrive(", "", 1)
				respBody = strings.Replace(respBody, ");", "", -1)
				baiDuBaiJiaResponse := BaiDuBaiJiaResponse{}
				err = json.Unmarshal([]byte(respBody), &baiDuBaiJiaResponse)
				if err != nil {
					fmt.Println("Error in translating,", err.Error())
				}
				reg := regexp.MustCompile(`<div class="time">(.*?)</div>`)
				matchers := reg.FindStringSubmatch(baiDuBaiJiaResponse.Html)
				if len(matchers) >= 2 {
					item.ModifyTime = matchers[1]
				}
		}
		}
	}

	time.Sleep(time.Second * 3)
	ch <- item
}

func main() {
	client := &http.Client{
		CheckRedirect: nil,
	}
	ch := make(chan BaiDuBaiJiaItem, 10)
	defer close(ch)
	file, err := os.Open("./src/data/urls.txt")
	if err != nil {
		log.Fatalln(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	n := 0
	for scanner.Scan() {
		item := BaiDuBaiJiaItem{}
		n++
		fmt.Println("################################################################################")
		fmt.Println(fmt.Sprintf("读取第 %d 行中...", n))
		fields := strings.Split(scanner.Text(), "\t")
		fieldsCount := len(fields)
		if fieldsCount == 3 {
			item.CategoryName = fields[0]
			item.Name = fields[1]
			item.Url = fields[2]
		} else if fieldsCount == 2 {
			item.CategoryName = ""
			item.Name = fields[0]
			item.Url = fields[1]
		} else if fieldsCount == 1 {
			item.CategoryName = ""
			item.Name = ""
			item.Url = fields[0]
		} else {
			log.Fatalln("行内容为空，忽略。")
			continue
		}
		if n%20 == 0 {
			// 需要歇息一下，否则服务端会主动关闭连接
			time.Sleep(time.Second * 2)
		}

		go get(item, ch, &wg, client)
	}

	rows := make([]BaiDuBaiJiaItem, 0)
	go func(ch2 chan BaiDuBaiJiaItem, wg *sync.WaitGroup) {
		for {
			select {
			case v := <-ch2:
				fmt.Println("Read value is", v)
				rows = append(rows, v)
				wg.Done()
			}
		}
	}(ch, &wg)
	wg.Wait()

	// Save to CSV file
	csvFile, err := os.Create("./src/data/urls-done.csv")
	if err != nil {
		panic(err)
	}
	defer csvFile.Close()

	csvWriter := csv.NewWriter(csvFile)
	defer csvWriter.Flush()
	fmt.Println("开始写入 CSV 文件")
	for _, value := range rows {
		d := []string{
			value.CategoryName,
			value.Name,
			value.Url,
			value.ModifyTime,
		}
		if err := csvWriter.Write(d); err != nil {
			log.Fatalln("Error writing record to csv:", err)
		}
		fmt.Print(".")
	}
	fmt.Println("")
	fmt.Println("数据处理完毕")
}
