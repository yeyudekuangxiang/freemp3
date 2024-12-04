package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/dhowden/tag"
	flac2 "github.com/eaburns/flac"
	"github.com/jfreymuth/oggvorbis"
	"github.com/mewkiz/flac"
	"github.com/yeyudekuangxiang/common-go/db"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"io"
	"log"
	"math"
	"mime"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

var downPath = flag.String("down", "./music", "")
var num = flag.Int("num", 2, "")
var mode = flag.String("mode", "", "")

type ArtistDetailReq struct {
	ArtistDetailReqToken
	Token string `json:"token"`
}
type ArtistDetailReqToken struct {
	Id string `json:"id"`
	T  int64  `json:"_t"`
}

var p int64

func replace(dirPath string) {
	dirs, err := os.ReadDir(dirPath)
	if err != nil {
		log.Panic(err)
	}

	for _, dir := range dirs {
		if dir.IsDir() {
			replace(path.Join(dirPath, dir.Name()))
			p++
			log.Println("已处理", p)
		} else {
			if strings.Contains(dir.Name(), "[music.migu.cn]") {

				oldName := path.Join(dirPath, dir.Name())
				newName := path.Join(dirPath, strings.ReplaceAll(dir.Name(), "[music.migu.cn]", ""))
				err := os.Rename(oldName, newName)
				if err != nil {
					log.Println("重命名失败", dir.Name(), err)
				}
			}
		}
	}
}

var header = map[string]string{}

func main() {
	requestHeaderStr := os.Getenv("freemp3header")
	if requestHeaderStr == "" {
		log.Panicln("请求header为空")
	}
	err := json.Unmarshal([]byte(requestHeaderStr), &header)
	if err != nil {
		log.Panicln("解析header失败", requestHeaderStr, err)
	}

	log.Printf("header%+v\n", header)
	linkDb, err := db.NewMysqlDB(db.Config{
		Type:         "mysql",
		Host:         "nas.znil.cn",
		UserName:     "jzl",
		Password:     "ZHUImeng521..",
		Database:     "freemusic",
		Port:         3306,
		TablePrefix:  "",
		MaxOpenConns: 20,
		MaxIdleConns: 10,
		MaxIdleTime:  100,
		MaxLifetime:  100,
		Logger:       logger.New(log.New(os.Stdout, "", 0), logger.Config{}),
	})
	if err != nil {
		log.Panic("连接数据库失败", err)
	}

	flag.Parse()

	go func() {
		err := http.ListenAndServe(":3000", nil)
		if err != nil {
			log.Println("启动pprof服务器失败", err)
		}
	}()

	go func() {
		// 创建一个通道来接收信号
		sigChan := make(chan os.Signal, 1)

		// 注册要监听的信号
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		cmd := exec.Command("node", "index.js")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		// 启动一个goroutine来处理信号
		go func() {
			<-sigChan
			log.Println("退出程序")
			if cmd.Process != nil {
				cmd.Process.Kill()
			}
			os.Exit(0)
		}()

		err := cmd.Run()
		if err != nil {
			log.Println(err)
		}
		log.Println("http已退出")
		/*err := http.ListenAndServe(":8022", http.FileServer(http.Dir("./")))
		if err != nil {
			log.Panicln(err)
		}*/
	}()

	log.Println("5秒后开启抓取")
	time.Sleep(time.Second * 5)
	for i := 0; i <= 10; i++ {
		if i == 10 {
			log.Println("http服务未启动,停止抓取")
			os.Exit(1)
		}
		resp, err := http.Get("http://127.0.0.1:3002/index.html")
		if err != nil {
			log.Println("检测失败，5秒后重试", err)
		} else {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				log.Println("http服务已启动")
				break
			} else {
				log.Println("http服务未启动,5秒后重试", resp.StatusCode)
			}
		}
		time.Sleep(5 * time.Second)
	}
	switch *mode {
	case "artist":
		log.Println("拉取歌手")
		saveAllArtist(linkDb)
	case "music":
		log.Println("拉取歌曲")
		saveAllMusic(linkDb)
	case "url":
		log.Println("获取歌曲下载链接")
		getAllMusicDownUrl(linkDb)
	case "down":
		log.Println("下载歌曲")
		downAllMusic(linkDb)

	default:
		log.Println("未知模式", *mode)
	}
	os.Exit(0)

	for _, name := range []string{} {
		downSinger(name)
	}
	for ai := 1; ai < 50; ai++ {
		listResp, err := GetArtistList(ai)
		if err != nil {
			log.Panicln(err.Error())
		}
		if listResp.Code != 200 {
			log.Panicln(listResp.Code)
		}
		if len(listResp.Data.List) == 0 {
			break
		}
		for _, item := range listResp.Data.List {
			downSinger(item.Name)
		}
	}
}
func saveAllArtist(linkDb *gorm.DB) {

	artistList := make([]Artist, 0)
	err := linkDb.Find(&artistList).Error
	if err != nil {
		log.Panicln("查询歌手失败", err)
	}
	existListMap := make(map[string]bool)
	for _, item := range artistList {
		existListMap[item.ArtistId] = true
	}

	for i := 1; i < 50; i++ {
		insertList := make([]Artist, 0)
		resp, err := GetArtistList(i)
		if err != nil {
			log.Panicln("获取歌手列表失败", i, err)
		}
		if resp.Code != 200 {
			log.Panicln("获取歌手列表失败", i, resp.Code, resp.Msg)
		}
		if len(resp.Data.List) == 0 {
			log.Println("歌手列表以获取完毕", i)
			break
		}
		for _, ar := range resp.Data.List {
			if existListMap[ar.Id] {
				log.Println("歌手已存在")
				continue
			}
			insertList = append(insertList, Artist{
				ArtistId: ar.Id,
				Name:     ar.Name,
				Pic:      ar.Pic,
			})
		}
		if len(insertList) == 0 {
			continue
		}
		err = linkDb.Create(&insertList).Error
		if err != nil {
			log.Panicln("入库失败", i, err)
		}
		log.Println("5秒后继续获取")
		time.Sleep(5 * time.Second)
	}
}
func fillReq(req *http.Request) {
	for k, v := range header {
		req.Header.Add(k, v)
	}
	return
	req.Header.Add("accept", "application/json, text/plain, */*")
	req.Header.Add("accept-language", "zh-CN,zh;q=0.9")
	req.Header.Add("cache-control", "no-cache")
	req.Header.Add("content-type", "application/json;charset=UTF-8")
	req.Header.Add("cookie", "cf_clearance=zSscDE8WTP9o4BnvuZUHjmesQIhFDDS8mTa_mtydt.A-1733194151-1.2.1.1-JHvSj3qWA2c9AGHD2UcR3TMV8.ladnVTzxfgGc4XRsBExr5A1oHvRrjz8ab.xCGxVHup79XUIgadJwyO1rd03x4LIOLlM8iSn82_d6X8zsLYwZ7IBUeX3R8ejuUIMkfCyusBptcYrIQCX_ewTfAC4wbbaXljfFIKb3ONrxLItovQh_Q5F2YutrFGDNyEoKOWu3CpZHDFRD6kUwyUlrbnFnUaz3nsTBHvEZLrBur_alXS8MnrkdBm2pNzeyyZ2qy6WM8f5vh76pA5liRzaK02zLPwbCA8VIv4_B5kUZChuemJzQ7qqkxyWoqqtqpCscv2WxGAatdImW4i50dqIRbj4Pwa0hJvAlSwlYrdUVpYdkEAgqA8RwxBFJC2_3aUN5CT_7pNMqz8Ln3Ex52dco1SjA; __gads=ID=a272399b69834f90:T=1731397705:RT=1733194151:S=ALNI_MaqdqkKdqQlexWAwVaiXUOT7B-eCA; __gpi=UID=00000db35530df2f:T=1731397705:RT=1733194151:S=ALNI_MbRkas7oneAguteRsqRc3jzwZhGEg; __eoi=ID=495a49f5fddb9e51:T=1731397705:RT=1733194151:S=AA-AfjYg5xi01cqDAASUWzyt3zC0; FCNEC=%5B%5B%22AKsRol84z8RsbKxZ6qjA_LhXPkb9FS2xykA2sSdqgVylQN-nLjIQec9E8ImgddiROgmqxEONiHtiCW0a39PESP5gaBsJpqza-YcU6O-S-xye12CEUeyRtc6uM62N8r2fhDKU3cxfMYha_hOMmI8C1mmz3QZuEv1ygA%3D%3D%22%5D%5D")
	req.Header.Add("origin", "https://tool.liumingye.cn")
	req.Header.Add("pragma", "no-cache")
	req.Header.Add("priority", "u=1, i")
	req.Header.Add("sec-ch-ua", "\"Google Chrome\";v=\"131\", \"Chromium\";v=\"131\", \"Not_A Brand\";v=\"24\"")
	req.Header.Add("sec-ch-ua-arch", "\"x86\"")
	req.Header.Add("sec-ch-ua-bitness", "\"64\"")
	req.Header.Add("sec-ch-ua-full-version", "\"131.0.6778.86\"")
	req.Header.Add("sec-ch-ua-full-version-list", "\"Google Chrome\";v=\"131.0.6778.86\", \"Chromium\";v=\"131.0.6778.86\", \"Not_A Brand\";v=\"24.0.0.0\"")
	req.Header.Add("sec-ch-ua-mobile", "?0")
	req.Header.Add("sec-ch-ua-model", "\"\"")
	req.Header.Add("sec-ch-ua-platform", "\"Windows\"")
	req.Header.Add("sec-ch-ua-platform-version", "\"15.0.0\"")
	req.Header.Add("sec-fetch-dest", "empty")
	req.Header.Add("sec-fetch-mode", "cors")
	req.Header.Add("sec-fetch-site", "same-origin")
	req.Header.Add("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")

}
func saveAllMusic(linkDb *gorm.DB) {
	artistList := make([]Artist, 0)
	err := linkDb.Find(&artistList).Error
	if err != nil {
		log.Panicln("查询所有歌手失败", err)
	}
	for _, singer := range artistList {
		if singer.IsFetch == 1 {
			log.Println("跳过歌手", singer.Name)
			continue
		}
		log.Println("搜索歌手", singer.Name)
		success := true
		for i := 1; i < 20; i++ {
			searchResp, err := Search(singer.Name, i)
			if err != nil {
				success = false
				log.Println("搜索失败", i, err)
				break
			}
			if searchResp.Code != 200 {
				success = false
				log.Println("搜索失败", i, searchResp)
				break
			}
			if len(searchResp.Data.List) == 0 {
				break
			}
			insertList := make([]Music, 0)
			for _, music := range searchResp.Data.List {
				insertList = append(insertList, Music{
					MusicId: music.Id,
					Name:    music.Name,
					Pic:     music.Pic,
					Lyric:   music.Lyric,
					Artist:  toJson(music.Artist),
					Album:   toJson(music.Album),
					Time:    music.Time,
					Quality: toJson(music.Quality),
				})
			}
			err = linkDb.Clauses(clause.OnConflict{DoNothing: true}).Create(&insertList).Error
			if err != nil {
				log.Panicln("创建歌曲失败", i, err)
			}
			log.Println("5秒后继续")
			time.Sleep(5 * time.Second)
		}
		if success {
			singer.IsFetch = 1
			err = linkDb.Save(&singer).Error
			if err != nil {
				log.Println("保存歌手歌曲拉取信息失败", singer, err)
			}
		}
	}
}
func downSinger(singerName string) {
	c := make(chan int, *num)
	_, err := os.Stat(path.Join(*downPath, singerName))
	if err == nil {
		log.Println("文件夹已存在", singerName)
		return
	}

	log.Println("搜索歌手歌曲", singerName)
	for i := 1; i < 20; i++ {
		searchResp, err := Search(singerName, i)
		if err != nil {
			log.Println("搜索失败", err)
			break
		}
		if searchResp.Code != 200 {
			log.Println("搜索失败", searchResp)
			break
		}
		if len(searchResp.Data.List) == 0 {
			break
		}
		wg := sync.WaitGroup{}
		for _, music := range searchResp.Data.List {
			log.Println("一秒后继续下载")
			time.Sleep(time.Second)
			c <- 1
			wg.Add(1)
			music := music
			go func() {
				defer func() {
					wg.Done()
					<-c
				}()
				log.Println("获取歌曲下载连接", music.Name)
				if len(music.Quality) == 0 {
					log.Println("没有质量列表", music.Name)
					return
				}
				q := music.Quality[len(music.Quality)-1]
				var u string
				switch qq := q.(type) {
				case int64:
					u, err = GetRealDownLoadUrl(music.Id, strconv.FormatInt(qq, 10))
				case string:
					u, err = GetRealDownLoadUrl(music.Id, qq)
				case map[string]interface{}:
					m := qq["name"].(string)
					u, err = GetRealDownLoadUrl(music.Id, m)
				case float64:
					u, err = GetRealDownLoadUrl(music.Id, strconv.FormatInt(int64(qq), 10))
				default:
					err = fmt.Errorf("music.%v", qq)
				}
				if err != nil {
					log.Println("获取下载连接失败", err)
					return
				}
				realSingerName := ""
				for _, ar := range music.Artist {
					realSingerName += "," + ar.Name
				}
				if len(realSingerName) > 0 {
					realSingerName = realSingerName[1:]
				}
				err = down(singerName, realSingerName, music.Name, u)
				if err != nil {
					/*if errors.Is(io.ErrUnexpectedEOF, err) {
						log.Println("下载中断重试一次", music.Name)
						err = down(singerName, music.Name, u)
						if err != nil {
							log.Println("重试失败", music.Name, err)
						}
					} else {
						log.Println("下载歌曲失败", music.Name, err)
					}*/
					log.Println("下载歌曲失败", music.Name, err)
					return
				}
			}()
		}
		wg.Wait()
	}
	/*artistResp, err := GetArtistDetail(item.Id)
	if err != nil {
		log.Panicln(err.Error())
	}
	if artistResp.Code != 200 {
		log.Panicln(artistResp.Code)
	}
	for _, music := range artistResp.Data.List {
		log.Println("获取歌曲下载连接", music.Name)
		q := music.Quality[len(music.Quality)-1]
		var u string
		switch qq := q.(type) {
		case int64:
			u, err = GetDownLoadUrl(music.Id, strconv.FormatInt(qq, 10))
		case string:
			u, err = GetDownLoadUrl(music.Id, qq)
		case map[string]interface{}:
			m := qq["name"].(string)
			u, err = GetDownLoadUrl(music.Id, m)
		case float64:
			u, err = GetDownLoadUrl(music.Id, strconv.FormatInt(int64(qq), 10))
		default:
			err = fmt.Errorf("music.%v", qq)
		}
		if err != nil {
			log.Println("获取下载连接失败", err)
			continue
		}
		err = down(singerName, u)

		if err != nil {
			log.Println("下载歌曲失败", err)
			continue
		}
		log.Println("一秒后继续下载")
		time.Sleep(time.Second)
	}*/
	close(c)
}
func encode(d interface{}) string {
	data, err := json.Marshal(d)
	if err != nil {
		log.Panicln(err)
	}

	basedata := base64.StdEncoding.EncodeToString(data)
	resp, err := http.Get("http://127.0.0.1:3002/encode?" + basedata)
	if err != nil {
		log.Panicln(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Panicln(resp.Status)
	}
	data, err = io.ReadAll(resp.Body)
	if err != nil {
		log.Panicln(err)
	}
	return string(data)
}

func main2(d interface{}) (string, error) {
	return encode(d), nil
	data, err := json.Marshal(d)
	if err != nil {
		return "", err
	}
	//log.Println(string(data))
	//data = []byte(`{"id":"zP8o","_t":1731480805511}`)
	basedata := base64.StdEncoding.EncodeToString(data)

	//log.Println(basedata)
	cmd := exec.Command("node", "index3.js", basedata)
	cmd.Dir = "./"
	encodeData, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(encodeData)), nil
}

type ArtistListResponse struct {
	Code int `json:"code"`
	Data struct {
		List []struct {
			Id   string `json:"id"`
			Name string `json:"name"`
			Pic  string `json:"pic"`
		} `json:"list"`
	} `json:"data"`
	Msg string `json:"msg"`
}
type ArtistDetailResponse struct {
	Code int `json:"code"`
	Data struct {
		Pic        string `json:"pic"`
		Name       string `json:"name"`
		Views      int    `json:"views"`
		UpdateTime string `json:"update_time"`
		List       []struct {
			Id      string        `json:"id"`
			Name    string        `json:"name"`
			Pic     *string       `json:"pic"`
			Url     *string       `json:"url"`
			Time    int           `json:"time"`
			Lyric   *string       `json:"lyric"`
			Quality []interface{} `json:"quality"`
			Album   *struct {
				Id   string `json:"id"`
				Name string `json:"name"`
				Pic  string `json:"pic"`
			} `json:"album"`
			Artist []struct {
				Id   string `json:"id"`
				Name string `json:"name"`
			} `json:"artist"`
			Pivot struct {
				ArtistId int `json:"artist_id"`
				TrackId  int `json:"track_id"`
				Sort     int `json:"sort"`
			} `json:"pivot"`
			Hash string `json:"hash"`
		} `json:"list"`
		Desc string `json:"desc"`
	} `json:"data"`
	Msg string `json:"msg"`
}
type ArtistListReqToken struct {
	Initial int   `json:"initial"`
	Page    int   `json:"page"`
	T       int64 `json:"_t"`
}
type ArtistListReq struct {
	ArtistListReqToken
	Token string `json:"token"`
}

func GetArtistList(page int) (*ArtistListResponse, error) {
	reqBodyObj := ArtistListReq{
		ArtistListReqToken: ArtistListReqToken{
			Initial: 0,
			Page:    page,
			T:       time.Now().UnixMilli(),
		},
	}
	token, err := main2(reqBodyObj.ArtistListReqToken)
	if err != nil {
		return nil, err
	}
	reqBodyObj.Token = token
	reqBody, err := json.Marshal(reqBodyObj)
	if err != nil {
		return nil, err
	}

	log.Println("GetArtistList", string(reqBody))
	req, err := http.NewRequest("POST", "https://api.liumingye.cn/m/api/artist/list", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	fillReq(req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, errors.New(resp.Status)
	}
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	v := ArtistListResponse{}
	return &v, json.Unmarshal(respBody, &v)
}
func GetArtistDetail(id string) (*ArtistDetailResponse, error) {
	reqBodyObj := ArtistDetailReq{
		ArtistDetailReqToken: ArtistDetailReqToken{
			Id: id,
			T:  time.Now().UnixMilli(),
		},
	}
	token, err := main2(reqBodyObj.ArtistDetailReqToken)
	if err != nil {
		return nil, err
	}
	reqBodyObj.Token = token
	reqBody, err := json.Marshal(reqBodyObj)
	if err != nil {
		return nil, err
	}
	log.Println("GetArtistDetail", string(reqBody))
	req, err := http.NewRequest("POST", "https://api.liumingye.cn/m/api/artist", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Add("accept", "application/json, text/plain, */*")
	req.Header.Add("accept-language", "zh-CN,zh;q=0.9")
	req.Header.Add("cache-control", "no-cache")
	req.Header.Add("content-type", "application/json;charset=UTF-8")
	req.Header.Add("origin", "https://tool.liumingye.cn")
	req.Header.Add("pragma", "no-cache")
	req.Header.Add("priority", "u=1, i")
	req.Header.Add("sec-ch-ua", "\"Chromium\";v=\"130\", \"Google Chrome\";v=\"130\", \"Not?A_Brand\";v=\"99\"")
	req.Header.Add("sec-ch-ua-mobile", "?0")
	req.Header.Add("sec-ch-ua-platform", "\"Windows\"")
	req.Header.Add("sec-fetch-dest", "empty")
	req.Header.Add("sec-fetch-mode", "cors")
	req.Header.Add("sec-fetch-site", "same-site")
	req.Header.Add("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36")
	req.Header.Add("Cookie", "sl-session=FbJJLA+nNWfoqUbiSRZyTQ==")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, errors.New(resp.Status)
	}
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	v := ArtistDetailResponse{}
	return &v, json.Unmarshal(respBody, &v)
}
func GetRealDownLoadUrl(id string, quality string) (string, error) {
	t := strconv.FormatInt(time.Now().UnixMilli(), 10)
	token, err := main2(DownLoadUrlReq{
		Id:      id,
		Quality: quality,
		T:       t,
	})
	if err != nil {
		return "", err
	}

	uuu := fmt.Sprintf("https://api.liumingye.cn/m/api/link?id=%s&quality=%s&_t=%s&token=%s", id, quality, t, token)
	log.Println("网盘跳转页", uuu)
	req, err := http.NewRequest("GET", uuu, nil)
	if err != nil {
		return "", err
	}
	fillReq(req)
	req.Header.Add("accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Del("Content-Type")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if strings.Contains(resp.Header.Get("Content-Type"), "audio") {
		if resp.StatusCode == http.StatusFound {
			location := resp.Header.Get("Location")
			if location != "" {
				return location, nil
			}
		}
		return resp.Request.URL.String(), nil
		//return uuu, nil
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return getLanRealDownFromBody(respBody)
}
func GetDownLoadUrl1(id string, quality string) (string, error) {
	t := strconv.FormatInt(time.Now().UnixMilli(), 10)
	token, err := main2(DownLoadUrlReq{
		Id:      id,
		Quality: quality,
		T:       t,
	})
	if err != nil {
		return "", err
	}

	uuu := fmt.Sprintf("https://api.liumingye.cn/m/api/link?id=%s&quality=%s&_t=%s&token=%s", id, quality, t, token)
	log.Println("网盘跳转页", uuu)
	req, err := http.NewRequest("GET", uuu, nil)
	if err != nil {
		return "", err
	}
	fillReq(req)
	req.Header.Add("accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Del("Content-Type")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	pageUrl := resp.Request.URL.String()
	if !strings.Contains(pageUrl, "liumingye") {
		return pageUrl, nil
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if bytes.Contains(body, []byte("版权")) {
		return "nocopyright", nil
	}
	return pageUrl, nil
}

type DownAjaxResp struct {
	Zt  int    `json:"zt"`
	Dom string `json:"dom"`
	Url string `json:"url"`
	Inf int    `json:"inf"`
}
type DownLoadUrlReq struct {
	Id      string `json:"id"`
	Quality string `json:"quality"`
	T       string `json:"_t"`
}

func downAjax(path string, data string) (string, error) {
	uuu := "https://m.lanzouy.com" + path
	method := "POST"

	m := make(map[string]interface{})
	err := json.Unmarshal([]byte(data), &m)
	if err != nil {
		return "", err
	}
	uv := url.Values{}
	for k, v := range m {
		switch vv := v.(type) {
		case int64:
			uv.Add(k, strconv.FormatInt(vv, 10))
		case string:
			uv.Add(k, vv)
		case float64:
			uv.Add(k, strconv.FormatInt(int64(vv), 10))
		}
	}
	client := &http.Client{}
	req, err := http.NewRequest(method, uuu, strings.NewReader(uv.Encode()))

	if err != nil {
		return "", err
	}
	req.Header.Add("Accept", "application/json, text/javascript, */*")
	req.Header.Add("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Add("Cache-Control", "no-cache")
	req.Header.Add("Connection", "keep-alive")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Cookie", "codelen=1; pc_ad1=1; Hm_lvt_fb7e760e987871d56396999d288238a4=1731484870; Hm_lpvt_fb7e760e987871d56396999d288238a4=1731484870; HMACCOUNT=EC959E5081FF80F7; uz_distinctid=193248a695398-0d9140bb03de21-26011951-130980-193248a69542e7; STDATA82=czst_eid%3D275618800-3821-%26ntime%3D3821; codelen=1; pc_ad1=1")
	req.Header.Add("Origin", "https://m.lanzouy.com")
	req.Header.Add("Pragma", "no-cache")
	req.Header.Add("Referer", "https://m.lanzouy.com/fn?A2VUPg5rAm9UMQNgBmQCMFY0ATxeJ1AmCjBRZlI5U2EHNlY1Cm8EZQZkUDAKZwcgV3oEZFVoAXAAblAxATNUPgNmVHoObgJhVFEDPAY4")
	req.Header.Add("Sec-Fetch-Dest", "empty")
	req.Header.Add("Sec-Fetch-Mode", "cors")
	req.Header.Add("Sec-Fetch-Site", "same-origin")
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36")
	req.Header.Add("X-Requested-With", "XMLHttpRequest")
	req.Header.Add("sec-ch-ua", "\"Chromium\";v=\"130\", \"Google Chrome\";v=\"130\", \"Not?A_Brand\";v=\"99\"")
	req.Header.Add("sec-ch-ua-mobile", "?0")
	req.Header.Add("sec-ch-ua-platform", "\"Windows\"")

	//log.Println("请求ajax", req)
	res, err := client.Do(req)
	//fmt.Println("ajax相应", res, err)
	if err != nil {
		return "", err
	}

	defer res.Body.Close()
	if res.StatusCode != 200 {
		return "", errors.New(res.Status)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	vvv := DownAjaxResp{}
	err = json.Unmarshal(body, &vvv)
	if err != nil {
		return "", err
	}
	if vvv.Zt == 0 {
		return "", fmt.Errorf("%d", vvv.Zt)
	}
	//https://down-load.lanrar.com/file/?BWMBP1tqUmMIAQE5U2ZdMVplVGwFvQaMVuAH5Fe7UOAC5QbZW7cPvwPvUYECuQK/UoBTsFGEApYH6QafVLtWsAWGAftb4FKrCNIBuFO7XdVaLVS3BdAGkla4B55XuFC2AoEG/FvgD/YD2VEpAjACb1JpUzVRKAJmB2kGbFRtVggFbAEyWztSPghnAWRTMl1qWjNUaQViBiJWNwdyVzZQYgIxBmhbNg9vA2FRNAJnAiVSeFMmUTMCMgcwBjJUOlZ4BTQBZ1spUjcIZgF4Uz9dPlpjVDcFYgYwVmMHM1dtUGUCOwYwWzMPbAMwUTICYgI3UjFTZVFoAjAHNgZkVGlWYwVkAWVbMVI1CDsBYlMpXTpabFQwBTkGIlYkB3JXblAjAmsGNVs7D2MDY1E1AmECM1I9U3BRegJpB20GZVRuVmoFNAFhWzVSNwhqAW9TMV1kWjNUYAV0BipWdwdnV2dQJgI/BmBbMQ9oA2RRMwJuAjdSP1NuUTsCJgd1BnBUf1ZqBTQBYFswUj4IaQFnUzVdbFo3VGYFfAZxVjgHcVc2UGACMgZlWygPagNjUT8CeAI2UjFTeFE9AjUHLgYmVGxWOAVyAThbWVJlCDUBalM3
	//"<a href="+dom_down+"/file/"+ date.url + lanosso +" target=_blank rel=noreferrer//><span class=txt>电信下载</span><span class='txt txtc'>联通下载</span><span class=txt>普通下载</span></a>
	return fmt.Sprintf("https://down-load.lanrar.com/file/?%s", vvv.Url), nil
}
func down(dirName string, singerName, musicName, u string) error {
	log.Println("downuuuuu", u)
	// 发送GET请求
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return err
	}
	req.Header.Add("accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Add("accept-language", "zh-CN,zh;q=0.9")
	req.Header.Add("cache-control", "no-cache")
	req.Header.Add("cookie", "down_ip=1")
	req.Header.Add("pragma", "no-cache")
	req.Header.Add("priority", "u=0, i")
	req.Header.Add("sec-ch-ua", "\"Chromium\";v=\"130\", \"Google Chrome\";v=\"130\", \"Not?A_Brand\";v=\"99\"")
	req.Header.Add("sec-ch-ua-mobile", "?0")
	req.Header.Add("sec-ch-ua-platform", "\"Windows\"")
	req.Header.Add("sec-fetch-dest", "document")
	req.Header.Add("sec-fetch-mode", "navigate")
	req.Header.Add("sec-fetch-site", "none")
	req.Header.Add("sec-fetch-user", "?1")
	req.Header.Add("upgrade-insecure-requests", "1")
	req.Header.Add("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36")
	resp, err := http.DefaultClient.Do(req)
	//log.Println(resp, err)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 检查Content-Disposition头以获取文件名
	cd := resp.Header.Get("Content-Disposition")
	var fileName string
	if cd != "" {
		_, params, err := mime.ParseMediaType(cd)
		if err == nil {
			fileName = params["filename"]
		}
	}

	// 如果没有在Content-Disposition中找到文件名，则从URL中提取
	if fileName == "" {
		ct := resp.Header.Get("Content-Type")
		switch ct {
		case "audio/mpeg":
			fileName = fmt.Sprintf("%s-%s.mp3", musicName, singerName)
		case "audio/wav":
			fileName = fmt.Sprintf("%s-%s.wav", musicName, singerName)
		case "audio/ogg", "audio/x-ogg":
			fileName = fmt.Sprintf("%s-%s.ogg", musicName, singerName)
		case "audio/acc":
			fileName = fmt.Sprintf("%s-%s.acc", musicName, singerName)
		case "audio/flac", "audio/x-flac":
			fileName = fmt.Sprintf("%s-%s.flac", musicName, singerName)
		default:
			fileName = fmt.Sprintf("%s-%s", musicName, singerName)
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	pathName := path.Join(*downPath, dirName)
	fileName = path.Join(pathName, fileName)
	os.MkdirAll(pathName, os.ModePerm)
	// 创建本地文件
	out, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer out.Close()

	// 将响应Body复制到文件中
	_, err = out.Write(body)
	if err != nil {
		return err
	}

	return nil
}

type SearchReq struct {
	SearchReqToken
	Token string `json:"token"`
}
type SearchReqToken struct {
	Type string `json:"type"`
	Text string `json:"text"`
	Page int    `json:"page"`
	V    string `json:"v"`
	T    int64  `json:"_t"`
}

func Search(title string, page int) (*SearchResp, error) {
	reqBodyObj := SearchReq{
		SearchReqToken: SearchReqToken{
			Type: "YQM",
			Text: title,
			Page: page,
			V:    "beta",
			T:    time.Now().UnixMilli(),
		},
	}
	token, err := main2(reqBodyObj.SearchReqToken)
	if err != nil {
		return nil, err
	}
	reqBodyObj.Token = token
	reqBody, err := json.Marshal(reqBodyObj)
	if err != nil {
		return nil, err
	}
	log.Println("Search", string(reqBody))

	req, err := http.NewRequest("POST", "https://api.liumingye.cn/m/api/search", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	fillReq(req)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, errors.New(res.Status)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	v := SearchResp{}
	err = json.Unmarshal(body, &v)
	if err != nil {
		log.Println(string(body))
		return nil, err
	}
	return &v, nil
}

type SearchResp struct {
	Code int `json:"code"`
	Data struct {
		List []struct {
			Id      string        `json:"id"`
			Lyric   string        `json:"lyric"`
			Name    string        `json:"name"`
			Time    int           `json:"time,omitempty"`
			Quality []interface{} `json:"quality"`
			Album   struct {
				Id   string `json:"id"`
				Name string `json:"name"`
				Pic  string `json:"pic"`
			} `json:"album,omitempty"`
			Artist []struct {
				Id   string `json:"id"`
				Name string `json:"name"`
			} `json:"artist"`
			Hash string `json:"hash,omitempty"`
			Pic  string `json:"pic,omitempty"`
		} `json:"list"`
		Total interface{} `json:"total"`
		Word  []string    `json:"word"`
	} `json:"data"`
	Msg string `json:"msg"`
}

type LoveList struct {
	Songname string `json:"songname"`
	Singer   []struct {
		Id   int    `json:"id"`
		Mid  string `json:"mid,omitempty"`
		Name string `json:"name"`
		FI   int    `json:"FI,omitempty"`
	} `json:"singer"`
}

func getFlacDuration2(seeker io.Reader) int64 {
	_, meta, err := flac2.Decode(seeker)
	if err != nil {
		log.Println("解析flac失败", err)
		return 0
	}
	log.Printf("解析信息 %+v", meta)
	return 0
}
func getFlacDuration(seeker io.Reader) int64 {
	//return getFlacDuration2(seeker)
	stream, err := flac.Parse(seeker)
	if err != nil {
		log.Println("解析flac失败", err)
		return 0
	}

	sampleRate := stream.Info.SampleRate
	totalSamples := float64(stream.Info.NSamples)
	return int64(math.Round(totalSamples / float64(sampleRate)))
}
func getOggDuration(reader io.Reader) int64 {
	// 解码OGG文件
	stream, err := oggvorbis.NewReader(reader)
	if err != nil {
		log.Println("解析失败", err)
		return 0
	}

	// 获取时长
	totalSamples := stream.Length()
	sampleRate := stream.SampleRate()

	// 计算时长（秒）
	return int64(math.Round(float64(totalSamples) / float64(sampleRate)))
}
func ext(dir string, total *int64) ([]MusicInfo, error) {
	list := make([]MusicInfo, 0)
	dirs, err := os.ReadDir(dir)
	cc := make(chan int, 3)
	if err != nil {
		return nil, err
	}
	wait := sync.WaitGroup{}
	for _, d := range dirs {
		if d.IsDir() {
			l2, err := ext(path.Join(dir, d.Name()), total)
			if err != nil {
				return nil, err
			}
			list = append(list, l2...)
		} else if strings.HasSuffix(d.Name(), ".flac") || strings.HasSuffix(d.Name(), ".mp3") || strings.HasSuffix(d.Name(), ".ogg") {
			cc <- 1
			wait.Add(1)

			d := d
			go func() {
				defer func() {
					wait.Done()
					<-cc
				}()
				f, err := os.Open(path.Join(dir, d.Name()))
				if err != nil {
					log.Println("读取文件失败", dir, d.Name(), err)
					return
				}
				dddd, _ := io.ReadAll(f)
				f.Close()

				meta, err := tag.ReadFrom(bytes.NewReader(dddd))
				if err != nil {
					log.Println("读取元信息失败", dir, d.Name(), err)
					return
				}

				duration := meta.Raw()["TLEN"]
				realDur := int64(0)
				switch ms := duration.(type) {
				case float64:
					realDur = int64(math.Round(ms / 1000))
				case float32:
					realDur = int64(math.Round(float64(ms / 1000)))
				case int64:
					realDur = int64(math.Round(float64(ms) / 1000))
				case int:
					realDur = int64(math.Round(float64(ms) / 1000))
				case int32:
					realDur = int64(math.Round(float64(ms) / 1000))
				case string:
					ddd, err := strconv.ParseInt(ms, 10, 64)
					if err != nil {
						log.Println("转换时间失败", dir, d.Name(), err)
					} else {
						realDur = int64(math.Round(float64(ddd) / 1000))
					}
				default:
					if strings.Contains(d.Name(), ".flac") {
						realDur = getFlacDuration(bytes.NewReader(dddd))
					} else if strings.Contains(d.Name(), ".ogg") {
						realDur = getOggDuration(bytes.NewReader(dddd))
					} else {
						log.Printf("未知类型 %s %+v\n", d.Name(), meta.Raw())
					}
				}
				titles := strings.Split(meta.Title(), "-")
				title := meta.Title()
				singerName := meta.Artist()
				if len(titles) == 2 {
					title = strings.TrimSpace(titles[1])
					if singerName == "" {
						singerName = strings.TrimSpace(titles[0])
					}
				}

				titles = strings.Split(d.Name(), ".")
				if len(titles) == 2 {
					titles = strings.Split(titles[0], "-")
					if len(titles) == 2 {
						if title == "" {
							title = strings.TrimSpace(titles[1])
						}
						if singerName == "" {
							singerName = strings.TrimSpace(titles[0])
						}
					}
				}
				atomic.AddInt64(total, 1)
				log.Println("已经读取", atomic.LoadInt64(total))
				list = append(list, MusicInfo{
					SongName:   title,
					SingerName: singerName,
					Path:       path.Join(dir, d.Name()),
					Duration:   realDur,
					Size:       int64(len(dddd)),
				})
			}()
		}
	}
	wait.Wait()
	return list, nil
}

type MusicInfo struct {
	SongName   string
	SingerName string
	Path       string
	Duration   int64
	Size       int64
}

func exportLove() {
	loveData, err := os.ReadFile("./love.json")
	if err != nil {
		log.Panicln(err)
	}
	loveList := make([]LoveList, 0)
	err = json.Unmarshal(loveData, &loveList)
	if err != nil {
		log.Panicln(err)
	}

	total := int64(0)
	sources, err := ext("./", &total)
	if err != nil {
		log.Panicln(err)
	}

	builder := bytes.Buffer{}
	builder.WriteString("#EXTM3U\n#PLAYLIST:我喜欢的音乐\n")
	success := 0
	for _, item := range loveList {

		matchList := make([]MusicInfo, 0)
		for _, s := range sources {
			singer := ""
			if len(item.Singer) > 0 {
				singer = item.Singer[0].Name
			}
			if strings.Contains(s.SongName, item.Songname) && strings.Contains(s.SingerName, singer) {
				matchList = append(matchList, s)

			}
		}
		sort.Slice(matchList, func(i, j int) bool {
			return matchList[i].Size > matchList[j].Size
		})
		if len(matchList) > 0 {
			builder.WriteString(fmt.Sprintf("#EXTINF:%d,%s - %s\n", matchList[0].Duration, matchList[0].SingerName, matchList[0].SongName))
			builder.WriteString(path.Join("/music", matchList[0].Path))
			builder.WriteString("\n")
			log.Println("匹配成功", item.Songname, matchList[0].Path)
			success++
		} else {
			log.Println("匹配失败", item.Songname)
		}
	}
	log.Printf("一共%d 成功%d", len(loveList), success)
	os.WriteFile("./我喜欢的音乐.m3u8", builder.Bytes(), 0755)
}

type Artist struct {
	ID       int64
	ArtistId string
	Name     string
	Pic      string
	IsFetch  int
}

func (a Artist) TableName() string {
	return "artist"
}

type Music struct {
	ID      int64
	MusicId string
	Name    string
	Pic     string
	Lyric   string
	Artist  string
	Album   string
	Time    int
	Quality string
	DownUrl string
	IsDown  int
	Path    string
}

func toJson(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		log.Println(err)
		return "{}"
	}
	return string(data)
}
func getAllMusicDownUrl(linkDb *gorm.DB) {
	musicList := make([]Music, 0)
	c := make(chan int, *num)
	linkDb.FindInBatches(&musicList, 100, func(tx *gorm.DB, batch int) error {
		wg := sync.WaitGroup{}
		for _, music := range musicList {
			if music.DownUrl != "" {
				continue
			}
			music := music
			time.Sleep(time.Second)
			c <- 1
			wg.Add(1)
			go func() {
				defer func() {
					wg.Done()
					<-c
				}()
				quality, err := getLastQuality(music.Quality)
				if err != nil {
					log.Println("解析质量失败", music.ID, err)
					return
				}
				downUrl, err := GetDownLoadUrl1(music.MusicId, quality)
				if err != nil {
					log.Println("获取下载链接失败", music.ID, err)
					return
				}
				if downUrl == "" {
					log.Println("未获取到下载链接", music.ID)
					return
				}
				music.DownUrl = downUrl
				err = tx.Save(&music).Error
				if err != nil {
					log.Println("保存下载链接失败", music, err)
				}
				return
			}()
		}
		wg.Wait()
		log.Println("3秒后继续获取下载链接")
		time.Sleep(time.Second * 3)
		return nil
	})
}
func downAllMusic(linkDb *gorm.DB) {
	musicList := make([]Music, 0)
	c := make(chan int, *num)
	linkDb.FindInBatches(&musicList, 100, func(tx *gorm.DB, batch int) error {
		wg := sync.WaitGroup{}
		for _, music := range musicList {
			if music.IsDown == 1 {
				continue
			}
			if music.DownUrl == "" {
				log.Println("没有下载链接", music.ID)
				continue
			}
			if music.DownUrl == "nocopyright" {
				log.Println("没有版权下架", music.ID)
				continue
			}
			music := music
			time.Sleep(time.Second)
			c <- 1
			wg.Add(1)
			go func() {
				defer func() {
					wg.Done()
					<-c
				}()
				dir, realSingerName := getSingerName(music.Artist)
				filePath, err := autoDown(dir, realSingerName, music.Name, music.DownUrl)
				if err != nil {
					log.Println("下载失败", music.ID, err)
				} else {
					music.IsDown = 1
					music.Path = filePath
					err = tx.Save(&music).Error
					if err != nil {
						log.Println("保存下载状态失败", music, err)
					}
				}
				return
			}()
		}
		wg.Wait()
		log.Println("5秒后继续下载")
		time.Sleep(time.Second * 5)
		return nil
	})
}

func getLastQuality(qualityStr string) (string, error) {
	quality := make([]interface{}, 0)
	err := json.Unmarshal([]byte(qualityStr), &quality)
	if err != nil {
		return "", err
	}
	if len(quality) == 0 {
		return "", errors.New("没有质量:" + qualityStr)
	}
	q := quality[len(quality)-1]
	switch qq := q.(type) {
	case int64:
		return strconv.FormatInt(qq, 10), nil
	case string:
		return qq, nil
	case map[string]interface{}:
		return qq["name"].(string), nil

	case float64:
		return strconv.FormatInt(int64(qq), 10), nil
	default:
		return "", errors.New("未识别到质量" + qualityStr)
	}
}
func getSingerName(singerStr string) (string, string) {
	m := make([]map[string]string, 0)
	err := json.Unmarshal([]byte(singerStr), &m)
	if err != nil || len(m) == 0 {
		log.Println("解析歌手失败", singerStr, err)
		return "未知", "未知"
	}

	realSingerName := ""
	for _, ar := range m {
		realSingerName += "," + ar["name"]
	}
	if len(realSingerName) > 0 {
		realSingerName = realSingerName[1:]
	}
	name := m[0]["name"]
	if name == "" {
		return "未知", "未知"
	}
	return name, realSingerName
}
func autoDown(dirName string, singerName, musicName, u string) (string, error) {
	log.Println("downuuuuu", u)

	if strings.Contains(u, "lanzouy.com") {
		realUrl, err := getLanRealDown(u)
		if err != nil {
			log.Println("获取蓝奏云真实下载链接失败", u, err)
			return "", err
		} else {
			log.Println("获取蓝奏云真实下载链接成功", realUrl)
			u = realUrl
		}
	}
	// 发送GET请求
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return "", err
	}
	if strings.Contains(u, "liumingye") {
		fillReq(req)
		req.Header.Set("accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	}
	resp, err := http.DefaultClient.Do(req)
	//log.Println(resp, err)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var fileName string
	ct := resp.Header.Get("Content-Type")
	switch ct {
	case "audio/mpeg":
		fileName = fmt.Sprintf("%s-%s.mp3", musicName, singerName)
	case "audio/wav":
		fileName = fmt.Sprintf("%s-%s.wav", musicName, singerName)
	case "audio/ogg", "audio/x-ogg":
		fileName = fmt.Sprintf("%s-%s.ogg", musicName, singerName)
	case "audio/acc":
		fileName = fmt.Sprintf("%s-%s.acc", musicName, singerName)
	case "audio/flac", "audio/x-flac":
		fileName = fmt.Sprintf("%s-%s.flac", musicName, singerName)
	default:
		log.Println("未知的音频格式", u, musicName, ct)
	}

	// 检查Content-Disposition头以获取文件名
	cd := resp.Header.Get("Content-Disposition")

	if cd != "" && fileName == "" {
		_, params, err := mime.ParseMediaType(cd)
		if err == nil {
			fileName = params["filename"]
		}
	}

	if fileName == "" {
		fileName = fmt.Sprintf("%s-%s", musicName, singerName)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if bytes.Contains(body, []byte("验证")) {
		return "", errors.New("安全验证")
	}
	pathName := path.Join(*downPath, dirName)
	fileName = path.Join(pathName, fileName)
	os.MkdirAll(pathName, os.ModePerm)
	// 创建本地文件
	out, err := os.Create(fileName)
	if err != nil {
		return "", err
	}
	defer out.Close()

	// 将响应Body复制到文件中
	_, err = out.Write(body)
	if err != nil {
		return "", err
	}

	return fileName, nil
}
func getLanRealDownFromBody(respBody []byte) (string, error) {

	if len(respBody) == 0 {
		return "", errors.New("body长度为0")
	}
	//log.Println(string(respBody))

	srcReg, err := regexp.Compile(`iframe.*src="(.*?)".*?iframe`)
	if err != nil {
		return "", err
	}
	list := srcReg.FindStringSubmatch(string(respBody))
	if len(list) != 2 {
		return "", errors.New("没有匹配地址")
	}

	log.Println("网盘下载页", fmt.Sprintf("https://m.lanzouy.com/%s", list[1]))

	downResp, err := http.Get(fmt.Sprintf("https://m.lanzouy.com/%s", list[1]))
	if err != nil {
		return "", err
	}
	defer downResp.Body.Close()
	downBody, err := io.ReadAll(downResp.Body)
	if err != nil {
		return "", err
	}
	//log.Println(string(downBody))

	uReg, err := regexp.Compile(`(/ajaxm.php.*?)'`)
	if err != nil {
		return "", err
	}
	list = uReg.FindStringSubmatch(string(downBody))
	if len(list) != 2 {
		return "", errors.New("未查到ajaxm")
	}
	ajaxUrl := list[1]
	dataReg, err := regexp.Compile(`data.*?:(.*?\})`)
	if err != nil {
		return "", err
	}
	list = dataReg.FindStringSubmatch(string(downBody))
	if len(list) != 2 {
		return "", errors.New("未查到ajaxm参数")
	}
	ajaxBody := list[1]
	ajaxBody = strings.ReplaceAll(ajaxBody, "ajaxdata", `'?ctdf'`)
	ajaxBody = strings.ReplaceAll(ajaxBody, "ciucjdsdc", `''`)
	ajaxBody = strings.ReplaceAll(ajaxBody, "aihidcms", `'7Sij'`)
	ajaxBody = strings.ReplaceAll(ajaxBody, "kdns", `1`)
	ajaxBody = strings.ReplaceAll(ajaxBody, `'`, `"`)
	//log.Println(ajaxUrl, ajaxBody)
	return downAjax(ajaxUrl, ajaxBody)

}
func getLanRealDown(pageUrl string) (string, error) {
	resp, err := http.Get(pageUrl)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return getLanRealDownFromBody(respBody)
}
