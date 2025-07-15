package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/syslog"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

/**
 * Defining Struct for config json
 */
type ConfigType struct {
	Servers ServerConfigType `json:"servers"`
	Jobs    []JobConfigType  `json:"jobs"`
}

type ServerConfigType struct {
	Email    EmailConfigType    `json:"email"`
	Endpoint EndpointConfigType `json:"endpoint"`
	Syslog   bool               `json:"syslog"`
	Delay    int64              `json:"delay"`
	MaxEmail int64              `json:"maximum_email_sent"`
}

type EmailConfigType struct {
	Enable   bool   `json:"enable"`
	From     string `json:"from"`
	Subject  string `json:"subject"`
	Host     string `json:"host"`
	Port     string `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type JobConfigType struct {
	Email       string  `json:"email"`
	SourceCoin  string  `json:"source_coin"`
	TargetCoin  string  `json:"target_coin"`
	SourceValue float64 `json:"source_value"`
	TargetValue float64 `json:"target_value"`
	Comparison  string  `json:"comparison"`
	EmailCount  int64   `json:"email_sent_count"`
}

type EndpointConfigType struct {
	DataEndpoint     string `json:"data_endpoint"`
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
	Id     int64
	Name   string
	Symbol string
}

/**
 * Defining struct for endpoint exhange data
 * @type {[type]}
 */
type ExchangeDataType struct {
	SourceSymbol string
	SourceId     int64
	SourceAmount float64
	TargetSymbol string
	TargetId     int64
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
	ex.SourceSymbol = sc.(map[string]interface{})["symbol"].(string)
	ex.SourceId, _ = strconv.ParseInt(sc.(map[string]interface{})["id"].(string), 10, 64)
	ex.SourceAmount = sc.(map[string]interface{})["amount"].(float64)

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
		wrappedErr := fmt.Errorf("Failed to load config.json: %w", err)
		log.Fatal(wrappedErr)
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
		wrappedErr := fmt.Errorf("Failed to load cryptos.json: %w", err)
		log.Fatal(wrappedErr)
	} else {
		log.Print("Cryptos Loaded")
	}
}

/**
 * Function for sending SMTP Email
 */
func sendEmail(recipient string, subject string, message string) {

	C := Config.Servers.Email

	if C.Enable != true {
		return
	}

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
	for k, v := range headers {
		body += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	body += "\r\n" + message

	// Preparing server
	auth := smtp.PlainAuth(user, user, pass, host)

	// TLS config
	tlsconfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         host,
	}

	// Here is the key, you need to call tls.Dial instead of smtp.Dial
	// for smtp servers running on 465 that require an ssl connection
	// from the very beginning (no starttls)
	conn, err := tls.Dial("tcp", host+":"+port, tlsconfig)
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

	// Failed Recipient
	if err = c.Rcpt(recipient); err != nil {
		log.Fatal(err)
	}

	// Failed Data
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
 * Send Email using local email server (exim4)
 */
func localSendEmail(recipient string, subject string, message string) {
	C := Config.Servers.Email

	if C.Enable != true {
		return
	}

	// Sender and recipient details
	from := C.From
	to := []string{recipient}
	msg := []byte("From: " + C.From + "\r\n" + "To: " + recipient + "\r\n" + "Subject: " + subject + "\r\n" + "\r\n" + message)
	srv := C.Host + ":" + C.Port

	// Send the email
	err := smtp.SendMail(srv, nil, from, to, msg)

	if err != nil {
		wrappedErr := fmt.Errorf("Failed to send email using local server: %w", err)
		log.Fatal(wrappedErr)
	}

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

	resp, err := client.Do(req)

	if err != nil {
		log.Fatal(err)
	}

	respBody, err := io.ReadAll(resp.Body)

	if err != nil {
		wrappedErr := fmt.Errorf("Failed to fetched cryptodata from CMC: %w", err)
		log.Fatal(wrappedErr)
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
	q.Add("amount", strconv.FormatFloat(Job.SourceValue, 'f', 4, 64))
	q.Add("id", strconv.FormatInt(sc, 10))
	q.Add("convert_id", strconv.FormatInt(tc, 10))

	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		wrappedErr := fmt.Errorf("Failed to fetch exchange data from CMC: %w", err)
		log.Fatal(wrappedErr)
	} else {
		log.Print("Fetched exchange data from CMC")
	}

	respBody, _ := io.ReadAll(resp.Body)

	// @todo better error reporting, check the response http status
	// and CMC error status as well
	// fmt.Println(resp.Status);
	// fmt.Println(string(respBody));

	return string(respBody)
}

/**
 * Function for extracting number of decimals from a float
 */
func NumDecPlaces(v float64) int {
	s := strconv.FormatFloat(v, 'f', -1, 64)
	i := strings.IndexByte(s, '.')
	if i > -1 {
		return len(s) - i - 1
	}
	return 0
}

/**
 * Examine the exhange data
 */
func examineData(JsonData string, Job JobConfigType) int {

	var Exchange ExchangeDataType
	err := json.Unmarshal([]byte(JsonData), &Exchange)

	if err != nil {
		wrappedErr := fmt.Errorf("Failed to examine data: %w", err)
		log.Fatal(wrappedErr)
	}

	if (strings.ToLower(Exchange.SourceSymbol) == strings.ToLower(Job.SourceCoin)) &&
		(strings.ToLower(Exchange.TargetSymbol) == strings.ToLower(Job.TargetCoin)) {

		// Decimal formatting for text
		svd := NumDecPlaces(Job.SourceValue)
		tvd := NumDecPlaces(Job.TargetValue)

		svs := fmt.Sprintf("%s%d%s", "%.", svd, "f")
		tvs := fmt.Sprintf("%s%d%s", "%.", tvd, "f")

		svt := fmt.Sprintf(svs, Job.SourceValue)
		tvt := fmt.Sprintf(tvs, Job.TargetValue)
		evt := fmt.Sprintf(tvs, Exchange.TargetAmount)

		subject := fmt.Sprintf("Monitored Target Price for %s %s %s Reached", Job.SourceCoin, Job.Comparison, Job.TargetCoin)
		message := fmt.Sprintf("Current conversion rate of %s %s is %s %s, which has reached the configured target of %s %s %s %s %s",
			svt,
			Job.SourceCoin,
			evt,
			Job.TargetCoin,
			svt,
			Job.SourceCoin,
			Job.Comparison,
			tvt, Job.TargetCoin,
		)

		// Debug
		// println(fmt.Sprintf("Target: %s, Exchange: %s", tvt, evt))

		if (Job.Comparison == ">" && Job.TargetValue < Exchange.TargetAmount) ||
			(Job.Comparison == "<" && Job.TargetValue > Exchange.TargetAmount) ||
			(Job.Comparison == "=" && tvt == evt) {

			log.Print(message)
			C := Config.Servers.Email

			if C.Host == "localhost" {
				localSendEmail(Job.Email, subject, message)

			} else {
				sendEmail(Job.Email, subject, message)
			}

			return 1

		} else {
			log.Print(fmt.Sprintf("Current conversion rate of %s %s is %s %s, has not reached the configured target of %s %s %s %s %s yet",
				svt,
				Job.SourceCoin,
				evt,
				Job.TargetCoin,
				svt,
				Job.SourceCoin,
				Job.Comparison,
				tvt,
				Job.TargetCoin,
			))
		}
	}

	return 0
}

/**
 * Helper function for creating file
 */
func createFile(fileName string, textString string) {
	out, err := os.Create(fileName)

	if err != nil {
		wrappedErr := fmt.Errorf("Failed to create file: %w", err)
		log.Fatal(wrappedErr)
	}

	defer out.Close()

	_, err2 := out.WriteString(textString)

	if err2 != nil {
		wrappedErr2 := fmt.Errorf("Failed to write file: %w", err2)
		log.Fatal(wrappedErr2)
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
 */
func main() {

	loadConfig()

	if Config.Servers.Syslog == true {
		syslogger, err := syslog.New(syslog.LOG_INFO, "jxcryptonotify")
		if err != nil {
			wrappedErr := fmt.Errorf("Failed to enable syslog for logging: %w", err)
			log.Fatalln(wrappedErr)
		}

		log.SetOutput(syslogger)
	}

	// Fetching cryptos.json from local or CMC
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

	// Processing Jobs
	for i, c := range Config.Jobs {

		// Email sent counter limit reached, dont process further
		if Config.Servers.MaxEmail > 0 && Config.Jobs[i].EmailCount >= Config.Servers.MaxEmail {
			log.Print(fmt.Sprintf("Not Monitoring Job #%d for %s %s %s due to maximum email sent limit reached.",
				i,
				c.SourceCoin,
				c.Comparison,
				c.TargetCoin,
			))

			continue
		}

		// Process job
		d := getExchangeData(c)
		x := examineData(d, c)

		// Email sent successfully, increase the limit counter and update the config.json
		if x == 1 && Config.Servers.MaxEmail > 0 {

			// Updating email count
			Config.Jobs[i].EmailCount++

			// Attempt to update config.json
			var buf bytes.Buffer
			encoder := json.NewEncoder(&buf)
			encoder.SetEscapeHTML(false)
			encoder.SetIndent("", "  ")

			err = encoder.Encode(Config)
			if err != nil {
				fmt.Println("Error encoding config.json data:", err)
			} else {
				err = os.WriteFile("config.json", buf.Bytes(), 0644)
				if err != nil {
					fmt.Println("Error writing config.json:", err)
				}
			}
		}

		time.Sleep(duration)
	}
}
