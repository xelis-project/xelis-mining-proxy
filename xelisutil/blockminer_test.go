package xelisutil

import (
	"encoding/base64"
	"encoding/hex"
	"testing"
)

const TEST_TIMESTAMP = 0x6553600123456789

func TestBlockMiner(t *testing.T) {

	bl := NewBlockMiner([32]byte{0x11, 0x22, 0x33}, [32]byte{0x44, 0x55, 0x66}, [32]byte{0x77, 0x88, 0x99})

	bl.SetTimestamp(TEST_TIMESTAMP)

	t.Logf("Data: %x\n", bl)
	t.Logf("Blob: %s\n", base64.StdEncoding.EncodeToString(bl.GetBlob()))

	bl2, err := NewBlockMinerFromBlob(bl.GetBlob())
	if err != nil {
		t.Fatal(err)
	}

	bl2.SetTimestamp(TEST_TIMESTAMP)

	t.Logf("Data: %x\n", bl2)

	if bl2 != bl {
		t.Fatal("blocks do not match")
	}

	bl.SetNonce(bl.GetNonce())
	bl.SetTimestamp(bl.GetTimestamp())

	t.Logf("Hash: %x", bl.Hash())

	var expected = [32]byte{212, 43, 173, 95, 141, 46, 3, 75, 142, 248, 13, 200, 57, 20, 28, 122,
		124, 69, 12, 56, 16, 246, 63, 0, 138, 215, 121, 34, 93, 202, 173, 175}

	if bl.Hash() != expected {
		t.Fatalf("expected: %x; got: %x", expected, bl.Hash())
	}

}

func TestBlockMiner2(t *testing.T) {
	workHash, _ := hex.DecodeString("9f580f905c4818bd99af010f545046d072b730c9f1f5445c53792bf446cb7f93")
	extraNonce, _ := hex.DecodeString("7cca1bbf11aa6ba434530292bd187515566c9b704dfa18d9cda1cb4dea176dce")
	pubkey, _ := hex.DecodeString("608338a1907914160c173f8a929af9025e6bfeaf6ab14ae71cad08a227f89e40")

	bm := NewBlockMiner([32]byte(workHash), [32]byte(extraNonce), [32]byte(pubkey))

	bm.SetNonce(5655905578498803587)
	bm.SetTimestamp(1715417118848)

	expected, _ := hex.DecodeString("452d2dbecb7023322e7f4737a65ea3bdaad29a55c5e93e39cc1a253d91fa8f36")

	if bm.Hash() != [32]byte(expected) {
		t.Fatalf("expected: %x, got: %x", expected, bm.Hash())
	}
}
