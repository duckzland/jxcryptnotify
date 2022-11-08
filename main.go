package main

import (
    "encoding/json"
    "bytes"
    "os"
    "io"
    "fmt"
    "time"
    "log"
    "log/syslog"
    "strings"
    "strconv"
    "net/smtp"
    "net/http"
    "net/url"
    "crypto/tls"
    "io/ioutil"
)

/**
 * Defining Struct for config json
 * @type {String}
 */
type ConfigType struct {
    Servers ServerConfigType `json:"servers"`
    Jobs [] JobConfigType `json:"jobs"`
}

type ServerConfigType struct {
    Email EmailConfigType `json:"email"`
    Endpoint EndpointConfigType `json:"endpoint"`
    Syslog bool `json:"syslog"`
    Delay int64 `json:"delay"`
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

type EndpointConfigType struct {
    DataEndpoint string `json:"data_endpoint"`
    ExchangeEndpoint string `json:"exchange_endpoint"`
}

/**
 * Defining struct for Endpoint cryptos.json
 * @type {String}
 */
type CryptosType struct {
    Values []CryptosValuesType `json:"values"`
}

type CryptosValuesType struct {
    Id int64
    Name string
    Symbol string
}


/**
 * Defining struct for endpoint exhange data
 * @type {[type]}
 */
type ExchangeDataType struct {
    SourceSymbol string
    SourceId int64
    SourceAmount float64
    TargetSymbol string
    TargetId int64
    TargetAmount float64
}

/**
 * Global variables
 */
var Config ConfigType
var Cryptos CryptosType


/**
 * Custom UnmarshalJSON for crypto.json
 */
func (cp *CryptosValuesType) UnmarshalJSON(data []byte) error {
    var v []interface{}
    err := json.Unmarshal(data, &v)
    if err != nil {
        log.Fatal(err)
        return err
    }

    cp.Id = int64(v[0].(float64))
    cp.Name = v[1].(string)
    cp.Symbol = v[2].(string)

    return nil
}


/**
 * Custom UnmarshalJSON for CMC Exchange sjon
 */
func (ex *ExchangeDataType) UnmarshalJSON(data []byte) error {


    var v map[string]interface{}
    err := json.Unmarshal(data, &v)
    if err != nil {
        log.Fatal(err)
        return err
    }

    sc := v["data"]
    tc := v["data"].(map[string]interface{})["quote"].([]interface{})[0]

    // CMC Json data is weird the the id is in string while cryptoId is in int64 (but golang cast this as float64)
    ex.SourceSymbol =  sc.(map[string]interface{})["symbol"].(string)
    ex.SourceId, _ = strconv.ParseInt(sc.(map[string]interface{})["id"].(string), 10, 64)
    ex.SourceAmount =  sc.(map[string]interface{})["amount"].(float64)

    ex.TargetSymbol = tc.(map[string]interface{})["symbol"].(string)
    ex.TargetId = int64(tc.(map[string]interface{})["cryptoId"].(float64))
    ex.TargetAmount = tc.(map[string]interface{})["price"].(float64)

    return nil
}


/**
 * Load Configuration Json into memory
 */
func loadConfig() {

    b := bytes.NewBuffer(nil)
    f, _ := os.Open("config.json")
    io.Copy(b, f)
    f.Close()

    err := json.Unmarshal(b.Bytes(), &Config)

    if err != nil {
        log.Fatal(err)
    } else {
        log.Print("Configuration Loaded")
    }
}


/**
 * Load CMC Crypto.json to memory
 */
func loadCryptos() {
    b := bytes.NewBuffer(nil)
    f, _ := os.Open("cryptos.json")
    io.Copy(b, f)
    f.Close()

    err := json.Unmarshal(b.Bytes(), &Cryptos)

    if err != nil {
        log.Fatal(err)
    } else {
        log.Print("Cryptos Loaded")
    }
}


/**
 * Function for sending SMTP Email
 */
func sendEmail(recipient string, subject string, message string) {

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

    log.Print(fmt.Sprintf("Sent Email to %s", recipient))
}


/**
 * Loading cryptos.json from CMC
 */
func getTickerData() string {

    C := Config.Servers.Endpoint

    url := C.DataEndpoint
    client := &http.Client{}
    req, err := http.NewRequest("GET", url, nil)

    if err != nil {
        log.Fatal(err)
    }

    resp, err := client.Do(req);

    if err != nil {
        log.Fatal(err)
    }

    respBody, err := ioutil.ReadAll(resp.Body)

    if err != nil {
        log.Fatal(err)
    } else {
        log.Print("Fetched cryptodata from CMC")
    }

    return string(respBody)
}


/**
 * Get the exchange data from CMC
 */
func getExchangeData(Job JobConfigType) string {

    C := Config.Servers.Endpoint

    client := &http.Client{}
    req, err := http.NewRequest("GET", C.ExchangeEndpoint, nil)

    if err != nil {
        log.Fatal(err)
    }

    sc := convertCryptoIdFromSymbol(Job.SourceCoin)
    if sc == 0 {
        log.Fatal("Failed to get proper crypto id")
    }

    tc := convertCryptoIdFromSymbol(Job.TargetCoin)
    if tc == 0 {
        log.Fatal("Failed to get proper crypto id")
    }

    q := url.Values{}
    q.Add("amount", Job.SourceValue)
    q.Add("id", strconv.FormatInt(sc, 10))
    q.Add("convert_id", strconv.FormatInt(tc, 10))

    // req.Header.Set("Accepts", "application/json")
    // req.Header.Add("X-CMC_PRO_API_KEY", C.Key)
    req.URL.RawQuery = q.Encode()

    //fmt.Printf("%s", req);

    resp, err := client.Do(req);
    if err != nil {
        log.Fatal(err)
    } else {
        log.Print("Fetched exchange data from CMC")
    }

    // @todo better error reporting!
    // fmt.Println(resp.Status);
    respBody, _ := ioutil.ReadAll(resp.Body)

    // fmt.Println(string(respBody));

    return string(respBody)
}


/**
 * Examine the exhange data
 */
func examineData(JsonData string, Job JobConfigType) {

    var Exchange ExchangeDataType
    err := json.Unmarshal([]byte(JsonData), &Exchange)

    if err != nil {
        log.Fatal(err)
    }

    if (strings.ToLower(Exchange.SourceSymbol) == strings.ToLower(Job.SourceCoin)) &&
        (strings.ToLower(Exchange.TargetSymbol) == strings.ToLower(Job.TargetCoin)) {

        sourceValue, _ := strconv.ParseFloat(Job.TargetValue, 64)
        targetValue := Exchange.TargetAmount

        subject := fmt.Sprintf("Monitored Target Price for %s %s %s Reached", Job.SourceCoin, Job.Comparison, Job.TargetCoin)
        message := fmt.Sprintf("Current conversion from %s:%s is %s:%f, which has reached the configured target of %s:%f %s %s:%f",
            Job.SourceCoin,
            Job.SourceValue,
            Job.TargetCoin,
            targetValue,
            Job.TargetCoin,
            sourceValue,
            Job.Comparison,
            Job.TargetCoin,
            targetValue,
        );

        if (Job.Comparison == ">" && sourceValue < targetValue) ||
            (Job.Comparison == "<" && sourceValue > targetValue) ||
            (Job.Comparison == "=" && sourceValue == targetValue) {

            log.Print(message)
            sendEmail(Job.Email, subject, message)

        } else {

            log.Print(fmt.Sprintf("Monitored Target Price for %s %s %s not reached yet",
                Job.SourceCoin,
                Job.Comparison,
                Job.TargetCoin,
            ))
        }
    }
}


/**
 * Helper function for creating file
 */
func createFile(fileName string, textString string) {
    out, err := os.Create(fileName)

    if err != nil {
        log.Fatalln(err)
    }

    defer out.Close()

    _, err2 := out.WriteString(textString)

    if err2 != nil {
        log.Fatal(err2)
    } else {
        log.Printf("Creating new file %s", fileName)
    }

    out.Sync()
    out.Close()

}


/**
 * Helper function for checking if file exists
 */
func fileExists(path string) (bool, error) {
    _, err := os.Stat(path)
    if err == nil {
        return true, nil
    }

    if os.IsNotExist(err) {
        return false, nil
    }

    return false, err
}


/**
 * Helper function for converting crypto symbol to CMC crypto id
 */
func convertCryptoIdFromSymbol(symbol string) int64 {
    s := strings.ToLower(symbol)

    for _, crypto := range Cryptos.Values {
        if strings.ToLower(crypto.Symbol) == s {
            return crypto.Id
        }
    }

    return 0
}


/**
 * Main Function
 * @type {[type]}
 */
func main() {

    loadConfig()

    if Config.Servers.Syslog == true {
        syslogger, err := syslog.New(syslog.LOG_INFO, "jxcryptonotify")
        if err != nil {
            log.Fatalln(err)
        }

        log.SetOutput(syslogger)
    }

    exists, err := fileExists("cryptos.json")
    if exists == false {
        cryptos := getTickerData()
        createFile("cryptos.json", cryptos)
    }

    if err != nil {
        log.Fatalln(err)
    }

    loadCryptos()

    // We are using free API, be considerate and pause 10 seconds between each call!
    duration := time.Duration(Config.Servers.Delay) * time.Second

    for _, c := range Config.Jobs {

        d := getExchangeData(c)
        examineData(d, c)

        time.Sleep(duration)
    }
}
