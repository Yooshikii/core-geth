// Copyright 2019 The multi-geth Authors
// This file is part of the multi-geth library.
//
// The multi-geth library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The multi-geth library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the multi-geth library. If not, see <http://www.gnu.org/licenses/>.


package sanity

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"path/filepath"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/params/convert"
	paramtypes "github.com/ethereum/go-ethereum/params/types"
	"github.com/ethereum/go-ethereum/params/types/ctypes"
	"github.com/ethereum/go-ethereum/params/types/parity"
	"github.com/ethereum/go-ethereum/tests"
	"github.com/go-test/deep"
)

func TestEquivalent_Features(t *testing.T) {

	mustValidate := func (c ctypes.ChainConfigurator) {
		zero, max := uint64(0), uint64(math.MaxUint64)
		for _, head := range []*uint64{
			nil, &zero, &max,
		} {
			if err := ctypes.IsValid(c, head); err != nil {
				t.Fatalf("invalid config, err: %v", err)
			}
		}
	}

	for name, oconf := range tests.Forks {
		log.Println(name)
		oconf := oconf

		if oconf.GetConsensusEngineType().IsUnknown() {
			oconf.MustSetConsensusEngineType(ctypes.ConsensusEngineT_Ethash)
		}

		mustValidate(oconf)

		// Integration tests: conversion

		mg := &paramtypes.MultiGethChainConfig{}
		err := convert.Convert(oconf, mg)
		if err != nil {
			t.Fatal(err)
		}

		mustValidate(mg)

		if mg.GetConsensusEngineType().IsUnknown() {
			t.Fatal("unknown consensus mg")
		}

		err = ctypes.Equivalent(oconf, mg)
		if err != nil {
			t.Log("--------------------")
			t.Errorf("%s oconf/mg err: %v", name, err)

			//s := spew.ConfigState{DisableMethods:true, DisablePointerAddresses: true, Indent: "    "}
			//t.Log("OG:", s.Sdump(oconf))
			//t.Log("MG:", s.Sdump(mg))

			nicelog := func (n *uint64) interface{} {
				if n == nil {
					return "nil"
				}
				return *n
			}
			t.Log("o 649", nicelog(oconf.GetEthashEIP649Transition()))
			t.Log("m 649", nicelog(mg.GetEthashEIP649Transition()))
			t.Log("o 1234", nicelog(oconf.GetEthashEIP1234Transition()))
			t.Log("m 1234", nicelog(mg.GetEthashEIP1234Transition()))

			t.Log(mg.GetEthashBlockRewardSchedule())

			// this looks right
			if v := oconf.GetEthashEIP649Transition(); v != nil {
				t.Log(name, "649T", *v)
			} else {
				t.Log(name, "649T", v)
			}
		}

		pc := &parity.ParityChainSpec{}
		err = convert.Convert(oconf, pc)
		if err != nil {
			t.Fatal(err)
		}

		mustValidate(pc)

		err = ctypes.Equivalent(mg, pc)
		if err != nil {
			t.Errorf("%s oconf/p err: %v", name, err)
		}

	}
}

func TestEquivalent_ReadParity(t *testing.T) {
	// These configs are tested by tests/ (ethereum/tests) suite.
	// If passing there, the config pairs are equivalent IN THE CONTEXT OF THOSE TESTS,
	// which is what the configs are for.
	// In order to pass those tests, however, configs do not need to be strictly equivalent.
	// For example, one config might specify EIP1234 fork without a prior EIP649 fork, and
	// another may specify both (either simulaneously or in succession).
	// Both configs in this case yield will equivalent results, but
	// are not, strictly speaking, equivalent.
	// I've left this test here for debugging, and to demonstrate this case.
	t.Skip("(meowsbits): Not required.")
	parityP := filepath.Join("..", "params", "parity.json.d")
	for k, v := range tests.MapForkNameChainspecFileState {
		a := tests.Forks[k]

		b := &parity.ParityChainSpec{}
		bs, err := ioutil.ReadFile(filepath.Join(parityP, v))
		if err != nil {
			t.Fatal(err)
		}
		err = json.Unmarshal(bs, b)
		if err != nil {
			t.Fatal(err)
		}
		err = ctypes.Equivalent(a, b)
		if err != nil {
			t.Log("-------------------")
			t.Log(b.Engine.Ethash.Params.BlockReward)
			t.Log(b.Engine.Ethash.Params.DifficultyBombDelays)
			t.Errorf("%s:%s err: %v", k, v, err)
		}
	}
}

func TestParityGeneses(t *testing.T) {
	testes := []struct{
		filename string
		defaultGenesis *paramtypes.Genesis
	}{
		{
			"foundation.json",
			params.DefaultGenesisBlock(),
		},
		{
			"classic.json",
			params.DefaultClassicGenesisBlock(),
		},
		{
			"mordor.json",
			params.DefaultMordorGenesisBlock(),
		},
		{
			"ropsten.json",
			params.DefaultTestnetGenesisBlock(),
		},
		{
			"kotti.json",
			params.DefaultKottiGenesisBlock(),
		},
	}
	for _, tt := range testes {
		p := filepath.Join("..", "params", "parity.json.d", tt.filename)
		pspec := &parity.ParityChainSpec{}
		b, err := ioutil.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		err = json.Unmarshal(b, pspec)
		if err != nil {
			t.Fatal(err)
		}
		genc := &paramtypes.Genesis{
			Config: &paramtypes.MultiGethChainConfig{},
		}
		err = convert.Convert(pspec, genc)
		if err != nil {
			t.Fatal(err)
		}

		wantBlock:= core.GenesisToBlock(tt.defaultGenesis, nil)
		gotBlock := core.GenesisToBlock(genc, nil)

		if wantBlock.Hash() != gotBlock.Hash() {
			t.Errorf("%s: mismatch gen hash, want(default): %s, got: %s", tt.filename, wantBlock.Hash().Hex(), gotBlock.Hash().Hex() )

			// state roots
			t.Logf("stateroots, want(default): %s, got: %s", wantBlock.Root().Hex(), gotBlock.Root().Hex())

			// extradata
			t.Logf("extras: %x, %x", wantBlock.Extra(), gotBlock.Extra())
			t.Logf("extras_orig: %x, %s", pspec.GetGenesisExtraData(), genc.GetGenesisExtraData())

			diffs := deep.Equal(wantBlock, gotBlock)
			t.Log("genesis block diffs len", len(diffs))
			for _, d := range diffs {
				t.Log(d)
			}

			t.Log("want(default)",spew.Sdump(wantBlock))
			t.Log("got",spew.Sdump(gotBlock))
		}
	}
}