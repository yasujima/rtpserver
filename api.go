package main

import (
	"context"
	"github.com/labstack/echo"
	"net/http"
)

func APIStart(ctx context.Context) <-chan interface{} {

	reqchan := make(chan interface{})	

	e := echo.New()
	e.GET("/ping", func(c echo.Context) error {
		ch := make(chan interface{})
		defer close(ch)

		req := GETReq{Id:0}
		obj := ReqObj{Ch:ch, AnyReq:req}
		
		reqchan <- obj
		res := <-ch
		return c.String(http.StatusOK, res.(string))
	})
	
	go func() {
		defer close(reqchan)
		e.Start(":8080")
	}()
	return reqchan
}
