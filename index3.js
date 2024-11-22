import puppeteer from 'puppeteer';

(async () => {

    const browser = await puppeteer.launch({
        args: ['--no-sandbox', '--disable-setuid-sandbox']
    });
    const page = await browser.newPage();
    //console.log("http://127.0.0.1:8022?"+process.argv[2])
    await page.goto("http://127.0.0.1:8022?"+process.argv[2], { waitUntil: 'networkidle0' });
    const result = await page.evaluate(() => {
        return document.body.innerText;
    });
    console.log(result);
    await browser.close();
})();