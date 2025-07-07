## Installation ##

- Install go package, under Ubuntu :
```
sudo apt -y install golang
```

- Build the package
```
# tested with go version 1.13.8
# Might need to update when using 1.19 or 2.x version
# go version go1.13.8 linux/amd64

# Build the package
go build main.go
```

- Configure config.json

1. Get SMTP server and its configuration for : host, port, username, password
2. Edit the config_example.json and rename it config.json

- [SYSTEMD] Using Systemd to fire the program instead of cronjob

1. Copy the cryptochecker_example.service to cryptochecker.service
2. Edit the cryptochecker.service to match your needs
3. Copy cryptochecker.service to /etc/systemd/system/
4. Reload systemd
```
sudo systemctl daemon-reload
```
5. Enable and start the service
```
sudo systemctl enable cryptochecker.service
sudo systemctl start cryptochecker.service
```
6. Check the status
```
sudo systemctl status cryptochecker.service
```
7. View the logs
```
sudo journalctl -u cryptochecker.service
```

- [CRONTAB] Create Cronjob to automatically fire the compiled program at x interval

Under the linux user account, as best practice.. Do Not Run this as root account!
```
crontab -e
```

Setup cron to fire every 10 minutes
```
*/10 * * * *  /path/to/the/compiled/file
```



### TODO ###
1. Proper & configurable folder path for storing JSON files
2. Daemonize the program
3. Cache the exchange data
4. Build against latest golang
