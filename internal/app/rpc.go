package app

// Stdio RPC bridge between the Electron main process and Bindings.
//
// Electron spawns `qail desktop-backend` and talks to it over the child's
// stdin/stdout using newline-delimited JSON. No port is opened, so nothing
// else on the machine (or a web page poking localhost) can reach it.
//
//	request  (stdin):  {"id":1,"method":"AddRepo","args":["api","git@..."]}
//	response (stdout): {"id":1,"result":null}  or  {"id":1,"error":"..."}
//	event    (stdout): {"event":"workspace:progress","data":"cloning api"}
//
// Each request runs on its own goroutine so a long clone doesn't block a
// list refresh. Responses can therefore arrive out of order; the client
// matches them by id.

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"sync"

	"github.com/ubaniak/qail/internal/config"
)

type rpcRequest struct {
	ID     uint64            `json:"id"`
	Method string            `json:"method"`
	Args   []json.RawMessage `json:"args"`
}

type rpcResponse struct {
	ID     uint64 `json:"id"`
	Result any    `json:"result"`
	Error  string `json:"error,omitempty"`
}

type rpcEvent struct {
	Event string `json:"event"`
	Data  any    `json:"data"`
}

var errorType = reflect.TypeOf((*error)(nil)).Elem()

// Server dispatches RPC requests to a Bindings by method name.
type Server struct {
	mu  sync.Mutex
	enc *json.Encoder
	b   *Bindings
}

// NewServer wires a Bindings to out: responses and events are both
// written there, one JSON document per line.
func NewServer(ctx context.Context, s config.Store, out io.Writer) *Server {
	srv := &Server{enc: json.NewEncoder(out)}
	srv.b = New(ctx, s, srv.Emit)
	return srv
}

// Emit writes an event line. It is the Emitter handed to Bindings.
func (s *Server) Emit(name string, data any) {
	s.write(rpcEvent{Event: name, Data: data})
}

func (s *Server) write(v any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// A write error means the parent went away; Serve will see EOF on
	// stdin shortly and return, so there is nothing useful to do here.
	_ = s.enc.Encode(v)
}

// Serve reads requests from in until EOF, then waits for in-flight calls
// to finish before returning.
func (s *Server) Serve(in io.Reader) error {
	var wg sync.WaitGroup
	defer wg.Wait()

	sc := bufio.NewScanner(in)
	// Script contents travel through WriteScript, so allow large lines.
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for sc.Scan() {
		var req rpcRequest
		if err := json.Unmarshal(sc.Bytes(), &req); err != nil {
			s.write(rpcResponse{Error: fmt.Sprintf("invalid request: %v", err)})
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := s.call(req.Method, req.Args)
			resp := rpcResponse{ID: req.ID, Result: result}
			if err != nil {
				resp.Error = err.Error()
			}
			s.write(resp)
		}()
	}
	return sc.Err()
}

// call invokes the exported Bindings method named method. Every bound
// method returns either error or (T, error); args are decoded
// positionally into the method's parameter types.
func (s *Server) call(method string, rawArgs []json.RawMessage) (result any, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%s: panic: %v", method, r)
		}
	}()

	m := reflect.ValueOf(s.b).MethodByName(method)
	if !m.IsValid() {
		return nil, fmt.Errorf("unknown method %q", method)
	}
	mt := m.Type()
	if mt.NumIn() != len(rawArgs) {
		return nil, fmt.Errorf("%s: want %d args, got %d", method, mt.NumIn(), len(rawArgs))
	}
	if mt.NumOut() == 0 || mt.NumOut() > 2 || mt.Out(mt.NumOut()-1) != errorType {
		return nil, fmt.Errorf("%s is not callable over RPC", method)
	}

	args := make([]reflect.Value, len(rawArgs))
	for i, raw := range rawArgs {
		v := reflect.New(mt.In(i))
		if err := json.Unmarshal(raw, v.Interface()); err != nil {
			return nil, fmt.Errorf("%s: arg %d: %w", method, i, err)
		}
		args[i] = v.Elem()
	}

	out := m.Call(args)
	if errV := out[len(out)-1]; !errV.IsNil() {
		return nil, errV.Interface().(error)
	}
	if len(out) == 2 {
		return emptyNils(out[0]).Interface(), nil
	}
	return nil, nil
}

// emptyNils returns v with every nil slice or map, at any depth, replaced
// by an empty one, so JSON carries [] / {} instead of null. The UI reads
// `.length` on list results without null checks (Wails' generated
// bindings used to do this conversion for it).
func emptyNils(v reflect.Value) reflect.Value {
	switch v.Kind() {
	case reflect.Slice:
		if v.IsNil() {
			return reflect.MakeSlice(v.Type(), 0, 0)
		}
		out := reflect.MakeSlice(v.Type(), v.Len(), v.Len())
		for i := 0; i < v.Len(); i++ {
			out.Index(i).Set(emptyNils(v.Index(i)))
		}
		return out
	case reflect.Map:
		out := reflect.MakeMapWithSize(v.Type(), v.Len())
		iter := v.MapRange()
		for iter.Next() {
			out.SetMapIndex(iter.Key(), emptyNils(iter.Value()))
		}
		return out
	case reflect.Struct:
		out := reflect.New(v.Type()).Elem()
		out.Set(v)
		for i := 0; i < v.NumField(); i++ {
			if f := out.Field(i); f.CanSet() {
				f.Set(emptyNils(f))
			}
		}
		return out
	}
	return v
}
