package main

import (
    "encoding/json"
    "bytes"
    "os"
    "io"
    "fmt"
    "log"
    "strings"
    "strconv"
    "net/smtp"
    "net/http"
    "net/url"
    "crypto/tls"
    "io/ioutil"
    "github.com/tidwall/gjson"
)

var Config ConfigType

type ConfigType struct {
    Servers ServerConfigType `json:"servers"`
    Jobs [] JobConfigType `json:"jobs"`
}

type ServerConfigType struct {
    Email EmailConfigType `json:"email"`
    Coinmarketcap CoinmarketcapConfigType `json:"coinmarketcap"`
}

type EmailConfigType struct {
    From string `json:"from"`
    Subject string `json:"subject"`
    Host string `json:"host"`
    Port string `json:"port"`
    Username string `json:"username"`
    Password string `json:"password"`
}

type JobConfigType struct {
    Email string `json:"email"`
    SourceCoin string `json:"source_coin"`
    TargetCoin string `json:"target_coin"`
    SourceValue string `json:"source_value"`
    TargetValue string `json:"target_value"`
    Comparison string `json:"comparison"`
}

type CoinmarketcapConfigType struct {
    Key string `json:"key"`
    Endpoint string `json:"endpoint"`
}


func LoadConfig() {

    b := bytes.NewBuffer(nil)
    f, _ := os.Open("config.json")
    io.Copy(b, f)
    f.Close()

    err := json.Unmarshal(b.Bytes(), &Config)

    if err != nil {
        log.Fatal(err)
    }
}


func SendEmail(recipient string, subject string, message string) {

    C := Config.Servers.Email;

    from := C.From
    host := C.Host
    port := C.Port
    user := C.Username
    pass := C.Password

    headers := make(map[string]string)
    headers["From"] = from
    headers["To"] = recipient
    headers["Subject"] = subject
    headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/plain; charset=\"utf-8\""


    body := ""
    for k,v := range headers {
        body += fmt.Sprintf("%s: %s\r\n", k, v)
    }
    body += "\r\n" + message

    // Preparing server
    auth := smtp.PlainAuth(user, user, pass, host)

    // TLS config
    tlsconfig := &tls.Config {
        InsecureSkipVerify: true,
        ServerName: host,
    }

    // Here is the key, you need to call tls.Dial instead of smtp.Dial
    // for smtp servers running on 465 that require an ssl connection
    // from the very beginning (no starttls)
    conn, err := tls.Dial("tcp", host + ":" + port, tlsconfig)
    if err != nil {
        log.Fatal(err)
    }

    // Failed to build client
    c, err := smtp.NewClient(conn, host)
    if err != nil {
        log.Fatal(err)
    }

    // Failed Auth
    if err = c.Auth(auth); err != nil {
        log.Fatal(err)
    }

    // To && From
    if err = c.Mail(from); err != nil {
        log.Fatal(err)
    }

    if err = c.Rcpt(recipient); err != nil {
        log.Fatal(err)
    }

    // Data
    w, err := c.Data()
    if err != nil {
        log.Fatal(err)
    }

    // Failed to write message
    _, err = w.Write([]byte(body))
    if err != nil {
        log.Fatal(err)
    }

    // Failed to close connection
    err = w.Close()
    if err != nil {
        log.Fatal(err)
    }

    c.Quit()

    fmt.Sprintf("Sent Email to %s", recipient)
}


func GetData(Job JobConfigType) string {

    C := Config.Servers.Coinmarketcap;

    client := &http.Client{}
    req, err := http.NewRequest("GET", C.Endpoint, nil)

    if err != nil {
        log.Fatal(err)
    }

    q := url.Values{}
    q.Add("amount", Job.SourceValue)
    q.Add("symbol", Job.SourceCoin)
    q.Add("convert", Job.TargetCoin)

    req.Header.Set("Accepts", "application/json")
    req.Header.Add("X-CMC_PRO_API_KEY", C.Key)
    req.URL.RawQuery = q.Encode()

    resp, err := client.Do(req);
    if err != nil {
        log.Fatal(err)
    }

    // @todo better error reporting!
    // fmt.Println(resp.Status);
    respBody, _ := ioutil.ReadAll(resp.Body)

    //fmt.Println(string(respBody));

    return string(respBody)
}


func ExamineData(JsonData string, Job JobConfigType) {

    gjson.Get(JsonData, "data" ).ForEach(func(_, data gjson.Result) bool {

        raw := data.String()
        sourceCoin := gjson.Get(raw, "symbol").String()

        if strings.ToLower(sourceCoin) != strings.ToLower(Job.SourceCoin) {
            return true
        }

        gjson.Get(raw, "quote").ForEach(func(targetCoin, rate gjson.Result) bool {

            if strings.ToLower(targetCoin.String()) != strings.ToLower(Job.TargetCoin) {
                return true
            }

            sourceValue, err := strconv.ParseFloat(Job.SourceValue, 64)
            if err != nil {
                log.Fatal(err)
            }

            targetValue, err := strconv.ParseFloat(gjson.Get(rate.String(), "price").String(), 64)
            if err != nil {
                log.Fatal(err)
            }

            subject := fmt.Sprintf("Monitored Target Price for %s %s %s Reached", Job.SourceCoin, Job.Comparison, Job.TargetCoin)
            message := fmt.Sprintf("Current price for %s is %f, configured detection to detect %s:%f %s %s:%f reached", Job.TargetCoin, targetValue, Job.SourceCoin, sourceValue, Job.Comparison, Job.TargetCoin, targetValue);

            switch Job.Comparison {
                case ">":
                    if sourceValue > targetValue {
                        SendEmail(Job.Email, subject, message)
                    }
                case "<":
                    if sourceValue < targetValue {
                        SendEmail(Job.Email, subject, message)
                    }
                case "=":
                    if sourceValue == targetValue {
                        SendEmail(Job.Email, subject, message)
                    }
            }

            return false
        })

        return false
    })
}


func main() {
    LoadConfig()

    for _, c := range Config.Jobs {
        d := GetData(c)
        ExamineData(d, c)
    }
}
