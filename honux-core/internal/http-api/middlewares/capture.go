package middlewares

import (
	"bytes"
	"net/http"
)

type captureWriter struct {
	realWriter  http.ResponseWriter
	statusCode  int
	body        bytes.Buffer
	header      http.Header
	wroteHeader bool
}

func newCaptureWriter(w http.ResponseWriter) *captureWriter {
	return &captureWriter{
		realWriter: w,
		statusCode: http.StatusOK,
		header:     make(http.Header),
	}
}

func (cw *captureWriter) Header() http.Header {
	return cw.header
}

func (cw *captureWriter) WriteHeader(statusCode int) {
	if !cw.wroteHeader {
		cw.statusCode = statusCode
		cw.wroteHeader = true
	}
}

func (cw *captureWriter) Write(b []byte) (int, error) {
	if !cw.wroteHeader {
		cw.WriteHeader(http.StatusOK)
	}
	return cw.body.Write(b)
}

func (cw *captureWriter) flushTo() {
	dest := cw.realWriter.Header()
	for key, values := range cw.header {
		for _, v := range values {
			dest.Add(key, v)
		}
	}
	cw.realWriter.WriteHeader(cw.statusCode)
	if cw.body.Len() > 0 {
		cw.realWriter.Write(cw.body.Bytes())
	}
}
