// Package martini is a powerful package for quickly writing modular web applications/services in Golang.
//
// For a full guide visit http://github.com/go-martini/martini
//
//  package main
//
//  import "github.com/go-martini/martini"
//
//  func main() {
//    m := martini.Classic()
//
//    m.Get("/", func() string {
//      return "Hello world!"
//    })
//
//    m.Run()
//  }
package martini

import (
	"log"
	"net/http"
	"os"
	"reflect"

	"github.com/codegangsta/inject"
)

// Martini represents the top level web application. inject.Injector methods can be invoked to map services on a global level.
type Martini struct {
	inject.Injector         //注入工具，利用反射实现函数注入 
	handlers []Handler 		//存储所有中间件
	action   Handler 		//路由匹配以及路由处理，在所有中间件都处理完之后执行
	logger   *log.Logger   	//日志工具
}




// New creates a bare bones Martini instance. Use this method if you want to have full control over the middleware that is used.
// 基础骨架：具备基本的注入与反射调用功能
func New() *Martini {
	m := &Martini{Injector: inject.New(), action: func() {}, logger: log.New(os.Stdout, "[martini] ", 0)}
	m.Map(m.logger)				  //标准输出的logger
	m.Map(defaultReturnHandler()) //type ReturnHandler func(Context, []reflect.Value)，调用c.Next()陷入下一个中间件
	return m
}


// Handlers sets the entire middleware stack with the given Handlers. This will clear any current middleware handlers.
// Will panic if any of the handlers is not a callable function
// 设置所有的中间件
func (m *Martini) Handlers(handlers ...Handler) {
	m.handlers = make([]Handler, 0)
	for _, handler := range handlers {
		m.Use(handler)
	}
}

// Action sets the handler that will be called after all the middleware has been invoked. This is set to martini.Router in a martini.Classic().
// 设置真正的路由处理器，所有中间件执行完之后才会执行
func (m *Martini) Action(handler Handler) {
	validateHandler(handler)
	m.action = handler
}

// Logger sets the logger
func (m *Martini) Logger(logger *log.Logger) {
	m.logger = logger
	m.Map(m.logger)
}

// Use adds a middleware Handler to the stack. Will panic if the handler is not a callable func. Middleware Handlers are invoked in the order that they are added.
// 添加一个中间件处理器，每一个http请求都会先执行，按照添加的顺序依次执行
func (m *Martini) Use(handler Handler) {
	validateHandler(handler)
	m.handlers = append(m.handlers, handler)
}

// ServeHTTP is the HTTP Entry point for a Martini instance. Useful if you want to control your own HTTP server.
// http接口，每一次http请求的用户级别处理的入口，会由 http.ListenAndServe(addr, inet) 回调调用。
func (m *Martini) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	m.createContext(res, req).run() // 每一个请求创建一个上下文，保存一些必要的信息，之后开始处理请求
}

// Run the http server on a given host and port.
// http 服务器启动
func (m *Martini) RunOnAddr(addr string) {
	// TODO: Should probably be implemented using a new instance of http.Server in place of
	// calling http.ListenAndServer directly, so that it could be stored in the martini struct for later use.
	// This would also allow to improve testing when a custom host and port are passed.

	// 此处的 logger 和 Martini.Classic() 中的 m.Use(Logger()) 有所不同，
	// 此处取出的 logger 创建于 martini.New() 中的 logger: log.New(os.Stdout, “[martini]”, 0)，
	// 故会打印到标准输出，而 Martini.Classic() 中的 m.Use(Logger()) 是一个中间件。
	logger := m.Injector.Get(reflect.TypeOf(m.logger)).Interface().(*log.Logger)
	logger.Printf("listening on %s (%s)\n", addr, Env)
	logger.Fatalln(http.ListenAndServe(addr, m))  // m是整个框架控制的核心，实现了 ServeHTTP 函数接口
}

// Run the http server. Listening on os.GetEnv("PORT") or 3000 by default.
func (m *Martini) Run() {
	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = "3000"
	}

	host := os.Getenv("HOST")

	m.RunOnAddr(host + ":" + port)
}



// 创建一个请求的上下文，与大部分的web框架一样，使用上下文的方式存储处理请求过程中的相关数据。
func (m *Martini) createContext(res http.ResponseWriter, req *http.Request) *context {
	// NewResponseWriter 对res进行了封装修饰，添加了一些其他功能，比如过滤器之类的。
	c := &context{inject.New(), m.handlers, m.action, NewResponseWriter(res), 0}
	c.SetParent(m)
	c.MapTo(c, (*Context)(nil))                      // Context 为接口类型，c 是实现了 Context 接口的具体类型结构体，以实现 接口类型 和 具体对象 的关联注入
	c.MapTo(c.rw, (*http.ResponseWriter)(nil))       // http.ResponseWrite 同样为接口类型，c.rw 是实现了该接口的具体类型结构体，这里也做一种映射
	c.Map(req) 										 // http.Request 是一种具体类型，这里则可以直接存储 req，无需做类型映射
	return c
}

// ClassicMartini represents a Martini with some reasonable defaults. Embeds the router functions for convenience.
// 经典的搭配，整合了路由以及martini核心功能
type ClassicMartini struct {
	*Martini
	Router // 匿名变量类型，需要一个实现了所有的接口的对象，这样 ClassicMartini 实例可以无缝调用 Router 的接口，比如 m.Get(pattern, handler)
}

// Classic creates a classic Martini with some basic default middleware - martini.Logger, martini.Recovery and martini.Static.
// Classic also maps martini.Routes as a service.
func Classic() *ClassicMartini {
	r := NewRouter()                 // 基础路由器，用于存储用户自定义路由规则以及处理器
	m := New()                       // 新建martini基础框架
	m.Use(Logger())                  // 注册logger中间件，请求前后打印日志，需要类型有 res http.ResponseWriter, req *http.Request, c Context, log *log.Logger，调用c.Next()陷入下一个中间件。
	m.Use(Recovery())                // 注册recover中间件，从各种panic中恢复回来并设置返回头和body
	m.Use(Static("public"))          // 注册Static中间件，支持静态文件服务，执行完之后不陷入c.Next()，貌似是直接返回的，然后执行下个handle。
	m.MapTo(r, (*Routes)(nil))       // Injector的Mapto方法，实现类型和对象的关联注入，nil 表示这里只需要一个类型
	m.Action(r.Handle)				 // 所有 router 中间件执行完才执行的 action 处理，相当于是正式的路由匹配处理，中间件做一些 web 常规必要的处理。
	return &ClassicMartini{m, r}     // 返回一个ClassMartini实例，继承了Martini的结构，提升了martini以及Router的相关方法
}

// Handler can be any callable function. Martini attempts to inject services into the handler's argument list.
// Martini will panic if an argument could not be fullfilled via dependency injection.
// 定义Handler类型为一个泛型
type Handler interface{}

// 检查Handler是否为函数类型
func validateHandler(handler Handler) {
	if reflect.TypeOf(handler).Kind() != reflect.Func {
		panic("martini handler must be a callable func")
	}
}

// Context represents a request context. Services can be mapped on the request level from this interface.
type Context interface {

	// 包含了另一个接口类型的所有接口，Context的实例必须实现所有的接口，或者包含一个匿名的具体事例实现该所有接口。
	inject.Injector

	// Next is an optional function that Middleware Handlers can call to yield the until after
	// the other Handlers have been executed. This works really well for any operations that must
	// happen after an http request

	// 中间件的顺序执行过程中按顺序执行时，使用Next接口不断的更新索引指向下一个中间件
	Next()

	// Written returns whether or not the response for this context has been written.
	// 返回是否 http 请求已经处理完并发送应答的标识
	Written() bool
}



// http请求处理的上下文实例
type context struct {
	// 匿名包含一个接口类型，初始化的时候需要一个具体实现的实例
	inject.Injector 
	// handler数组，处理http请求时按顺序一个一个执行	
	handlers []Handler
	// 其实就是最后一个handler
	action   Handler
	// 对http.ResponseWriter的进一步封装，加入更多功能，比如过滤器、Before After等处理
	rw       ResponseWriter
	// 表示当前第n个hanlder的索引
	index    int
}




// 取出当前第n个处理器，如果索引值到达最大值，则返回action函数，即开始路由匹配逻辑
func (c *context) handler() Handler {
	if c.index < len(c.handlers) {
		return c.handlers[c.index]
	}
	if c.index == len(c.handlers) {
		return c.action
	}
	panic("invalid index for context handler")
}

// 更新指向下一个处理器，之后继续执行剩余处理器对请求的处理
func (c *context) Next() {
	c.index += 1
	c.run()
}

// 判断是否已发送应答，若已发送，则不需要再进行处理
func (c *context) Written() bool {
	return c.rw.Written()
}

func (c *context) run() {
	// 循环调用，直到有 handler/action 的返回 error 引发 panic，或者有往 ResponseWriter() 输出结果的，则结束循环，直接返回。
	for c.index <= len(c.handlers) {  
		_, err := c.Invoke(c.handler())     // c.Invoke 对当前 c.handler() 函数进行回调，函数参数此前已由 injector 注入，返回值存储在 c 中。
		if err != nil {
			panic(err)
		}
		c.index += 1 						// for 循环先通过 c.Invoke() 反射调用处理函数，再更新索引，因此与 c.Next() 中的更新索引 index 并不冲突。
		if c.Written() {
			return
		}
	}
}
