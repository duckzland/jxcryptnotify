#### Installation

Install go package, under Ubuntu :
```
sudo apt -y install golang
```

&nbsp;
&nbsp;

 
#### Build the package
```
# tested with go version 1.23.1
# Build the package
go build main.go
```
&nbsp;
&nbsp;

#### Configuring config.json

1. Get SMTP server and its configuration for : host, port, username, password
2. Edit the config_example.json and rename it config.json

*** Remember to fetch the correct UCID from CMC for the target and source token

&nbsp;
&nbsp;

#### Using Systemd to execute the program
This is the preferred method when you have the proper access to administrate systemd service. Otherwise you can use cronjob to achieve the same result.

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
&nbsp;
&nbsp;

#### Use CronJob to execute the program

Under the linux user account, as best practice.. Do Not Run this as root account!
```
crontab -e
```

Setup cron to fire every 10 minutes
```
*/10 * * * *  /path/to/the/compiled/file
```
&nbsp;
&nbsp;


#### Installing and using the configurator UI
The configurator is just a simple ui made using python3, to install:
```
apt install python3
```

To configure the ui, edit the configui_example.json and rename it configui.json

The fire up the ui:
```
python3 job_editor.py
```

