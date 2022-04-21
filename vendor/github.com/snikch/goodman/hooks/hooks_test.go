package hooks

import (
	"testing"

	trans "github.com/snikch/goodman/transaction"
)

func TestNewHooks(t *testing.T) {
	hooks := NewHooks()

	if len(hooks.beforeEach) != 0 {
		t.Errorf("New hooks should have empty beforeEach hooks")
	}

	if len(hooks.beforeAll) != 0 {
		t.Errorf("New hooks should have empty beforeAll hooks")
	}

	if len(hooks.beforeEachValidation) != 0 {
		t.Errorf("New hooks should have empty beforeEachValidation hooks")
	}

	if len(hooks.beforeValidation) != 0 {
		t.Errorf("New hooks should have empty beforeValidation hooks")
	}

	if len(hooks.before) != 0 {
		t.Errorf("New hooks should have empty before hooks")
	}

	if len(hooks.afterEach) != 0 {
		t.Errorf("New hooks should have empty afterEach hooks")
	}

	if len(hooks.afterAll) != 0 {
		t.Errorf("New hooks should have empty afterAll hooks")
	}
}

func TestRunnerImplementation(t *testing.T) {
	name := "name"
	tss := trans.Transaction{
		Name: name,
	}
	var hooks *Hooks
	var invoked bool
	reply := []*trans.Transaction{}
	cbs := []Callback{
		func(ts *trans.Transaction) {
			invoked = true
		},
	}
	allCbs := []AllCallback{
		func(ts []*trans.Transaction) {
			invoked = true
		},
	}
	fns := []func(){
		func() {
			hooks = &Hooks{
				beforeAll: allCbs,
			}
			NewHooksRunner(hooks).RunBeforeAll([]*trans.Transaction{&tss}, &reply)
		},
		func() {
			hooks = &Hooks{
				beforeEach: cbs,
			}
			NewHooksRunner(hooks).RunBeforeEach(tss, &tss)
		},
		func() {
			hooks = &Hooks{
				beforeEachValidation: cbs,
			}
			NewHooksRunner(hooks).RunBeforeEachValidation(tss, &tss)
		},
		func() {
			before := map[string][]Callback{
				name: cbs,
			}
			hooks = &Hooks{
				before: before,
			}
			NewHooksRunner(hooks).RunBefore(tss, &tss)
		},
		func() {
			beforeValidation := map[string][]Callback{
				name: cbs,
			}
			hooks = &Hooks{
				beforeValidation: beforeValidation,
			}
			NewHooksRunner(hooks).RunBeforeValidation(tss, &tss)
		},
		func() {
			after := map[string][]Callback{
				name: cbs,
			}
			hooks = &Hooks{
				after: after,
			}
			NewHooksRunner(hooks).RunAfter(tss, &tss)
		},
		func() {
			hooks = &Hooks{
				afterEach: cbs,
			}
			NewHooksRunner(hooks).RunAfterEach(tss, &tss)
		},
		func() {
			hooks = &Hooks{
				afterAll: allCbs,
			}
			NewHooksRunner(hooks).RunAfterAll([]*trans.Transaction{&tss}, &reply)
		},
	}
	for _, hookFn := range fns {
		invoked = false
		hookFn()
		if !invoked {
			t.Errorf("Callback was never invoked %#v", hookFn)
		}
	}
}

func TestDefiningHooks(t *testing.T) {
	hooks := NewHooks()
	name := "name"
	cb := func(ts *trans.Transaction) {}
	allCb := func(ts []*trans.Transaction) {}

	hooks.BeforeAll(allCb)
	hooks.BeforeEach(cb)
	hooks.Before(name, cb)
	hooks.BeforeEachValidation(cb)
	hooks.BeforeValidation(name, cb)
	hooks.AfterAll(allCb)
	hooks.AfterEach(cb)
	hooks.After(name, cb)

	if len(hooks.beforeAll) != 1 {
		t.Errorf("should have one callback")
	}

	if len(hooks.beforeEach) != 1 {
		t.Errorf("should have one callback")
	}

	if len(hooks.before[name]) != 1 {
		t.Errorf("should have one callback")
	}

	if len(hooks.beforeEachValidation) != 1 {
		t.Errorf("should have one callback")
	}

	if len(hooks.beforeValidation[name]) != 1 {
		t.Errorf("should have one callback")
	}

	if len(hooks.afterAll) != 1 {
		t.Errorf("should have one callback")
	}

	if len(hooks.afterEach) != 1 {
		t.Errorf("should have one callback")
	}

	if len(hooks.after[name]) != 1 {
		t.Errorf("should have one callback")
	}
}
