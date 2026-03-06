package http

import "net/http"

func HanlePOSTFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	pattern = "POST " + pattern
	http.HandleFunc(pattern, handler)
}

func HanleGETFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	pattern = "GET " + pattern
	http.HandleFunc(pattern, handler)
}

func HanlePUTFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	pattern = "PUT " + pattern
	http.HandleFunc(pattern, handler)
}

func HanleDELETEFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	pattern = "DELETE " + pattern
	http.HandleFunc(pattern, handler)
}
