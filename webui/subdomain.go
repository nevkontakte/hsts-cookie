package webui

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/nevkontakte/hsts-cookie/config"
	"github.com/nevkontakte/hsts-cookie/cookie"
	"net/http"
	"strconv"
)

func GetToken(request *http.Request) cookie.Token {
	params := mux.Vars(request)
	token_string := params["token"]
	token, _ := strconv.ParseUint(token_string, 0, 32)
	return cookie.Token(token)
}

func SetBitHandler(response http.ResponseWriter, request *http.Request) {
	params := mux.Vars(request)

	response.Header().Add("Content-Type", "text/css")
	if params["switch"] == "on" {
		response.Header().Set("Strict-Transport-Security", "max-age="+config.CookieLifetime)
	} else {
		response.Header().Set("Strict-Transport-Security", "max-age=0")
	}

	fmt.Fprintf(response, "/* %s -> %s */", params["subdomain"], params["switch"])
}

func GetBitHandler(response http.ResponseWriter, request *http.Request) {
	//	delay_sec := 2 + (rand.Int63() % 3)
	//	println(delay_sec)
	//	time.Sleep(time.Duration(delay_sec) * time.Second)

	params := mux.Vars(request)
	subdomain := params["subdomain"]
	bit_offset, _ := strconv.ParseUint(subdomain, 32, 32)

	op := ResolveBitOp{
		token:  GetToken(request),
		offset: uint32(bit_offset),
		value:  request.TLS != nil,
		result: make(chan cookie.MaybeCookie),
	}
	resolve <- &op
	mc := <-op.result

	response.Header().Add("Content-Type", "text/css")
	if mc.Cookie != nil {
		fmt.Fprintf(response, "/* Cookie: %d */\n", mc.Cookie.Id)
		fmt.Fprintf(response, ".get {display: block}")
		fmt.Fprintf(response, ".get:after {content: \"%04X\"}", mc.Cookie.Id)
	} else if mc.Error != nil {
		http.Error(response, mc.Error.Error(), 400)
	} else {
		fmt.Fprintf(response, "/* Keep resolving... */")
	}
}
