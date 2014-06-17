package main

import (
	"code.google.com/p/freetype-go/freetype"
	"code.google.com/p/freetype-go/freetype/raster"
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

var (
	KEEP  int
	CHARS int
	FONT  string
	ADDR  string
	TTL   time.Duration

	CTX *freetype.Context
	ENC *base64.Encoding
	DB  *Database
	SRV *http.Server

	SIZE image.Rectangle
	BG   image.Image
	FG   image.Image
	PT   raster.Point
)

type Database struct {
	Map   map[int]*image.Image
	Token int
	Mtx   sync.RWMutex
}

func (d *Database) Save(img *image.Image) (token int) {
	d.Mtx.Lock()
	token = d.Token
	d.Map[token] = img
	d.Token = d.Token + 1
	d.Mtx.Unlock()

	go func(d *Database, token int) {
		time.Sleep(TTL)
		d.Mtx.Lock()
		delete(d.Map, token)
		d.Mtx.Unlock()

		return
	}(d, token)

	return
}

func (d *Database) Get(token int) (img *image.Image, ok bool) {
	d.Mtx.RLock()
	img, ok = d.Map[token]
	d.Mtx.RUnlock()

	return
}

func (d *Database) PeriodicClean() {
	for {
		time.Sleep(time.Duration((KEEP + 1)) * TTL)
		d.Mtx.Lock()
		if token := d.Token; token >= KEEP {
			d.Token = 0
		}
		d.Mtx.Unlock()
	}

	return
}

func genImage() (s string, i image.Image, err error) {
	b := make([]byte, CHARS)
	rand.Read(b)
	s = ENC.EncodeToString(b)

	img := image.NewGray(SIZE)
	draw.Draw(img, SIZE, BG, image.ZP, draw.Src)
	CTX.SetDst(img)
	_, err = CTX.DrawString(s, PT)

	return s, img, err
}

func setupFreetype() {
	SIZE = image.Rect(0, 0, 300, 100)
	BG = image.White
	FG = image.Black

	fontBytes, err := ioutil.ReadFile(FONT)
	if err != nil {
		panic(err)
	}

	font, err := freetype.ParseFont(fontBytes)
	if err != nil {
		panic(err)
	}

	CTX = freetype.NewContext()
	CTX.SetFont(font)
	CTX.SetFontSize(64)
	CTX.SetClip(SIZE)
	CTX.SetSrc(FG)
	PT = freetype.Pt(2, 2+int(CTX.PointToFix32(64)>>8))
}

func genHandler(res http.ResponseWriter, req *http.Request) {
	ans, img, err := genImage()
	if err != nil {
		log.Println(err)
		http.Error(res, "Internal error", 500)
	} else {
		token := DB.Save(&img)
		fmt.Fprintf(res, "{\"t\":%d,\"a\":\"%s\"}\n", token, ans)
	}

	return
}

func imageHandler(res http.ResponseWriter, req *http.Request) {
	var err error

	err = req.ParseForm()
	if err == nil {
		arg, ok := req.Form["id"]
		if (ok) && (len(arg) == 1) {
			token, err := strconv.Atoi(arg[0])
			if err == nil {
				img, ok := DB.Get(token)
				if ok {
					err = png.Encode(res, *img)
					if err == nil {
						return
					}
				}
			}
		}
	}

	if err != nil {
		log.Println(err)
	}
	http.NotFound(res, req)

	return
}

func setupServer() {
	SRV = &http.Server{
		Addr: ADDR,
	}

	http.HandleFunc("/gen", genHandler)
	http.HandleFunc("/image", imageHandler)
}

func main() {
	flag.StringVar(&ADDR, "a", "127.0.0.1:9923", "bind address")
	flag.StringVar(&FONT, "f", "./sample.ttf", "font file path")
	flag.IntVar(&KEEP, "k", 100, "db cleanup threshold")
	flag.IntVar(&CHARS, "c", 6, "random bytes number")
	ttl := flag.Int("t", 30, "ttl for images (seconds)")
	flag.Parse()
	TTL = time.Duration(*ttl) * time.Second

	DB = &Database{
		Token: 0,
		Map:   make(map[int]*image.Image),
	}
	go DB.PeriodicClean()
	ENC = base64.StdEncoding

	setupFreetype()
	setupServer()

	log.Fatal(SRV.ListenAndServe())
}
