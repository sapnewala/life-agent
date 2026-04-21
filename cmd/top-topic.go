package main

import (
	"flag"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/sapnewala/life-agent/pkg/http2"
)

type Stock struct {
	Code                  string  `json:"code"`
	KoName                string  `json:"ko_name"`
	EnName                string  `json:"en_name"`
	Industry              string  `json:"industry"`
	Close                 float64 `json:"close"`
	Returns               float64 `json:"returns"`
	Volume                float64 `json:"volume"`
	VolumeReturns         float64 `json:"volume_returns"`
	VolumeValued          float64 `json:"volume_valued"`
	VolumeValuedReturns   float64 `json:"volume_valued_returns"`
	Logo                  string  `json:"logo"`
	StockId               int     `json:"stock_id"`
	ReferenceClose        float64 `json:"reference_close"`
	ReferenceVolume       float64 `json:"reference_volume"`
	ReferenceVolumeValued float64 `json:"reference_volume_valued"`
}

type StockResponse struct {
	Count int     `json:"count"`
	Data  []Stock `json:"data"`
}

type StockTheme struct {
	Id         int    `json:"id"`
	Name       string `json:"name"`
	BigThemeId int    `json:"big_theme_id"`
	IsOld      bool   `json:"is_old"`
	Reason     string `json:"reason"`
}

type StockThemesResponse struct {
	Data []StockTheme `json:"data"`
}

//const BASE_URL = "http://localhost:8081"
const BASE_URL = "http://140.245.67.237:8081"

func init() {
}

func main() {
	flag.Parse()
	limit := flag.Int("limit", 30, "limit")
	top := flag.Int("top", 10, "top")
	minChange := flag.Float64("min_change", 4.0, "min change rate")

	themeCounts := make(map[int]int)
	themeNames := make(map[int]string)
	themeStocks := make(map[int]string)

	url := BASE_URL + "/stock/volume_valued_top?limit=" + strconv.Itoa(*limit)
	resp, err := http2.Get(url, nil, nil, 10*time.Second, http2.DefaultRetryConfig)
	if err != nil {
		log.Fatalf("Failed to get stock volume valued top: %v", err)
	}
	//fmt.Println(string(resp.Body))

	//{"count":2462,"data":[{"code":"005930","ko_name":"삼성전자","en_name":"SamsungElec","industry":"반도체/반도체장비","close":213000.0,"returns":8.3969,"volume":81498731.0,"volume_returns":20.6853,"volume_valued":17111188967509.0,"volume_valued_returns":28.3288,"logo":"https://file.alphasquare.co.kr/media/images/stock_logo/kr/005930.png","stock_id":1014,"reference_close":196500.0,"reference_volume":67529951,"reference_volume_valued":13333867748200.0},
	var data StockResponse
	err = resp.DecodeJSON(&data)
	if err != nil {
		log.Fatalf("Failed to decode stock volume valued top: %v", err)
	}
	for i, stock := range data.Data {
		if stock.Returns < *minChange {
			continue
		}
		url2 := BASE_URL + "/stock/themes-for-stock?stock_id=" + strconv.Itoa(stock.StockId)
		resp2, err := http2.Get(url2, nil, nil, 10*time.Second, http2.DefaultRetryConfig)
		if err != nil {
			log.Fatalf("Failed to get stock themes for stock: %v", err)
		}

		// {"data":[{"id":22,"name":"원전","big_theme_id":36,"is_old":false,"reason":"원자로 용기 제작이 가능한 단조 설비를 가진 세계적인 회사 중 하나이며, 국내 첫 원자력 발전 해외 수출 1호인 아랍에미리트(UAE) 원자력 발전소 주기기 공급사로 참여"},
		var themesData StockThemesResponse
		err = resp2.DecodeJSON(&themesData)
		if err != nil {
			log.Fatalf("Failed to decode stock themes for stock: %v", err)
		}
		var themes []string
		for _, theme := range themesData.Data {
			themeCounts[theme.Id]++
			themeNames[theme.Id] = theme.Name
			themeStocks[theme.Id] = themeStocks[theme.Id] + "," + stock.KoName
			themes = append(themes, fmt.Sprintf("%d: %s", theme.Id, theme.Name))
		}
		fmt.Println(i, stock.KoName, stock.Close, stock.Returns, strings.Join(themes, ", "))
	}

	type themeCount struct {
		Id    int
		Name  string
		Count int
	}

	results := make([]themeCount, 0, len(themeCounts))
	for id, count := range themeCounts {
		results = append(results, themeCount{
			Id:    id,
			Name:  themeNames[id],
			Count: count,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Count == results[j].Count {
			return results[i].Id < results[j].Id
		}
		return results[i].Count > results[j].Count
	})

	if len(results) < *top {
		*top = len(results)
	}

	fmt.Println("==== Top ", *top, " Themes ====")
	for i := 0; i < *top; i++ {
		t := results[i]
		fmt.Printf("%d) %d / %s / count=%d / stocks=%s\n", i+1, t.Id, t.Name, t.Count, themeStocks[t.Id])
	}
}
