import puppeteer from 'puppeteer';
import path from 'path'
import fs from 'fs'
import http from 'http'
import { fileURLToPath } from 'url';
import { dirname } from 'path';

// 获取当前文件的URL
const __filename = fileURLToPath(import.meta.url);
// 获取当前目录名
const __dirname = dirname(__filename);

const browser = await puppeteer.launch({
    args: ['--no-sandbox', '--disable-setuid-sandbox']
});


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
