package martini

import (
	"github.com/codegangsta/inject"
	"net/http"
	"reflect"
)

// ReturnHandler is a service that Martini provides that is called
// when a route handler returns something. The ReturnHandler is
// responsible for writing to the ResponseWriter based on the values
// that are passed into this function.
type ReturnHandler func(Context, []reflect.Value)

func defaultReturnHandler() ReturnHandler {
	return func(ctx Context, vals []reflect.Value) {                        // vals是返回值
		rv := ctx.Get(inject.InterfaceOf((*http.ResponseWriter)(nil)))      // 从 ctx 中取出 http.ResponseWriter 类型的对象
		res := rv.Interface().(http.ResponseWriter)                         // 从reflect.Value转化为http.ResponseWriter
		var responseVal reflect.Value
		if len(vals) > 1 && vals[0].Kind() == reflect.Int {                 // 第一个返回值 vals[0] 如果是int类型就将其写到返回的http头当中
			res.WriteHeader(int(vals[0].Int()))
			responseVal = vals[1] 											// 接下来的 vals[1] 存到 responseVal
		} else if len(vals) > 0 {                                           // 如果只有一个返回值，则直接存到 responseVal
			responseVal = vals[0]
		}

	
		// 如果返回值 responseVal 是接口指针类型则解引用到其包含或者指向对象
		if canDeref(responseVal) {
			responseVal = responseVal.Elem()
		}

		// 如果返回值 responseVal 是 uint8 slice 类型，也即字节数组，即直接按字节写入到body中
		if isByteSlice(responseVal) {
			res.Write(responseVal.Bytes())
		} else {
			res.Write([]byte(responseVal.String()))
		}
	}
}

func isByteSlice(val reflect.Value) bool {
	return val.Kind() == reflect.Slice && val.Type().Elem().Kind() == reflect.Uint8
}

func canDeref(val reflect.Value) bool {
	return val.Kind() == reflect.Interface || val.Kind() == reflect.Ptr
}
