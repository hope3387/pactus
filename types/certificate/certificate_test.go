package certificate_test

import (
	"fmt"
	"testing"

	"github.com/fxamacker/cbor/v2"
	"github.com/pactus-project/pactus/crypto/bls"
	"github.com/pactus-project/pactus/types/certificate"
	"github.com/pactus-project/pactus/types/validator"
	"github.com/pactus-project/pactus/util"
	"github.com/pactus-project/pactus/util/testsuite"
	"github.com/stretchr/testify/assert"
)

func TestCertificateCBORMarshaling(t *testing.T) {
	ts := testsuite.NewTestSuite(t)

	c1 := ts.GenerateTestCertificate()
	bz1, err := cbor.Marshal(c1)
	assert.NoError(t, err)
	var c2 certificate.Certificate
	err = cbor.Unmarshal(bz1, &c2)
	assert.NoError(t, err)
	assert.NoError(t, c2.BasicCheck())
	assert.Equal(t, c1.Hash(), c1.Hash())

	assert.Equal(t, c1.Hash(), c2.Hash())

	err = cbor.Unmarshal([]byte{1}, &c2)
	assert.Error(t, err)
}

func TestCertificateSignBytes(t *testing.T) {
	ts := testsuite.NewTestSuite(t)

	hash := ts.RandHash()
	height := ts.RandHeight()
	c1 := ts.GenerateTestCertificate()
	bz := certificate.BlockCertificateSignBytes(hash, height, c1.Round())
	assert.NotEqual(t, bz, certificate.BlockCertificateSignBytes(hash, height, c1.Round()+1))
	assert.NotEqual(t, bz, certificate.BlockCertificateSignBytes(ts.RandHash(), height, c1.Round()))
}

func TestInvalidCertificate(t *testing.T) {
	ts := testsuite.NewTestSuite(t)

	cert0 := ts.GenerateTestCertificate()

	t.Run("Invalid height", func(t *testing.T) {
		cert := certificate.NewCertificate(0, 0, cert0.Committers(), cert0.Absentees(), cert0.Signature())

		err := cert.BasicCheck()
		assert.ErrorIs(t, err, certificate.BasicCheckError{
			Reason: "height is not positive: 0",
		})
	})

	t.Run("Invalid round", func(t *testing.T) {
		cert := certificate.NewCertificate(1, -1, cert0.Committers(), cert0.Absentees(), cert0.Signature())

		err := cert.BasicCheck()
		assert.ErrorIs(t, err, certificate.BasicCheckError{
			Reason: "round is negative: -1",
		})
	})

	t.Run("Committers is nil", func(t *testing.T) {
		cert := certificate.NewCertificate(cert0.Height(), cert0.Round(), nil, cert0.Absentees(), cert0.Signature())

		err := cert.BasicCheck()
		assert.ErrorIs(t, err, certificate.BasicCheckError{
			Reason: "committers is missing",
		})
	})

	t.Run("Absentees is nil", func(t *testing.T) {
		cert := certificate.NewCertificate(cert0.Height(), cert0.Round(), cert0.Committers(), nil, cert0.Signature())

		err := cert.BasicCheck()
		assert.ErrorIs(t, err, certificate.BasicCheckError{
			Reason: "absentees is missing",
		})
	})

	t.Run("Signature is nil", func(t *testing.T) {
		cert := certificate.NewCertificate(cert0.Height(), cert0.Round(), cert0.Committers(), cert0.Absentees(), nil)

		err := cert.BasicCheck()
		assert.ErrorIs(t, err, certificate.BasicCheckError{
			Reason: "signature is missing",
		})
	})

	t.Run("Invalid Absentees ", func(t *testing.T) {
		abs := cert0.Absentees()
		abs = append(abs, 0)
		cert := certificate.NewCertificate(cert0.Height(), cert0.Round(), cert0.Committers(), abs, cert0.Signature())

		err := cert.BasicCheck()
		assert.ErrorIs(t, err, certificate.BasicCheckError{
			Reason: fmt.Sprintf("absentees are not a subset of committers: %v, %v",
				cert.Committers(), abs),
		})
	})

	t.Run("Invalid Absentees ", func(t *testing.T) {
		abs := []int32{2, 1}
		cert := certificate.NewCertificate(cert0.Height(), cert0.Round(), cert0.Committers(), abs, cert0.Signature())

		err := cert.BasicCheck()
		assert.ErrorIs(t, err, certificate.BasicCheckError{
			Reason: fmt.Sprintf("absentees are not a subset of committers: %v, %v",
				cert.Committers(), abs),
		})
	})
}

func TestCertificateHash(t *testing.T) {
	ts := testsuite.NewTestSuite(t)

	temp := ts.GenerateTestCertificate()

	cert1 := certificate.NewCertificate(temp.Height(), temp.Round(),
		[]int32{10, 18, 2, 6}, []int32{}, temp.Signature())
	assert.Equal(t, cert1.Committers(), []int32{10, 18, 2, 6})
	assert.Equal(t, cert1.Absentees(), []int32{})
	assert.NoError(t, cert1.BasicCheck())

	cert2 := certificate.NewCertificate(temp.Height(), temp.Round(),
		[]int32{10, 18, 2, 6}, []int32{2, 6}, temp.Signature())
	assert.Equal(t, cert2.Committers(), []int32{10, 18, 2, 6})
	assert.Equal(t, cert2.Absentees(), []int32{2, 6})
	assert.NoError(t, cert2.BasicCheck())

	cert3 := certificate.NewCertificate(temp.Height(), temp.Round(),
		[]int32{10, 18, 2, 6}, []int32{18}, temp.Signature())
	assert.Equal(t, cert3.Committers(), []int32{10, 18, 2, 6})
	assert.Equal(t, cert3.Absentees(), []int32{18})
	assert.NoError(t, cert3.BasicCheck())
}

func TestEncodingCertificate(t *testing.T) {
	ts := testsuite.NewTestSuite(t)

	cert1 := ts.GenerateTestCertificate()
	length := cert1.SerializeSize()

	for i := 0; i < length; i++ {
		w := util.NewFixedWriter(i)
		assert.Error(t, cert1.Encode(w), "encode test %v failed", i)
	}
	w := util.NewFixedWriter(length)
	assert.NoError(t, cert1.Encode(w))

	for i := 0; i < length; i++ {
		cert := new(certificate.Certificate)
		r := util.NewFixedReader(i, w.Bytes())
		assert.Error(t, cert.Decode(r), "decode test %v failed", i)
	}

	cert2 := new(certificate.Certificate)
	r := util.NewFixedReader(length, w.Bytes())
	assert.NoError(t, cert2.Decode(r))
	assert.Equal(t, cert1.Hash(), cert2.Hash())
}

func TestCertificateValidation(t *testing.T) {
	ts := testsuite.NewTestSuite(t)

	pub1, prv1 := ts.RandBLSKeyPair()
	pub2, _ := ts.RandBLSKeyPair()
	pub3, prv3 := ts.RandBLSKeyPair()
	pub4, prv4 := ts.RandBLSKeyPair()
	val1 := validator.NewValidator(pub1, ts.RandInt32(10000))
	val2 := validator.NewValidator(pub2, ts.RandInt32(10000))
	val3 := validator.NewValidator(pub3, ts.RandInt32(10000))
	val4 := validator.NewValidator(pub4, ts.RandInt32(10000))

	validators := []*validator.Validator{val1, val2, val3, val4}
	committers := []int32{
		val1.Number(), val2.Number(), val3.Number(), val4.Number(),
	}
	blockHash := ts.RandHash()
	blockHeight := ts.RandHeight()
	blockRound := ts.RandRound()
	signBytes := certificate.BlockCertificateSignBytes(blockHash, blockHeight, blockRound)
	sig1 := prv1.Sign(signBytes).(*bls.Signature)
	sig3 := prv3.Sign(signBytes).(*bls.Signature)
	sig4 := prv4.Sign(signBytes).(*bls.Signature)
	aggSig := bls.SignatureAggregate(sig1, sig3, sig4)

	t.Run("Invalid height, should return error", func(t *testing.T) {
		cert := certificate.NewCertificate(blockHeight+1, blockRound, committers,
			[]int32{}, aggSig)

		err := cert.Validate(blockHeight, validators, signBytes)
		assert.ErrorIs(t, err, certificate.UnexpectedHeightError{
			Expected: blockHeight,
			Got:      blockHeight + 1,
		})
	})

	t.Run("Invalid committer, should return error", func(t *testing.T) {
		invCommitters := append(committers, ts.Rand.Int31n(1000))
		cert := certificate.NewCertificate(blockHeight, blockRound, invCommitters,
			[]int32{}, aggSig)

		err := cert.Validate(blockHeight, validators, signBytes)

		assert.ErrorIs(t, err, certificate.UnexpectedCommittersError{
			Committers: invCommitters,
		})
	})

	t.Run("Invalid committers, should return error", func(t *testing.T) {
		invCommitters := []int32{
			ts.Rand.Int31n(1000), val2.Number(), val3.Number(), val4.Number(),
		}
		cert := certificate.NewCertificate(blockHeight, blockRound, invCommitters,
			[]int32{}, aggSig)

		err := cert.Validate(blockHeight, validators, signBytes)
		assert.ErrorIs(t, err, certificate.UnexpectedCommittersError{
			Committers: invCommitters,
		})
	})

	t.Run("Doesn't have 2/3 majority", func(t *testing.T) {
		cert := certificate.NewCertificate(blockHeight, blockRound, committers,
			[]int32{val1.Number(), val2.Number()}, aggSig)

		err := cert.Validate(blockHeight, validators, signBytes)
		assert.ErrorIs(t, err, certificate.InsufficientPowerError{
			SignedPower:   2,
			RequiredPower: 3,
		})
	})

	t.Run("Invalid signature, should return error", func(t *testing.T) {
		cert := certificate.NewCertificate(blockHeight, blockRound, committers,
			[]int32{val3.Number()}, aggSig)

		err := cert.Validate(blockHeight, validators, signBytes)
		assert.ErrorIs(t, err, certificate.InvalidSignatureError{
			Signature: aggSig,
		})
	})
	t.Run("Ok, should return no error", func(t *testing.T) {
		cert := certificate.NewCertificate(blockHeight, blockRound, committers,
			[]int32{val2.Number()}, aggSig)

		err := cert.Validate(blockHeight, validators, signBytes)
		assert.NoError(t, err)
	})
}