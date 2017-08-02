/*
Copyright 2017 Google Inc. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package jsonnet

// readyValue
// -------------------------------------

// readyValue is a wrapper which allows to use a concrete value where normally
// some evaluation would be expected (e.g. object fields). It's not part
// of the value interface for increased type safety (it would be very easy
// to "overevaluate" otherwise) and conveniently it also saves us from implementing
// these methods for all value types.
type readyValue struct {
	content value
}

func (rv *readyValue) getValue(i *interpreter, t *TraceElement) (value, error) {
	return rv.content, nil
}

func (rv *readyValue) bindToObject(sb selfBinding, origBinding bindingFrame) potentialValue {
	return rv
}

// potentialValues
// -------------------------------------

// thunk holds code and environment in which the code is supposed to be evaluated
type thunk struct {
	name identifier
	env  environment
	body astNode
}

func makeThunk(name identifier, env environment, body astNode) *cachedThunk {
	return makeCachedThunk(&thunk{
		name: name,
		env:  env,
		body: body,
	})
}

func (t *thunk) getValue(i *interpreter, trace *TraceElement) (value, error) {
	context := TraceContext{
		Name: "thunk <" + string(t.name) + ">",
	}
	return i.EvalInCleanEnv(trace, &context, &t.env, t.body)
}

// callThunk represents a concrete, but not yet evaluated call to a function
type callThunk struct {
	function evalCallable
	args     callArguments
}

func makeCallThunk(ec evalCallable, args callArguments) *callThunk {
	return &callThunk{function: ec, args: args}
}

func (th *callThunk) getValue(i *interpreter, trace *TraceElement) (value, error) {
	evaluator := makeEvaluator(i, trace)
	// TODO(sbarzowski): actually this trace is kinda useless inside...
	return th.function.EvalCall(th.args, evaluator)
}

// cachedThunk is a wrapper that caches the value of a potentialValue after
// the first evaluation.
// Note: All potentialValues are required to provide the same value every time,
// so it's only there for efficiency.
// TODO(sbarzowski) better name?
// TODO(sbarzowski) force use cached/ready everywhere? perhaps an interface tag?
// TODO(sbarzowski) investigate efficiency of various representations
type cachedThunk struct {
	pv potentialValue
}

func makeCachedThunk(pv potentialValue) *cachedThunk {
	return &cachedThunk{pv}
}

func (t *cachedThunk) getValue(i *interpreter, trace *TraceElement) (value, error) {
	v, err := t.pv.getValue(i, trace)
	if err != nil {
		// TODO(sbarzowski) perhaps cache errors as well
		// may be necessary if we allow handling them in any way
		return nil, err
	}
	t.pv = &readyValue{v}
	return v, nil
}

// errorThunk can be used when potentialValue is expected, but we already
// know that something went wrong
type errorThunk struct {
	err error
}

func (th *errorThunk) getValue(i *interpreter, trace *TraceElement) (value, error) {
	return nil, th.err
}

func makeErrorThunk(err error) *errorThunk {
	return &errorThunk{err}
}

// unboundFields
// -------------------------------------

type codeUnboundField struct {
	body astNode
}

func (f *codeUnboundField) bindToObject(sb selfBinding, origBinding bindingFrame) potentialValue {
	// TODO(sbarzowski) better object names (perhaps include a field name too?)
	return makeThunk("object_field", makeEnvironment(origBinding, sb), f.body)
}

// evalCallables
// -------------------------------------

type closure struct {
	// base environment of a closure
	// arguments should be added to it, before executing it
	env      environment
	function *astFunction
}

func (closure *closure) EvalCall(arguments callArguments, e *evaluator) (value, error) {
	argThunks := make(bindingFrame)
	for i, arg := range arguments.positional {
		argThunks[closure.function.parameters[i]] = arg
	}

	calledEnvironment := makeEnvironment(
		addBindings(closure.env.upValues, argThunks),
		closure.env.sb,
	)
	// TODO(sbarzowski) better function names
	context := TraceContext{
		Name: "function <anonymous>",
	}
	return e.evalInCleanEnv(&context, &calledEnvironment, closure.function.body)
}

func (closure *closure) Parameters() identifiers {
	return closure.function.parameters
}

func makeClosure(env environment, function *astFunction) *closure {
	return &closure{
		env:      env,
		function: function,
	}
}
