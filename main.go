package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var mode = flag.String("mode", "http", "")

type ArtistDetailReq struct {
	ArtistDetailReqToken
	Token string `json:"token"`
}
type ArtistDetailReqToken struct {
	Id string `json:"id"`
	T  int64  `json:"_t"`
}

func main() {
	/*uuu, err := GetDownLoadUrl("kOY9", "2000")
	if err != nil {
		log.Panicln(err)
	}
	err = down(uuu)
	if err != nil {
		log.Panicln(err)
	}
	return*/
	flag.Parse()
	if *mode != "client" {
		http.ListenAndServe(":8022", http.FileServer(http.Dir("./")))
		return
	}

	log.Println("5秒后开启抓取")
	time.Sleep(time.Second * 5)
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
			_, err := os.Stat("F:\\music\\" + item.Name)
			if err == nil {
				log.Println("文件夹已存在", item.Name)
				continue
			}
			log.Println("搜索歌手歌曲", item.Name)
			for i := 1; i < 20; i++ {
				searchResp, err := Search(item.Name, i)
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
				for _, music := range searchResp.Data.List {
					log.Println("一秒后继续下载")
					time.Sleep(time.Second)
					log.Println("获取歌曲下载连接", music.Name)
					if len(music.Quality) == 0 {
						log.Println("没有质量列表", music.Name)
						continue
					}
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
					err = down(item.Name, music.Name, u)

					if err != nil {
						log.Println("下载歌曲失败", err)
						continue
					}

				}
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
				err = down(item.Name, u)

				if err != nil {
					log.Println("下载歌曲失败", err)
					continue
				}
				log.Println("一秒后继续下载")
				time.Sleep(time.Second)
			}*/
		}
	}
}
func main2(d interface{}) (string, error) {
	data, err := json.Marshal(d)
	if err != nil {
		return "", err
	}
	//log.Println(string(data))
	//data = []byte(`{"id":"zP8o","_t":1731480805511}`)
	basedata := base64.StdEncoding.EncodeToString(data)

	//log.Println(basedata)
	cmd := exec.Command("node", "index3.js", basedata)
	cmd.Dir = "E:\\code\\go\\src\\github.com\\yeyudekuangxiang\\freemp3"
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
func GetDownLoadUrl(id string, quality string) (string, error) {
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
	req.Header.Add("accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Add("accept-language", "zh-CN,zh;q=0.9")
	req.Header.Add("cache-control", "no-cache")
	req.Header.Add("cookie", "__gads=ID=a272399b69834f90:T=1731397705:RT=1731397705:S=ALNI_MaqdqkKdqQlexWAwVaiXUOT7B-eCA; __gpi=UID=00000db35530df2f:T=1731397705:RT=1731397705:S=ALNI_MbRkas7oneAguteRsqRc3jzwZhGEg; __eoi=ID=495a49f5fddb9e51:T=1731397705:RT=1731397705:S=AA-AfjYg5xi01cqDAASUWzyt3zC0; FCNEC=%5B%5B%22AKsRol-vC3o3y8wHzX2AaSE_EStgtIYuhFklK-t2CovjU0xouQMaNC3I5dyVp5fZwlIoLCF6R4OeJDYU0cvPTyZPzJGXMYnqLVXxfTSYOkTWu3Qk4Ys-HTBk0v1VTl98N-qNgFdnVka6lfdcEOtvrf6Ju4Df_jBV-g%3D%3D%22%5D%5D; sl-session=irU0VeunNWdBfUeqLFkKrw==; sl-session=jgMSNcWoNWcnIeNtlmGyAw==")
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
	if err != nil {
		return "", err
	}
	if strings.Contains(resp.Header.Get("Content-Type"), "audio") {
		return uuu, nil
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
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
func down(singer string, musicName, u string) error {
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
			fileName = fmt.Sprintf("%s-%s.mp3", musicName, singer)
		case "audio/wav":
			fileName = fmt.Sprintf("%s-%s.wav", musicName, singer)
		case "audio/ogg", "audio/x-ogg":
			fileName = fmt.Sprintf("%s-%s.ogg", musicName, singer)
		case "audio/acc":
			fileName = fmt.Sprintf("%s-%s.acc", musicName, singer)
		case "audio/flac", "audio/x-flac":
			fileName = fmt.Sprintf("%s-%s.flac", musicName, singer)
		default:
			fileName = fmt.Sprintf("%s-%s", musicName, singer)
		}
	}

	fileName = fmt.Sprintf("F:\\music\\%s\\%s", singer, fileName)
	os.MkdirAll("F:\\music\\"+singer, os.ModePerm)
	// 创建本地文件
	out, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer out.Close()

	// 将响应Body复制到文件中
	_, err = io.Copy(out, resp.Body)
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
	req.Header.Add("Cookie", "sl-session=xxRRZKT5NWeIZ3Ii1tsP6Q==")

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
