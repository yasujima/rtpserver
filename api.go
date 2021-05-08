package main

import (
	"context"
	"github.com/labstack/echo"
	"net/http"
	//	"strconv"
)

func APIStart(ctx context.Context, callback func(obj APIObj) <-chan interface{}) {

	e := echo.New()
	e.GET("/api/:id", func(c echo.Context) error {
		str := c.Param("id")
		req := GETReq{Id: str}
		ch := callback(req)
		res := <-ch
		return c.String(http.StatusOK, res.(string))
	})
	e.POST("/api/:id", func(c echo.Context) error {
		str := c.Param("id")
		req := POSTReq{Id: str}
		ch := callback(req)
		res := <-ch
		return c.String(http.StatusOK, res.(string))
	})
	e.PUT("/api/:id", func(c echo.Context) error {
		str := c.Param("id")
		req := PUTReq{Id: str}
		ch := callback(req)
		res := <-ch
		return c.String(http.StatusOK, res.(string))
	})
	e.DELETE("/api/:id", func(c echo.Context) error {
		str := c.Param("id")
		req := DELETEReq{Id: str}
		ch := callback(req)
		res := <-ch
		return c.String(http.StatusOK, res.(string))
	})

	go func() {
		e.Start(":8080")
	}()
}
