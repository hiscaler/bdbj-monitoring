package main

// 监控百度百家帐号最后发文时间

import (
	"os"
	"log"
	"fmt"
	"strings"
	"bufio"
	"net/http"
	"io/ioutil"
	"encoding/json"
	"regexp"
	"encoding/csv"
	"runtime"
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

func get(url string, modifyTime chan string) {
	fmt.Println(url)
	appId := strings.Replace(url, "https://baijiahao.baidu.com/u?app_id=", "", 1)
	apiUrl := strings.Replace("https://author.baidu.com/pipe?tab=2&app_id={appId}&num=6&pagelets[]=article&reqID=1&ispeed=1", "{appId}", appId, 1)
	// 发起请求
	req, err := http.NewRequest("GET", apiUrl, nil)
	if err != nil {
		log.Fatalln(err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/70.0.3538.110 Safari/537.36")
	req.Header.Set("Referer", url)
	req.Header.Set("Host", "author.baidu.com")
	req.Header.Set("Cookie", "PSTM=1493687267; BIDUPSID=8AD0DF0F2DBFDA345617B6A86D49F575; BAIDUID=6B8F061B7B14E2B7B7791909C97EF073:SL=0:NR=10:FG=1; BDUSS=kt4WWVaY2dmWDFDNnZwMEpHTDRJa3FJOGlHbDVTYWRRLWd2R0pBSzZKWDNMVnhiQVFBQUFBJCQAAAAAAAAAAAEAAAB6CvtkAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAPegNFv3oDRbQ; SIGNIN_UC=70a2711cf1d3d9b1a82d2f87d633bd8a02814595666; H_WISE_SIDS=125822_125302_122159_126245_125663_126305_126121_125915_120219_123018_118892_118876_118853_118818_118801_107318_126026_126145_117328_117428_125776_126442_124619_126380_126163_125926_126200_126094_125853_126055_124030_125058_110085_125451; delPer=0; PSINO=7; BDRCVFR[feWj1Vr5u3D]=I67x6TjHwwYf0; locale=zh; cflag=15%3A3; BCLID=8492456839767200788; BDSFRCVID=Rw-OJeC62ZuQjVJ9WLsdMFd0Em5jAG6TH6aIfv_PQCvUdnZhdpaHEG0Pef8g0KubYanIogKKLmOTHpKF_2uxOjjg8UtVJeC6EG0P3J; H_BDCLCKID_SF=tJFHVI_MJD83D-blqRrHh4-hMMr-J5_XKKOLVMo5Hl7keq8CDl6h0tIZyGrUWnJbBmcbop5EJRcNsIQ2y5jHytKBBn3I3b5P5IozBnbMKlnpsIJMbtDWbT8U5f5k546AaKviahREBMb1qhvDBT5h2M4qMxtOLR3pWDTm_q5TtUt5OCcnK4-Xj5oQeHoP; H_PS_PSSID=26524_1460_21108_18560_28019_22160; BDORZ=B490B5EBF6F3CD402E515D22BCDA1598")

	response, err1 := http.DefaultClient.Do(req)
	if err1 != nil {
		log.Fatalln(err1)
	}
	content, err := ioutil.ReadAll(response.Body)
	respBody := string(content)
	respBody = strings.Replace(respBody, "BigPipe.onPageletArrive(", "", 1)
	respBody = strings.Replace(respBody, ");", "", -1)
	baiDuBaiJiaResponse := BaiDuBaiJiaResponse{}
	err = json.Unmarshal([]byte(respBody), &baiDuBaiJiaResponse)
	if err != nil {
		fmt.Println("error in translating,", err.Error())
	}
	reg := regexp.MustCompile(`<div class="time">(.*?)</div>`)
	matchers := reg.FindStringSubmatch(baiDuBaiJiaResponse.Html)
	if len(matchers) >= 2 {
		modifyTime <- matchers[1]
		fmt.Println("最后修改时间: " + matchers[1])
	} else {
		modifyTime <- ""
	}
}

func main() {
	file, err := os.Open("./src/data/urls.txt")
	if err != nil {
		log.Fatalln(err)
	}
	defer file.Close()

	rows := make([]BaiDuBaiJiaItem, 0)
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	n := 0
	modifyTime := make(chan string, 500)
	runtime.GOMAXPROCS(1)
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

		go get(item.Url, modifyTime)
		item.ModifyTime = <-modifyTime
		rows = append(rows, item)
	}
	close(modifyTime)

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
