import puppeteer from 'puppeteer';
import path from 'path'
import fs from 'fs'
import http from 'http'
import { fileURLToPath } from 'url';
import { dirname } from 'path';
import qqMusic from 'qq-music-api'

// 获取当前文件的URL
const __filename = fileURLToPath(import.meta.url);
// 获取当前目录名
const __dirname = dirname(__filename);

const browser = await puppeteer.launch({
    args: ['--no-sandbox', '--disable-setuid-sandbox']
});
(async ()=>{
    await loadQQLove()
})()

// 创建服务器
const server = http.createServer((req, res) => {
    if (req.url.includes(".js") || req.url.includes(".html")) {
        // 如果请求的URL是/index.html，返回本地文件index.html
        let file  = req.url.substring(req.url.indexOf("/")).split("?")[0]
        const filePath = path.join(__dirname,file);
        fs.readFile(filePath, (err, content) => {
            if (err) {
                res.writeHead(500, { 'Content-Type': 'text/plain' });
                res.end('Server Error');
            } else {
                if (req.url.includes(".js")){
                    res.writeHead(200,{'Content-Type':'application/javascript'})
                }else{
                    res.writeHead(200, { 'Content-Type': 'text/html' });
                }
                res.end(content);
            }
        });
    } else if (req.url.includes("/encode") ) {
       let paths = req.url.split("?");
       let data = paths[paths.length-1];
       loadPage(data).then((d)=>{
           res.writeHead(200, { 'Content-Type': 'text/plain' });
           res.end(d);
       }).catch(e=>{
           res.writeHead(500, { 'Content-Type': 'text/plain' });
           console.log("解密异常",data,e)
           res.end("解密异常");
       })
    } else {
        // 对于其他请求，返回404
        res.writeHead(404, { 'Content-Type': 'text/plain' });
        res.end('Not Found');
    }
});
async function loadPage(data){
    const page = await browser.newPage();
   /* page.on('console', msg => {
        // 打印消息类型和值
        console.log(`Console message: ${msg.type()}: ${msg.text()}`);
    });*/
    await page.goto("http://127.0.0.1:3002/index.html?"+data, { waitUntil: 'networkidle0' });
    const result = await page.evaluate(() => {
        return document.body.innerText;
    });
    await page.close();
    return result
}

// 在脚本退出时关闭浏览器
const closeBrowser = async () => {
    if (browser) {
        console.log('Browser closed.');
        await browser.close();
    }
    if (server){
        server.close()
    }
};

// 捕获进程的退出事件
process.on('exit', closeBrowser);
process.on('SIGINT', closeBrowser);  // 捕获 Ctrl+C
process.on('SIGTERM', closeBrowser); // 捕获 kill 命令


// 监听端口3000
server.listen(3002, () => {
    console.log('Server is running on http://localhost:3002');
});


async function loadQQLove() {
    qqMusic.setCookie("pgv_pvid=8787804069; fqm_pvqid=db8961dc-2778-4a94-83c0-2633ca65f9d1; fqm_sessionid=ccf227f6-4cbf-4ee7-af67-df683ba10812; pgv_info=ssid=s3760339968; ts_refer=www.baidu.com/link; ts_uid=1868027922; _qpsvr_localtk=0.41167863332374344; RK=JXOdH3uCku; ptcz=69f44cf532b78d6685136a19401732e4a07b7c2a7027c32c447cb34036425506; login_type=1; psrf_qqaccess_token=45A23A53CB61AC3AD9EF443D979A1ADD; wxrefresh_token=; tmeLoginType=2; music_ignore_pskey=202306271436Hn@vBj; wxunionid=; wxopenid=; euin=ow-A7wns7e4Aon**; uin=2226064520; qm_keyst=Q_H_L_63k3NEnnkiSavGPC-veWPHtq6SvT-nfQG22LAazuv4-MMECI4lzdajjTEfSmghZ8qfe7q7ETdK1KK-HAmcK7arEBeVyo; psrf_qqopenid=C15BE4CF59E3E3F57378EEAAF1EEC268; psrf_access_token_expiresAt=1733119027; psrf_musickey_createtime=1732514227; psrf_qqrefresh_token=5E626A5FB31F2684B1F60D6B376D0A1F; qqmusic_key=Q_H_L_63k3NEnnkiSavGPC-veWPHtq6SvT-nfQG22LAazuv4-MMECI4lzdajjTEfSmghZ8qfe7q7ETdK1KK-HAmcK7arEBeVyo; psrf_qqunionid=9DCC8C39FFC7B73BBD513006122A9FE9; ts_last=y.qq.com/n/ryqq/profile")
    let resp =await qqMusic.api("/songlist",{id:611561185})
    fs.writeFile("./resp.json",JSON.stringify(resp),(err)=>{})
    let list = []
    for(let i=0;i<resp.songlist.length;i++){
        let song = resp.songlist[i]
        list.push({
            "songname":song.songname,
            "singer":song.singer
        })
    }
    let str = JSON.stringify(list)
    fs.writeFile("./love.json",str, (err)=>{})
}

