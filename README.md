# livechan
A chat server written in Go.

### Dependancies

* go 1.4
* sqlite 3.8.9 ( previous versions don't have a certain bugfix we need )


### Installation

- Install Go.

- Install go imagick
```
sudo apt-get --no-install-recommends install libmagickwand-dev
go get github.com/gographics/imagick/imagick
```

- Install gorilla/session.
```
go get github.com/gorilla/sessions
```
- Install gorilla/websocket.
```
go get github.com/gorilla/websocket
```
- Install captcha lib
```
go get github.com/dchest/captcha
```
- Install sqlite3 drivers.
```
go get github.com/mattn/go-sqlite3
```
- Get configparser
```
go get github.com/majestrate/configparser
```
- Get the source code.
```
git clone https://github.com/majestrate/livechan.git
```
- Run the server.
```
go run *.go
```
- Open a browser and go to `localhost:18080`.

