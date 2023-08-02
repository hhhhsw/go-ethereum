package native

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"math/big"
	"sync/atomic"
)

func init() {
	// This is how Geth will become aware of the tracer and register it under a given name
	tracers.DefaultDirectory.Register("opcounter", newOpcounter, false)
}

type opcounter struct {
	env       *vm.EVM
	counts    map[string]int // Store opcode counts
	interrupt uint32         // Atomic flag to signal execution interruption
	reason    error          // Textual reason for the interruption
}

func newOpcounter(ctx *tracers.Context, cfg json.RawMessage) (tracers.Tracer, error) {
	return &opcounter{counts: make(map[string]int)}, nil
}

// CaptureStart implements the EVMLogger interface to initialize the tracing operation.
func (t *opcounter) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	t.env = env
}

// CaptureState implements the EVMLogger interface to trace a single step of VM execution.
func (t *opcounter) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	// Skip if tracing was interrupted
	if atomic.LoadUint32(&t.interrupt) > 0 {
		t.env.Cancel()
		return
	}

	name := op.String()
	if _, ok := t.counts[name]; !ok {
		t.counts[name] = 0
	}
	t.counts[name]++
}

// CaptureEnter is called when EVM enters a new scope (via call, create or selfdestruct).
func (t *opcounter) CaptureEnter(op vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
}

// CaptureExit is called when EVM exits a scope, even if the scope didn't
// execute any code.
func (t *opcounter) CaptureExit(output []byte, gasUsed uint64, err error) {}

// CaptureFault implements the EVMLogger interface to trace an execution fault.
func (t *opcounter) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error) {
}

// CaptureEnd is called after the call finishes to finalize the tracing.
func (t *opcounter) CaptureEnd(output []byte, gasUsed uint64, err error) {}

func (*opcounter) CaptureTxStart(gasLimit uint64) {}

func (*opcounter) CaptureTxEnd(restGas uint64) {}

// GetResult returns the json-encoded nested list of call traces, and any
// error arising from the encoding or forceful termination (via `Stop`).
func (t *opcounter) GetResult() (json.RawMessage, error) {
	res, err := json.Marshal(t.counts)
	if err != nil {
		return nil, err
	}
	return res, t.reason
}

// Stop terminates execution of the tracer at the first opportune moment.
func (t *opcounter) Stop(err error) {
	t.reason = err
	atomic.StoreUint32(&t.interrupt, 1)
}
