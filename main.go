package main

import (
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type EachCity func(string, CityInfo) bool
type CityInfo struct {
	//{"id":"101010100","cityEn":"beijing","cityZh":"\u5317\u4eac","provinceEn":"beijing",
	//"provinceZh":"\u5317\u4eac","leaderEn":"beijing","leaderZh":"\u5317\u4eac",
	//"lat":"39.904989","lon":"116.405285"}
	Id         string `json:"id"`
	CityEn     string `json:"cityEn"`
	CityZh     string `json:"cityZh"`
	ProvinceEn string `json:"provinceEn"`
	ProvinceZh string `json:"provinceZh"`
	LeaderEn   string `json:"leaderEn"`
	LeaderZh   string `json:"leaderZh"`
	Lat        string `json:"lat"`
	Lon        string `json:"lon"` //data
}

//天气加距离查询
func main() {
	fzinfo := GetCityInfo("福州")
	bjinfo := GetCityInfo("北京")
	fmt.Printf("%s到%s 距离%.2f千米\n",
		fzinfo.CityZh,
		bjinfo.CityZh,
		CalcDistance(fzinfo, bjinfo))
	info := QueryWeather(30, 1100, "雪", "", "15", fzinfo)

	println(info)
	//println(QueryWeather(30, -1, "大雪", "", "7", fzinfo))
	//println(QueryWeather(30, -1, "暴", "", "7", fzinfo))
	//println(QueryWeather(30, -1, "暴", "", "15", fzinfo))
	//println(QueryWeather(30, -1, "雨夹雪", "", "15", fzinfo))
	//println(QueryWeather(, fzinfo))

}
func GetCityInfo(name string) CityInfo {
	var reInfo CityInfo

	EchoCity(func(id string, info CityInfo) bool {
		if info.CityZh == name {
			reInfo = info
			return true
		}
		return false
	})
	return reInfo
}
func CalcDistance(fromInfo, toInfo CityInfo) float64 {
	return GetDistance(toFloat(fromInfo.Lat), toFloat(toInfo.Lat), toFloat(fromInfo.Lon), toFloat(toInfo.Lon))
}
func toFloat(s string) float64 {

	v1, err := strconv.ParseFloat(s, 32)
	if err != nil {
		panic(err.Error())
		return -1
	}
	return v1
}
func EchoCity(f EachCity) {
	open, err := os.Open("city.json")
	if err != nil {
		println(err.Error())
		return
	}
	var data = []CityInfo{}
	err = json.NewDecoder(open).Decode(&data)
	if err != nil {
		println(err.Error())
		return
	}
	for i := range data {

		if f(data[i].Id, data[i]) { //true时退出
			return
		}
	}
}

func QueryWeather(maxItemLimit, distanceLimit int, weatherKeyword, provinceName, nextDays_7_or_15 string, fromMiddleCity CityInfo) string {
	infoStr := "全国"
	if provinceName != "" {
		infoStr = provinceName
	}
	infoStr += "未来" + nextDays_7_or_15 +
		"天有" + weatherKeyword + " " + toString(maxItemLimit) + "个城市"
	if distanceLimit > 0 {
		infoStr += "(距离" + toString(distanceLimit) + "千米之内)"
	}
	foundCount := 0
	EchoCity(func(id string, info CityInfo) bool {

		distance := CalcDistance(fromMiddleCity, info)
		if distanceLimit > 0 && distance > float64(distanceLimit) {
			return false
		}
		if provinceName != "" && info.ProvinceZh != provinceName {
			return false
		}
		winfo := ""
		if nextDays_7_or_15 == "15" {
			winfo = GetWeatherInfo15d(id)
		} else {
			winfo = GetWeatherInfo7d(id)
		}
		if strings.Contains(winfo, weatherKeyword) {
			infoStr += "\n" + toString(foundCount+1) + "." + id
			infoStr += " " + info.ProvinceZh + "-" + info.CityZh
			if distance > 0 {
				infoStr += " 距离: " + toString(distance) + " 千米"
			}
			foundCount++
			return foundCount >= maxItemLimit //查到限制个数就退出
		}
		return false
	})
	return infoStr + "\n(查询结果:" + toString(foundCount) + "个城市)"
}
func toString(v interface{}) string {
	switch v.(type) {
	case float64:
		return strconv.FormatFloat(v.(float64), 'f', -0, 32)
	case int:
		return strconv.Itoa(v.(int))
	}
	return ""
}
func GetWeatherInfo15d(cityId string) string {
	return getWeather("15", cityId)
}
func GetWeatherInfo7d(cityId string) string {
	return getWeather("7", cityId)
}
func getWeather(days, cityId string) string {
	pathStr := "cache/" + days + "d"
	_ = os.MkdirAll(pathStr, 0777)
	fileName := pathStr + "/" + cityId + ".json"
	s, _ := os.Stat(fileName)
	if s != nil && s.Size() > 0 {
		re, _ := ioutil.ReadFile(fileName)
		return string(re)
	}
	newStr := getWeatherInfoNday(days, cityId)
	if newStr != "" {
		_ = ioutil.WriteFile(fileName, []byte(newStr), 0777)
	}
	return newStr
}
func getWeatherInfoNday(days, cityId string) string {
	s := "http://www.weather.com.cn/weather"

	if days == "15" {
		s += "15d"
	} else if days != "7" {
		panic("目前只支持,7天与15天")
	}
	s += "/" + cityId + ".shtml"
	//doc, err := goquery.NewDocument(s)
	rsp, err := http.Get(s)
	if err != nil {
		return err.Error()
	}

	doc, err := goquery.NewDocumentFromReader(rsp.Body)
	if err != nil {
		return err.Error()
	}

	//ra,e:=ioutil.ReadAll(rsp.Body)
	//htmlStr:=string(ra)

	//	<div id="15d" class="c15d">
	div := doc.Find("div#" +
		days +
		"d>ul>li")
	if len(div.Nodes) == 0 {
		return ""
	}
	//<ul class="t clearfix">
	//<li class="t">
	//<span class="time">周二（14日）</span>
	//<big class="png30 d01"></big>
	//<big class="png30 n02"></big>
	//<span class="wea">多云转阴</span>
	//<span class="tem"><em>17℃</em>/11℃</span>
	//<span class="wind">东北风</span>
	//<span class="wind1">&lt;3级</span>
	//</li>
	ss := []map[int]string{}
	for _, node := range div.Nodes {
		//println(i, node.Data)
		item := map[int]string{}

		fc := node.FirstChild
		for {
			if fc == nil {
				break
			}
			if fc.FirstChild != nil {
				item[(len(item))] = fc.FirstChild.Data
				//println("\t:", fc.FirstChild.Data)

			}
			fc = fc.NextSibling
		}
		ss = append(ss, item)

	}
	b1, _ := json.MarshalIndent(&ss, " ", " ")
	return string(b1)
}
func getWeatherInfo(cityId string) string {
	//1天
	//#http://www.weather.com.cn/weather1d/101230101.shtml
	//15天
	///http://www.weather.com.cn/weather15d/101230101.shtml
	//7天
	//http://www.weather.com.cn/weather/101230101.shtml
	s := "http://www.weather.com.cn/weather1d/" + cityId + ".shtml"
	//s := "http://www.weather.com.cn/weather15d/" + cityId + ".shtml"
	rsp, err := http.Get(s)
	if err != nil {
		return err.Error()
	}
	ra, e := ioutil.ReadAll(rsp.Body)
	if e != nil {
		return e.Error()
	}
	htmlStr := string(ra)
	//rx,e:=regexp.Compile("<script[^>]+>([\\s\\S]*?)</script>")
	rx, e := regexp.Compile("<script>([\\s\\S]*?)</script>")
	if e != nil {
		return e.Error()
	}
	mstrs := rx.FindAllStringSubmatch(htmlStr, -1)
	for i := range mstrs {
		println(mstrs[i][1])
		//for j := range mstrs[i] {
		//	println(mstrs[i][j])
		//}
	}
	return ""
}

//返回单位为：千米
func GetDistance(lat1, lat2, lng1, lng2 float64) float64 {
	radius := 6371000.0 //6378137.0
	rad := math.Pi / 180.0
	lat1 = lat1 * rad
	lng1 = lng1 * rad
	lat2 = lat2 * rad
	lng2 = lng2 * rad
	theta := lng2 - lng1
	dist := math.Acos(math.Sin(lat1)*math.Sin(lat2) + math.Cos(lat1)*math.Cos(lat2)*math.Cos(theta))
	return dist * radius / 1000
}
