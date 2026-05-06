package runner

import "context"

// Recorder is a test Runner adapter. It records every Run call in order and
// replies with canned Responses (matched by call index). Use Respond to seed
// canned outputs; missing entries default to a zero Result with no error.
type Recorder struct {
	Calls     []Command
	Responses []Response
}

// Response is a canned reply for a single Run call.
type Response struct {
	Result Result
	Err    error
}

// NewRecorder returns an empty Recorder.
func NewRecorder() *Recorder { return &Recorder{} }

// Respond appends a canned response served on the next unmatched call.
func (r *Recorder) Respond(res Result, err error) *Recorder {
	r.Responses = append(r.Responses, Response{Result: res, Err: err})
	return r
}

// RespondOK is a shorthand for Respond with stdout bytes and no error.
func (r *Recorder) RespondOK(stdout []byte) *Recorder {
	return r.Respond(Result{Stdout: stdout, ExitCode: 0}, nil)
}

// Run records cmd and returns the next canned response.
func (r *Recorder) Run(_ context.Context, cmd Command) (Result, error) {
	idx := len(r.Calls)
	r.Calls = append(r.Calls, cmd)
	if idx < len(r.Responses) {
		return r.Responses[idx].Result, r.Responses[idx].Err
	}
	return Result{}, nil
}

// LastCall returns the most recent recorded Command, or panics if none.
func (r *Recorder) LastCall() Command {
	if len(r.Calls) == 0 {
		panic("recorder: LastCall with no recorded calls")
	}
	return r.Calls[len(r.Calls)-1]
}
