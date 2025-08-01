// Copyright (c) 2013-2018 The btcsuite developers
// Copyright (c) 2015-2018 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package txscript

import (
	"fmt"
	"math/big"

	"github.com/gcash/bchd/bchec"
	"github.com/gcash/bchd/wire"
)

// ScriptFlags is a bitmask defining additional operations or tests that will be
// done when executing a script pair.
type ScriptFlags uint32

const (
	// ScriptBip16 defines whether the bip16 threshold has passed and thus
	// pay-to-script hash transactions will be fully validated.
	ScriptBip16 ScriptFlags = 1 << iota

	// ScriptStrictMultiSig defines whether to verify the stack item
	// used by CHECKMULTISIG is zero length.
	ScriptStrictMultiSig

	// ScriptDiscourageUpgradableNops defines whether to verify that
	// NOP1 through NOP10 are reserved for future soft-fork upgrades.  This
	// flag must not be used for consensus critical code nor applied to
	// blocks as this flag is only for stricter standard transaction
	// checks.  This flag is only applied when the above opcodes are
	// executed.
	ScriptDiscourageUpgradableNops

	// ScriptVerifyCheckLockTimeVerify defines whether to verify that
	// a transaction output is spendable based on the locktime.
	// This is BIP0065.
	ScriptVerifyCheckLockTimeVerify

	// ScriptVerifyCheckSequenceVerify defines whether to allow execution
	// pathways of a script to be restricted based on the age of the output
	// being spent.  This is BIP0112.
	ScriptVerifyCheckSequenceVerify

	// ScriptVerifyCleanStack defines that the stack must contain only
	// one stack element after evaluation and that the element must be
	// true if interpreted as a boolean.  This is rule 6 of BIP0062.
	// This flag should never be used without the ScriptBip16 flag nor the
	// ScriptVerifyWitness flag.
	ScriptVerifyCleanStack

	// ScriptVerifyDERSignatures defines that signatures are required
	// to compily with the DER format.
	ScriptVerifyDERSignatures

	// ScriptVerifyLowS defines that signtures are required to comply with
	// the DER format and whose S value is <= order / 2.  This is rule 5
	// of BIP0062.
	ScriptVerifyLowS

	// ScriptVerifyMinimalData defines that signatures must use the smallest
	// push operator. This is both rules 3 and 4 of BIP0062.
	ScriptVerifyMinimalData

	// ScriptVerifyMinimalIf requires the argument of OP_IF/NOTIF to be exactly
	// 0x01 or empty vector
	ScriptVerifyMinimalIf

	// ScriptVerifyNullFail defines that signatures must be empty if
	// a CHECKSIG or CHECKMULTISIG operation fails.
	ScriptVerifyNullFail

	// ScriptVerifySigPushOnly defines that signature scripts must contain
	// only pushed data.  This is rule 2 of BIP0062.
	ScriptVerifySigPushOnly

	// ScriptVerifyStrictEncoding defines that signature scripts and
	// public keys must follow the strict encoding requirements.
	ScriptVerifyStrictEncoding

	// ScriptVerifyCompressedPubkey defines that public keys must be in the
	// compressed format.
	ScriptVerifyCompressedPubkey

	// ScriptVerifyBip143SigHash defines that signature hashes should
	// be calculated using the bip0143 signature hashing algorithm.
	ScriptVerifyBip143SigHash

	// ScriptVerifyCheckDataSig enables verification of the OP_CHECKDATASIG
	// OP_CHECKDATASIGVERIFY opcodes. Without this flag the opcodes will
	// behave as if they are disabled.
	ScriptVerifyCheckDataSig

	// ScriptVerifySchnorr enables verification of schnorr signatures,
	// in addition to ECDSA signatures, in OP_CHECKSIG, OP_CHECKSIGVEERIFY,
	// OP_CHECKDATASIG, and OP_CHECKDATASIGVERIFY.
	ScriptVerifySchnorr

	// ScriptVerifyAllowSegwitRecovery enables an exemption to
	// ScriptVerifyCleanStack which allows certain outputs to be exempted
	// from the rule in order to allow users who accidentally sent funds to
	// segwit addresses to recover them.
	ScriptVerifyAllowSegwitRecovery

	// ScriptVerifySchnorrMultisig enables the use of schnorr signatures
	// with OP_CHECKMULTISIG. When active the dummy element signals the
	// use of schnorr or ECDSA.
	ScriptVerifySchnorrMultisig

	// ScriptReportSigChecks is used to signal that the sig checks reported
	// by the vm need to be checked against the consensus rules.
	ScriptReportSigChecks

	// ScriptVerifyInputSigChecks verifies that the sig check density in
	// the input script is less than the max.
	ScriptVerifyInputSigChecks

	// ScriptVerifyReverseBytes enables the use of OP_REVERSEBYTES in the
	// script.
	ScriptVerifyReverseBytes
)

// HasFlag returns whether the ScriptFlags has the passed flag set.
func (scriptFlags ScriptFlags) HasFlag(flag ScriptFlags) bool {
	return scriptFlags&flag == flag
}

const (
	// MaxStackSize is the maximum combined height of stack and alt stack
	// during execution.
	MaxStackSize = 1000

	// MaxScriptSize is the maximum allowed length of a raw script.
	MaxScriptSize = 10000
)

// halforder is used to tame ECDSA malleability (see BIP0062).
var halfOrder = new(big.Int).Rsh(bchec.S256().N, 1)

// Engine is the virtual machine that executes scripts.
type Engine struct {
	scripts         [][]parsedOpcode
	scriptIdx       int
	scriptOff       int
	lastCodeSep     int
	dstack          stack // data stack
	astack          stack // alt stack
	tx              wire.MsgTx
	txIdx           int
	condStack       []int
	numOps          int
	flags           ScriptFlags
	sigCache        *SigCache
	hashCache       *TxSigHashes
	bip16           bool     // treat execution as pay-to-script-hash
	savedFirstStack [][]byte // stack from first script for bip16 scripts
	inputAmount     int64
	sigChecks       int
}

// hasFlag returns whether the script engine instance has the passed flag set.
func (vm *Engine) hasFlag(flag ScriptFlags) bool {
	return vm.flags.HasFlag(flag)
}

// IsBranchExecuting returns whether or not the current conditional branch is
// actively executing. For example, when the data stack has an OP_FALSE on it
// and an OP_IF is encountered, the branch is inactive until an OP_ELSE or
// OP_ENDIF is encountered.  It properly handles nested conditionals.
func (vm *Engine) IsBranchExecuting() bool {
	if len(vm.condStack) == 0 {
		return true
	}
	return vm.condStack[len(vm.condStack)-1] == OpCondTrue
}

// executeOpcode performs execution on the passed opcode. It takes into account
// whether or not it is hidden by conditionals, but some rules still must be
// tested in this case.
func (vm *Engine) executeOpcode(pop *parsedOpcode) error {
	// Disabled opcodes are fail on program counter.
	if pop.isDisabled() {
		str := fmt.Sprintf("attempt to execute disabled opcode %s",
			pop.opcode.name)
		return scriptError(ErrDisabledOpcode, str)
	}

	// Always-illegal opcodes are fail on program counter.
	if pop.alwaysIllegal() {
		str := fmt.Sprintf("attempt to execute reserved opcode %s",
			pop.opcode.name)
		return scriptError(ErrReservedOpcode, str)
	}

	// Note that this includes OP_RESERVED which counts as a push operation.
	if pop.opcode.value > OP_16 {
		vm.numOps++
		if vm.numOps > MaxOpsPerScript {
			str := fmt.Sprintf("exceeded max operation limit of %d",
				MaxOpsPerScript)
			return scriptError(ErrTooManyOperations, str)
		}

	} else if len(pop.data) > MaxScriptElementSize {
		str := fmt.Sprintf("element size %d exceeds max allowed size %d",
			len(pop.data), MaxScriptElementSize)
		return scriptError(ErrElementTooBig, str)
	}

	// Nothing left to do when this is not a conditional opcode and it is
	// not in an executing branch.
	if !vm.IsBranchExecuting() && !pop.isConditional() {
		return nil
	}

	// Ensure all executed data push opcodes use the minimal encoding when
	// the minimal data verification flag is set.
	if vm.dstack.verifyMinimalData && vm.IsBranchExecuting() &&
		pop.opcode.value >= 0 && pop.opcode.value <= OP_PUSHDATA4 {
		if err := pop.checkMinimalDataPush(); err != nil {
			return err
		}
	}

	return pop.opcode.opfunc(pop, vm)
}

// disasm is a helper function to produce the output for DisasmPC and
// DisasmScript.  It produces the opcode prefixed by the program counter at the
// provided position in the script.  It does no error checking and leaves that
// to the caller to provide a valid offset.
func (vm *Engine) disasm(scriptIdx, scriptOff int) string {
	return fmt.Sprintf("%02x:%04x: %s", scriptIdx, scriptOff,
		vm.scripts[scriptIdx][scriptOff].print(false))
}

// validPC returns an error if the current script position is valid for
// execution, nil otherwise.
func (vm *Engine) validPC() error {
	if vm.scriptIdx >= len(vm.scripts) {
		str := fmt.Sprintf("past input scripts %v:%v %v:xxxx",
			vm.scriptIdx, vm.scriptOff, len(vm.scripts))
		return scriptError(ErrInvalidProgramCounter, str)
	}
	if vm.scriptOff >= len(vm.scripts[vm.scriptIdx]) {
		str := fmt.Sprintf("past input scripts %v:%v %v:%04d",
			vm.scriptIdx, vm.scriptOff, vm.scriptIdx,
			len(vm.scripts[vm.scriptIdx]))
		return scriptError(ErrInvalidProgramCounter, str)
	}
	return nil
}

// curPC returns either the current script and offset, or an error if the
// position isn't valid.
func (vm *Engine) curPC() (script, off int, err error) {
	err = vm.validPC()
	if err != nil {
		return 0, 0, err
	}
	return vm.scriptIdx, vm.scriptOff, nil
}

// DisasmPC returns the string for the disassembly of the opcode that will be
// next to execute when Step() is called.
func (vm *Engine) DisasmPC() (string, error) {
	scriptIdx, scriptOff, err := vm.curPC()
	if err != nil {
		return "", err
	}
	return vm.disasm(scriptIdx, scriptOff), nil
}

// DisasmScript returns the disassembly string for the script at the requested
// offset index.  Index 0 is the signature script and 1 is the public key
// script.
func (vm *Engine) DisasmScript(idx int) (string, error) {
	if idx >= len(vm.scripts) {
		str := fmt.Sprintf("script index %d >= total scripts %d", idx,
			len(vm.scripts))
		return "", scriptError(ErrInvalidIndex, str)
	}

	var disstr string
	for i := range vm.scripts[idx] {
		disstr = disstr + vm.disasm(idx, i) + "\n"
	}
	return disstr, nil
}

// SigChecks returns the number of SigChecks accumulated during
// execution. Note this will return the number at the specific
// step the engine is currently on. If you want the total number
// of SigChecks this must be called after script execution has
// finished.
func (vm *Engine) SigChecks() int {
	return vm.sigChecks
}

// CheckErrorCondition returns nil if the running script has ended and was
// successful, leaving a a true boolean on the stack.  An error otherwise,
// including if the script has not finished.
func (vm *Engine) CheckErrorCondition(finalScript bool) error {
	// Check execution is actually done.  When pc is past the end of script
	// array there are no more scripts to run.
	if vm.scriptIdx < len(vm.scripts) {
		return scriptError(ErrScriptUnfinished,
			"error check when script unfinished")
	}

	// If the script is a segwit exemption we exit before checking the cleanstack
	// rule and before popping the top stack item off the stack and verifying that
	// it is non-zero. In other words, if this is a p2sh segwit script and the
	// redeem script hashed to the correct hash in the PKScript then we consider
	// the script valid. Even if the stop stack item is zero or there is more
	// than one item on the stack.
	if finalScript && vm.isSegwitExemption() {
		return nil
	}

	if finalScript && vm.hasFlag(ScriptVerifyCleanStack) && vm.dstack.Depth() != 1 {
		str := fmt.Sprintf("stack contains %d unexpected items",
			vm.dstack.Depth()-1)
		return scriptError(ErrCleanStack, str)
	} else if vm.dstack.Depth() < 1 {
		return scriptError(ErrEmptyStack,
			"stack empty at end of script execution")
	}

	if vm.hasFlag(ScriptVerifyInputSigChecks) {
		if len(vm.tx.TxIn[vm.txIdx].SignatureScript) < vm.sigChecks*43-60 {
			return scriptError(ErrInputSigChecks,
				"script exceeds maximum signature density")
		}
	}

	v, err := vm.dstack.PopBool(false)
	if err != nil {
		return err
	}
	if !v {
		// Log interesting data.
		log.Tracef("%v", newLogClosure(func() string {
			dis0, _ := vm.DisasmScript(0)
			dis1, _ := vm.DisasmScript(1)
			return fmt.Sprintf("scripts failed: script0: %s\n"+
				"script1: %s", dis0, dis1)
		}))
		return scriptError(ErrEvalFalse,
			"false stack entry at end of script execution")
	}
	return nil
}

// Step will execute the next instruction and move the program counter to the
// next opcode in the script, or the next script if the current has ended.  Step
// will return true in the case that the last opcode was successfully executed.
//
// The result of calling Step or any other method is undefined if an error is
// returned.
func (vm *Engine) Step() (done bool, err error) {
	// Verify that it is pointing to a valid script address.
	err = vm.validPC()
	if err != nil {
		return true, err
	}
	opcode := &vm.scripts[vm.scriptIdx][vm.scriptOff]
	vm.scriptOff++

	// Execute the opcode while taking into account several things such as
	// disabled opcodes, illegal opcodes, maximum allowed operations per
	// script, maximum script element sizes, and conditionals.
	err = vm.executeOpcode(opcode)
	if err != nil {
		return true, err
	}

	// The number of elements in the combination of the data and alt stacks
	// must not exceed the maximum number of stack elements allowed.
	combinedStackSize := vm.dstack.Depth() + vm.astack.Depth()
	if combinedStackSize > MaxStackSize {
		str := fmt.Sprintf("combined stack size %d > max allowed %d",
			combinedStackSize, MaxStackSize)
		return false, scriptError(ErrStackOverflow, str)
	}

	// Prepare for next instruction.
	if vm.scriptOff >= len(vm.scripts[vm.scriptIdx]) {
		// Illegal to have an `if' that straddles two scripts.
		if len(vm.condStack) != 0 {
			return false, scriptError(ErrUnbalancedConditional,
				"end of script reached in conditional execution")
		}

		// Alt stack doesn't persist.
		_ = vm.astack.DropN(vm.astack.Depth())

		vm.numOps = 0 // number of ops is per script.
		vm.scriptOff = 0
		if vm.scriptIdx == 0 && vm.bip16 {
			vm.scriptIdx++
			vm.savedFirstStack = vm.GetStack()
		} else if vm.scriptIdx == 1 && vm.bip16 {
			// Put us past the end for CheckErrorCondition()
			vm.scriptIdx++
			// Check script ran successfully and pull the script
			// out of the first stack and execute that.
			err := vm.CheckErrorCondition(false)
			if err != nil {
				return false, err
			}

			script := vm.savedFirstStack[len(vm.savedFirstStack)-1]
			pops, err := parseScript(script)
			if err != nil {
				return false, err
			}
			vm.scripts = append(vm.scripts, pops)

			// Set stack to be the stack from first script minus the
			// script itself
			vm.SetStack(vm.savedFirstStack[:len(vm.savedFirstStack)-1])
		} else {
			vm.scriptIdx++
		}
		// there are zero length scripts in the wild
		if vm.scriptIdx < len(vm.scripts) && vm.scriptOff >= len(vm.scripts[vm.scriptIdx]) {
			vm.scriptIdx++
		}
		vm.lastCodeSep = 0
		if vm.scriptIdx >= len(vm.scripts) {
			return true, nil
		}
	}
	return false, nil
}

// Execute will execute all scripts in the script engine and return either nil
// for successful validation or an error if one occurred.
func (vm *Engine) Execute() (err error) {
	done := false
	for !done {
		log.Tracef("%v", newLogClosure(func() string {
			dis, err := vm.DisasmPC()
			if err != nil {
				return fmt.Sprintf("stepping (%v)", err)
			}
			return fmt.Sprintf("stepping %v", dis)
		}))

		done, err = vm.Step()
		if err != nil {
			return err
		}
		log.Tracef("%v", newLogClosure(func() string {
			var dstr, astr string

			// if we're tracing, dump the stacks.
			if vm.dstack.Depth() != 0 {
				dstr = "Stack:\n" + vm.dstack.String()
			}
			if vm.astack.Depth() != 0 {
				astr = "AltStack:\n" + vm.astack.String()
			}

			return dstr + astr
		}))
	}

	return vm.CheckErrorCondition(true)
}

// subScript returns the script since the last OP_CODESEPARATOR.
func (vm *Engine) subScript() []parsedOpcode {
	return vm.scripts[vm.scriptIdx][vm.lastCodeSep:]
}

// checkHashTypeEncoding returns whether or not the passed hashtype adheres to
// the strict encoding requirements if enabled.
func (vm *Engine) checkHashTypeEncoding(hashType SigHashType) error {
	if !vm.hasFlag(ScriptVerifyStrictEncoding) {
		return nil
	}

	sigHashType := hashType & ^SigHashAnyOneCanPay
	if vm.hasFlag(ScriptVerifyBip143SigHash) {
		sigHashType ^= SigHashForkID
		if hashType&SigHashForkID == 0 {
			str := fmt.Sprintf("hash type does not contain uahf forkID 0x%x", hashType)
			return scriptError(ErrInvalidSigHashType, str)
		}
	}

	if sigHashType < SigHashAll || sigHashType > SigHashSingle {
		str := fmt.Sprintf("invalid hash type 0x%x", hashType)
		return scriptError(ErrInvalidSigHashType, str)
	}
	return nil
}

// checkPubKeyEncoding returns whether or not the passed public key adheres to
// the strict encoding requirements if enabled.
func (vm *Engine) checkPubKeyEncoding(pubKey []byte) error {
	if vm.hasFlag(ScriptVerifyStrictEncoding) {
		if len(pubKey) == 33 && (pubKey[0] == 0x02 || pubKey[0] == 0x03) {
			// Compressed
			return nil
		}

		if vm.hasFlag(ScriptVerifyCompressedPubkey) {
			return scriptError(ErrPubKeyType, "compressed public keys are required")
		}

		if len(pubKey) == 65 && pubKey[0] == 0x04 {
			// Uncompressed
			return nil
		}

		return scriptError(ErrPubKeyType, "unsupported public key type")
	}

	if vm.hasFlag(ScriptVerifyCompressedPubkey) && (len(pubKey) != 33 || (pubKey[0] != 0x02 && pubKey[0] != 0x03)) {
		return scriptError(ErrPubKeyType, "compressed public keys are required")
	}

	return nil
}

// checkSignatureEncoding returns whether or not the passed signature adheres to
// the strict encoding requirements if enabled.
func (vm *Engine) checkSignatureEncoding(sig []byte) error {
	if !vm.hasFlag(ScriptVerifyDERSignatures) &&
		!vm.hasFlag(ScriptVerifyLowS) &&
		!vm.hasFlag(ScriptVerifyStrictEncoding) {
		return nil
	}

	if vm.hasFlag(ScriptVerifySchnorr) && len(sig) == 64 {
		return nil
	}

	// The format of a DER encoded signature is as follows:
	//
	// 0x30 <total length> 0x02 <length of R> <R> 0x02 <length of S> <S>
	//   - 0x30 is the ASN.1 identifier for a sequence
	//   - Total length is 1 byte and specifies length of all remaining data
	//   - 0x02 is the ASN.1 identifier that specifies an integer follows
	//   - Length of R is 1 byte and specifies how many bytes R occupies
	//   - R is the arbitrary length big-endian encoded number which
	//     represents the R value of the signature.  DER encoding dictates
	//     that the value must be encoded using the minimum possible number
	//     of bytes.  This implies the first byte can only be null if the
	//     highest bit of the next byte is set in order to prevent it from
	//     being interpreted as a negative number.
	//   - 0x02 is once again the ASN.1 integer identifier
	//   - Length of S is 1 byte and specifies how many bytes S occupies
	//   - S is the arbitrary length big-endian encoded number which
	//     represents the S value of the signature.  The encoding rules are
	//     identical as those for R.
	const (
		asn1SequenceID = 0x30
		asn1IntegerID  = 0x02

		// minSigLen is the minimum length of a DER encoded signature and is
		// when both R and S are 1 byte each.
		//
		// 0x30 + <1-byte> + 0x02 + 0x01 + <byte> + 0x2 + 0x01 + <byte>
		minSigLen = 8

		// maxSigLen is the maximum length of a DER encoded signature and is
		// when both R and S are 33 bytes each.  It is 33 bytes because a
		// 256-bit integer requires 32 bytes and an additional leading null byte
		// might required if the high bit is set in the value.
		//
		// 0x30 + <1-byte> + 0x02 + 0x21 + <33 bytes> + 0x2 + 0x21 + <33 bytes>
		maxSigLen = 72

		// sequenceOffset is the byte offset within the signature of the
		// expected ASN.1 sequence identifier.
		sequenceOffset = 0

		// dataLenOffset is the byte offset within the signature of the expected
		// total length of all remaining data in the signature.
		dataLenOffset = 1

		// rTypeOffset is the byte offset within the signature of the ASN.1
		// identifier for R and is expected to indicate an ASN.1 integer.
		rTypeOffset = 2

		// rLenOffset is the byte offset within the signature of the length of
		// R.
		rLenOffset = 3

		// rOffset is the byte offset within the signature of R.
		rOffset = 4
	)

	// The signature must adhere to the minimum and maximum allowed length.
	sigLen := len(sig)
	if sigLen < minSigLen {
		str := fmt.Sprintf("malformed signature: too short: %d < %d", sigLen,
			minSigLen)
		return scriptError(ErrSigTooShort, str)
	}
	if sigLen > maxSigLen {
		str := fmt.Sprintf("malformed signature: too long: %d > %d", sigLen,
			maxSigLen)
		return scriptError(ErrSigTooLong, str)
	}

	// The signature must start with the ASN.1 sequence identifier.
	if sig[sequenceOffset] != asn1SequenceID {
		str := fmt.Sprintf("malformed signature: format has wrong type: %#x",
			sig[sequenceOffset])
		return scriptError(ErrSigInvalidSeqID, str)
	}

	// The signature must indicate the correct amount of data for all elements
	// related to R and S.
	if int(sig[dataLenOffset]) != sigLen-2 {
		str := fmt.Sprintf("malformed signature: bad length: %d != %d",
			sig[dataLenOffset], sigLen-2)
		return scriptError(ErrSigInvalidDataLen, str)
	}

	// Calculate the offsets of the elements related to S and ensure S is inside
	// the signature.
	//
	// rLen specifies the length of the big-endian encoded number which
	// represents the R value of the signature.
	//
	// sTypeOffset is the offset of the ASN.1 identifier for S and, like its R
	// counterpart, is expected to indicate an ASN.1 integer.
	//
	// sLenOffset and sOffset are the byte offsets within the signature of the
	// length of S and S itself, respectively.
	rLen := int(sig[rLenOffset])
	sTypeOffset := rOffset + rLen
	sLenOffset := sTypeOffset + 1
	if sTypeOffset >= sigLen {
		str := "malformed signature: S type indicator missing"
		return scriptError(ErrSigMissingSTypeID, str)
	}
	if sLenOffset >= sigLen {
		str := "malformed signature: S length missing"
		return scriptError(ErrSigMissingSLen, str)
	}

	// The lengths of R and S must match the overall length of the signature.
	//
	// sLen specifies the length of the big-endian encoded number which
	// represents the S value of the signature.
	sOffset := sLenOffset + 1
	sLen := int(sig[sLenOffset])
	if sOffset+sLen != sigLen {
		str := "malformed signature: invalid S length"
		return scriptError(ErrSigInvalidSLen, str)
	}

	// R elements must be ASN.1 integers.
	if sig[rTypeOffset] != asn1IntegerID {
		str := fmt.Sprintf("malformed signature: R integer marker: %#x != %#x",
			sig[rTypeOffset], asn1IntegerID)
		return scriptError(ErrSigInvalidRIntID, str)
	}

	// Zero-length integers are not allowed for R.
	if rLen == 0 {
		str := "malformed signature: R length is zero"
		return scriptError(ErrSigZeroRLen, str)
	}

	// R must not be negative.
	if sig[rOffset]&0x80 != 0 {
		str := "malformed signature: R is negative"
		return scriptError(ErrSigNegativeR, str)
	}

	// Null bytes at the start of R are not allowed, unless R would otherwise be
	// interpreted as a negative number.
	if rLen > 1 && sig[rOffset] == 0x00 && sig[rOffset+1]&0x80 == 0 {
		str := "malformed signature: R value has too much padding"
		return scriptError(ErrSigTooMuchRPadding, str)
	}

	// S elements must be ASN.1 integers.
	if sig[sTypeOffset] != asn1IntegerID {
		str := fmt.Sprintf("malformed signature: S integer marker: %#x != %#x",
			sig[sTypeOffset], asn1IntegerID)
		return scriptError(ErrSigInvalidSIntID, str)
	}

	// Zero-length integers are not allowed for S.
	if sLen == 0 {
		str := "malformed signature: S length is zero"
		return scriptError(ErrSigZeroSLen, str)
	}

	// S must not be negative.
	if sig[sOffset]&0x80 != 0 {
		str := "malformed signature: S is negative"
		return scriptError(ErrSigNegativeS, str)
	}

	// Null bytes at the start of S are not allowed, unless S would otherwise be
	// interpreted as a negative number.
	if sLen > 1 && sig[sOffset] == 0x00 && sig[sOffset+1]&0x80 == 0 {
		str := "malformed signature: S value has too much padding"
		return scriptError(ErrSigTooMuchSPadding, str)
	}

	// Verify the S value is <= half the order of the curve.  This check is done
	// because when it is higher, the complement modulo the order can be used
	// instead which is a shorter encoding by 1 byte.  Further, without
	// enforcing this, it is possible to replace a signature in a valid
	// transaction with the complement while still being a valid signature that
	// verifies.  This would result in changing the transaction hash and thus is
	// a source of malleability.
	if vm.hasFlag(ScriptVerifyLowS) {
		sValue := new(big.Int).SetBytes(sig[sOffset : sOffset+sLen])
		if sValue.Cmp(halfOrder) > 0 {
			return scriptError(ErrSigHighS, "signature is not canonical due "+
				"to unnecessarily high S value")
		}
	}

	return nil
}

// getStack returns the contents of stack as a byte array bottom up
func getStack(stack *stack) [][]byte {
	array := make([][]byte, stack.Depth())
	for i := range array {
		// PeekByteArry can't fail due to overflow, already checked
		array[len(array)-i-1], _ = stack.PeekByteArray(int32(i))
	}
	return array
}

// setStack sets the stack to the contents of the array where the last item in
// the array is the top item in the stack.
func setStack(stack *stack, data [][]byte) {
	// This can not error. Only errors are for invalid arguments.
	_ = stack.DropN(stack.Depth())

	for i := range data {
		stack.PushByteArray(data[i])
	}
}

// isSegwitScript is used to identify segwit input scripts to determine
// if they are eligible for cleanstack exemption under ScriptVerifyAllowSegwitRecovery.
func isSegwitScript(scriptSig []parsedOpcode) bool {
	// A segwit script sig contains a single data push of a redeem script
	if len(scriptSig) != 1 {
		return false
	}

	// The first (and only) opcode must be a data push and since the redeem script must
	// be no less than 4 bytes, the opcode must be greater or equal to OP_DATA_4. It also
	// cannot be greater than OP_PUSHDATA4 since there are no more pushes above that range.
	if scriptSig[0].opcode.value < OP_DATA_4 || scriptSig[0].opcode.value > OP_PUSHDATA4 {
		return false
	}

	redeemScript := scriptSig[0].data

	// The redeem script must be between 4 and 42 bytes.
	if len(redeemScript) < 4 || len(redeemScript) > 42 {
		return false
	}

	// The first byte of the redeem script must be either OP_0 or in the range OP_1 - OP_16
	if redeemScript[0] != OP_0 && (redeemScript[0] < OP_1 || redeemScript[0] > OP_16) {
		return false
	}

	// The second byte of the redeem script must equal the length of the redeem script minus 2.
	return redeemScript[1] == uint8(len(redeemScript)-2)
}

// isSegwitExemption returns true if ScriptVerifyAllowSegwitRecovery
// is set and this is a bip16 (P2SH) input and the script passes the
// isSegwitScript checks.
func (vm *Engine) isSegwitExemption() bool {
	return vm.hasFlag(ScriptVerifyAllowSegwitRecovery) && vm.bip16 && isSegwitScript(vm.scripts[0])
}

// GetStack returns the contents of the primary stack as an array. where the
// last item in the array is the top of the stack.
func (vm *Engine) GetStack() [][]byte {
	return getStack(&vm.dstack)
}

// SetStack sets the contents of the primary stack to the contents of the
// provided array where the last item in the array will be the top of the stack.
func (vm *Engine) SetStack(data [][]byte) {
	setStack(&vm.dstack, data)
}

// GetAltStack returns the contents of the alternate stack as an array where the
// last item in the array is the top of the stack.
func (vm *Engine) GetAltStack() [][]byte {
	return getStack(&vm.astack)
}

// SetAltStack sets the contents of the alternate stack to the contents of the
// provided array where the last item in the array will be the top of the stack.
func (vm *Engine) SetAltStack(data [][]byte) {
	setStack(&vm.astack, data)
}

// Clone deep copies the state of the vm into a new instance
// and returns it.
func (vm *Engine) Clone() *Engine {
	if vm == nil {
		return nil
	}
	newVM := &Engine{
		txIdx:       vm.txIdx,
		scriptOff:   vm.scriptOff,
		scriptIdx:   vm.scriptIdx,
		numOps:      vm.numOps,
		lastCodeSep: vm.lastCodeSep,
		tx:          vm.tx,
		bip16:       vm.bip16,
		flags:       vm.flags,
		inputAmount: vm.inputAmount,
		sigCache:    vm.sigCache,
		hashCache:   vm.hashCache,
	}
	newVM.savedFirstStack = make([][]byte, len(vm.savedFirstStack))
	for i, stack := range vm.savedFirstStack {
		newVM.savedFirstStack[i] = make([]byte, len(stack))
		copy(newVM.savedFirstStack[i], stack)
	}

	newVM.condStack = make([]int, len(vm.condStack))
	copy(newVM.condStack, vm.condStack)

	newVM.scripts = make([][]parsedOpcode, len(vm.scripts))
	for i, script := range vm.scripts {
		newVM.scripts[i] = make([]parsedOpcode, len(script))
		copy(newVM.scripts[i], script)
	}

	newVM.astack.verifyMinimalData = vm.astack.verifyMinimalData
	newVM.astack.stk = make([][]byte, len(vm.astack.stk))
	for i, data := range vm.astack.stk {
		newVM.astack.stk[i] = make([]byte, len(data))
		copy(newVM.astack.stk[i], data)
	}

	newVM.dstack.verifyMinimalData = vm.dstack.verifyMinimalData
	newVM.dstack.stk = make([][]byte, len(vm.dstack.stk))
	for i, data := range vm.dstack.stk {
		newVM.dstack.stk[i] = make([]byte, len(data))
		copy(newVM.dstack.stk[i], data)
	}

	return newVM
}

// NewEngine returns a new script engine for the provided public key script,
// transaction, and input index.  The flags modify the behavior of the script
// engine according to the description provided by each flag.
func NewEngine(scriptPubKey []byte, tx *wire.MsgTx, txIdx int, flags ScriptFlags,
	sigCache *SigCache, hashCache *TxSigHashes, inputAmount int64,
) (*Engine, error) {
	// The provided transaction input index must refer to a valid input.
	if txIdx < 0 || txIdx >= len(tx.TxIn) {
		str := fmt.Sprintf("transaction input index %d is negative or "+
			">= %d", txIdx, len(tx.TxIn))
		return nil, scriptError(ErrInvalidIndex, str)
	}
	scriptSig := tx.TxIn[txIdx].SignatureScript

	// When both the signature script and public key script are empty the
	// result is necessarily an error since the stack would end up being
	// empty which is equivalent to a false top element.  Thus, just return
	// the relevant error now as an optimization.
	if len(scriptSig) == 0 && len(scriptPubKey) == 0 {
		return nil, scriptError(ErrEvalFalse,
			"false stack entry at end of script execution")
	}

	// The clean stack flag (ScriptVerifyCleanStack) is not allowed without
	// the pay-to-script-hash (P2SH) evaluation (ScriptBip16) flag.
	//
	// Recall that evaluating a P2SH script without the flag set results in
	// non-P2SH evaluation which leaves the P2SH inputs on the stack.
	// Thus, allowing the clean stack flag without the P2SH flag would make
	// it possible to have a situation where P2SH would not be a soft fork
	// when it should be.
	vm := Engine{
		flags: flags, sigCache: sigCache, hashCache: hashCache,
		inputAmount: inputAmount,
	}
	if vm.hasFlag(ScriptVerifyCleanStack) && (!vm.hasFlag(ScriptBip16)) {
		return nil, scriptError(ErrInvalidFlags,
			"invalid flags combination")
	}

	// The signature script must only contain data pushes when the
	// associated flag is set.
	if vm.hasFlag(ScriptVerifySigPushOnly) && !IsPushOnlyScript(scriptSig) {
		return nil, scriptError(ErrNotPushOnly,
			"signature script is not push only")
	}

	// The engine stores the scripts in parsed form using a slice.  This
	// allows multiple scripts to be executed in sequence.  For example,
	// with a pay-to-script-hash transaction, there will be ultimately be
	// a third script to execute.
	scripts := [][]byte{scriptSig, scriptPubKey}
	vm.scripts = make([][]parsedOpcode, len(scripts))
	for i, scr := range scripts {
		if len(scr) > MaxScriptSize {
			str := fmt.Sprintf("script size %d is larger than max "+
				"allowed size %d", len(scr), MaxScriptSize)
			return nil, scriptError(ErrScriptTooBig, str)
		}
		var err error
		vm.scripts[i], err = parseScript(scr)
		if err != nil {
			return nil, err
		}
	}

	// Advance the program counter to the public key script if the signature
	// script is empty since there is nothing to execute for it in that
	// case.
	if len(scripts[0]) == 0 {
		vm.scriptIdx++
	}

	if vm.hasFlag(ScriptBip16) && isScriptHash(vm.scripts[1]) {
		// Only accept input scripts that push data for P2SH.
		if !isPushOnly(vm.scripts[0]) {
			return nil, scriptError(ErrNotPushOnly,
				"pay to script hash is not push only")
		}
		vm.bip16 = true
	}
	if vm.hasFlag(ScriptVerifyMinimalData) {
		vm.dstack.verifyMinimalData = true
		vm.astack.verifyMinimalData = true
	}

	vm.tx = *tx
	vm.txIdx = txIdx

	return &vm, nil
}
