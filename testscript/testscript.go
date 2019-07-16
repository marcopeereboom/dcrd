package main

import (
	"encoding/hex"
	"fmt"
	"os"

	"github.com/davecgh/go-spew/spew"
	"github.com/decred/dcrd/chaincfg/chainhash"
	"github.com/decred/dcrd/txscript/v2"
	"github.com/decred/dcrd/wire"
	"github.com/decred/slog"
	//"github.com/davecgh/go-spew/spew"
)

const baseStandardVerifyFlags = txscript.ScriptDiscourageUpgradableNops |
	txscript.ScriptVerifyCleanStack |
	txscript.ScriptVerifyCheckLockTimeVerify |
	txscript.ScriptVerifyCheckSequenceVerify |
	txscript.ScriptVerifyTreasury

// createSpendTx generates a basic spending transaction given the passed
// signature and public key scripts.
func createSpendingTx(sigScript, pkScript []byte) (*wire.MsgTx, error) {
	coinbaseTx := wire.NewMsgTx()

	outPoint := wire.NewOutPoint(&chainhash.Hash{}, ^uint32(0), 0)
	txIn := wire.NewTxIn(outPoint, 0, []byte{txscript.OP_0, txscript.OP_0})
	txOut := wire.NewTxOut(0, pkScript)
	coinbaseTx.AddTxIn(txIn)
	coinbaseTx.AddTxOut(txOut)

	spendingTx := wire.NewMsgTx()
	coinbaseTxSha := coinbaseTx.TxHash()
	outPoint = wire.NewOutPoint(&coinbaseTxSha, 0, 0)
	txIn = wire.NewTxIn(outPoint, 0, sigScript)
	txOut = wire.NewTxOut(0, nil)

	spendingTx.AddTxIn(txIn)
	spendingTx.AddTxOut(txOut)

	return spendingTx, nil
}

func main() {
	// For reference, this was signed with private key:
	//   0x0000000000000000000000000000000000000000000000000000000000000001

	backend := slog.NewBackend(os.Stdout)
	logger := backend.Logger("SCRP")
	logger.SetLevel(slog.LevelTrace)
	txscript.UseLogger(logger)
	sigScript, err := hex.DecodeString("483045022100e67ff2e958d066294263617782bedca9eddf1a88e15711caa8a0dd5db5317cfb0220490ca23dced8b91476492ef7e510847f2dd9e67c8f4a7f24ad2bc0e6e574b70201210279be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798")
	if err != nil {
		fmt.Println(err)
		return
	}
	pkScript, err := hex.DecodeString("76a914e280cb6e66b96679aec288b1fbdbd4db08077a1b88ac")
	if err != nil {
		fmt.Println(err)
		return
	}

	tx, err := createSpendingTx(sigScript, pkScript)
	if err != nil {
		fmt.Println(err)
		return
	}

	vm, err := txscript.NewEngine(pkScript, tx, 0, baseStandardVerifyFlags, 0, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	//if err := vm.Execute(); err != nil {
	//	fmt.Println(err)
	//	return
	//}

	// Alternatively, you can comment out the execute above, step through
	// it manually, and examine the stack.
	///*
	done := false
	for !done {
		// Display the instruction and associated data at the program
		// counter.
		dis, err := vm.DisasmPC()
		if err != nil {
			fmt.Printf("stepping (%v)\n", err)
		}
		fmt.Printf("stepping %v\n", dis)

		// Execute the next instruction.
		done, err = vm.Step()
		if err != nil {
			fmt.Println(err)
			return
		}

		// Display the stacks.
		dataStack := vm.GetStack()
		if len(dataStack) != 0 {
			fmt.Println("Data stack:")
			spew.Dump(dataStack)
			fmt.Println("")
		}
		altStack := vm.GetAltStack()
		if len(altStack) != 0 {
			fmt.Println("Alternate stack:")
			spew.Dump(altStack)
			fmt.Println("")
		}
	}
	if err := vm.CheckErrorCondition(true); err != nil {
		fmt.Println(err)
	}
	//*/
}
