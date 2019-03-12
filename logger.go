package martini

import (
	"log"
	"net/http"
	"time"
)




// 此处的 logger 和 Martini.Classic() 中的 m.Use(Logger()) 有所不同，
// 此处取出的 logger 创建于 martini.New() 中的 logger: log.New(os.Stdout, “[martini]”, 0)，
// 故会打印到标准输出，而 Martini.Classic() 中的 m.Use(Logger()) 中间件是这样定义的：

// Logger returns a middleware handler that logs the request as it goes in and the response as it goes out.
func Logger() Handler {
	return func(res http.ResponseWriter, req *http.Request, c Context, log *log.Logger) {
		start := time.Now()

		addr := req.Header.Get("X-Real-IP")
		if addr == "" {
			addr = req.Header.Get("X-Forwarded-For")
			if addr == "" {
				addr = req.RemoteAddr
			}
		}

		log.Printf("Started %s %s for %s", req.Method, req.URL.Path, addr)

		rw := res.(ResponseWriter)
		c.Next()

		log.Printf("Completed %v %s in %v\n", rw.Status(), http.StatusText(rw.Status()), time.Since(start))
	}
}
