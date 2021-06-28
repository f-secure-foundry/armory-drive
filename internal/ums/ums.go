// copyright (c) f-secure corporation
// https://foundry.f-secure.com
//
// use of this source code is governed by the license
// that can be found in the license file.

package ums

import (
	"github.com/f-secure-foundry/armory-drive/internal/crypto"

	"github.com/f-secure-foundry/tamago/soc/imx6/usdhc"
)

type Card interface {
	Detect() error
	Info() usdhc.CardInfo
	ReadBlocks(int, []byte) error
	WriteBlocks(int, []byte) error
}

// Drive represents an encrypted drive instance.
type Drive struct {
	// Keyring instance
	Keyring *crypto.Keyring

	// Lock is the function which locks the encrypted drive
	Lock func()

	// Ready represents the logical device status
	Ready bool

	// PairingComplete signals pairing completion
	PairingComplete chan bool

	// Mult is the block multiplier
	Mult int

	// Card represents the underlying storage instance
	Card Card

	// send is the queue for IN device responses
	send chan []byte

	// free is the queue for IN device DMA buffers for later release
	free chan uint32

	// dataPending is the buffer for write commands which spawn across
	// multiple USB transfers
	dataPending *writeOp
}

func (d *Drive) Init(card *usdhc.USDHC) (err error) {
	if err = card.Detect(); err != nil {
		return
	}

	d.Card = card
	d.PairingComplete = make(chan bool)
	d.send = make(chan []byte, 2)
	d.free = make(chan uint32, 1)

	return
}
