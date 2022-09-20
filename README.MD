## Installation ##

- Install go package, under Ubuntu :
```
sudo apt -y install golang
```

- Build the package
```
go build main.go
```

- Configure config.json

1. You must retrieve proper API key from coinmarketcap.com by signing up at https://pro.coinmarketcap.com
2. Get SMTP server and its configuration for : host, port, username, password
3. Edit the config_example.json and rename it config.json


- Create Cronjob to automatically fire the compiled program at x interval

Under the linux user account
```
crontab -e [username]
```

Setup cron to fire every 10 minutes (Free coinmarketcap account has limit for 300 hits perday)
```
*/10 * * * *  /path/to/the/compiled/file
```

