package state

import (
	"testing"

	"github.com/kong/go-kong/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func certsCollection() *CertificatesCollection {
	return state().Certificates
}

func TestCertificateInsert(t *testing.T) {
	collection := certsCollection()

	var certificate Certificate
	require.Error(t, collection.Add(certificate))

	certificate.ID = new("first")
	require.Error(t, collection.Add(certificate))

	certificate.Key = new("firstKey")
	require.Error(t, collection.Add(certificate))

	certificate.Cert = new("firstCert")
	err := collection.Add(certificate)
	require.NoError(t, err)

	// re-insert
	require.Error(t, collection.Add(certificate))
}

func TestCertificateGetUpdate(t *testing.T) {
	assert := assert.New(t)
	collection := certsCollection()

	var certificate Certificate
	certificate.Cert = new("firstCert")
	certificate.Key = new("firstKey")
	certificate.ID = new("first")
	err := collection.Add(certificate)
	require.NoError(t, err)

	se, err := collection.GetByCertKey("firstCert", "firstKey")
	require.NoError(t, err)
	assert.NotNil(se)
	se.ID = nil
	require.Error(t, collection.Update(*se))

	se.ID = new("first")
	se.Key = nil
	se.Cert = new("firstCert-updated")
	err = collection.Update(*se)
	require.Error(t, err)

	se.Key = new("firstKey-updated")
	err = collection.Update(*se)
	require.NoError(t, err)

	se, err = collection.Get("")
	assert.Nil(se)
	require.Error(t, err)

	se, err = collection.GetByCertKey("firstCert-updated", "firstKey-updated")
	require.NoError(t, err)
	assert.NotNil(se)
	assert.Equal("firstCert-updated", *se.Cert)

	se, err = collection.GetByCertKey("", "")
	require.Error(t, err)
	assert.Nil(se)

	se, err = collection.GetByCertKey("not-present", "firstsdfsdfKey")
	assert.Equal(ErrNotFound, err)
	assert.Nil(se)
}

// Regression test
// to ensure that the memory reference of the pointer returned by Get()
// is different from the one stored in MemDB.
func TestCertificateGetMemoryReference(t *testing.T) {
	assert := assert.New(t)
	collection := certsCollection()

	var cert Certificate
	cert.Cert = new("my-cert")
	cert.Key = new("my-key")
	cert.ID = new("first")
	err := collection.Add(cert)
	require.NoError(t, err)

	c, err := collection.Get("first")
	require.NoError(t, err)
	assert.NotNil(c)
	c.Cert = new("my-new-cert")

	c, err = collection.Get("first")
	require.NoError(t, err)
	assert.NotNil(c)
	assert.Equal("my-cert", *c.Cert)
}

func TestCertificatesInvalidType(t *testing.T) {
	assert := assert.New(t)
	collection := certsCollection()

	var upstream Upstream
	upstream.Name = new("my-upstream")
	upstream.ID = new("first")
	txn := collection.db.Txn(true)
	err := txn.Insert(certificateTableName, &upstream)
	require.Error(t, err)
	txn.Abort()

	type badCertificate struct {
		kong.Certificate
		Meta
	}

	certificate := badCertificate{
		Certificate: kong.Certificate{
			ID:   new("id"),
			Cert: new("Cert"),
			Key:  new("Key"),
		},
	}

	txn = collection.db.Txn(true)
	err = txn.Insert(certificateTableName, &certificate)
	require.NoError(t, err)
	txn.Commit()

	assert.Panics(func() {
		collection.Get("id")
	})

	assert.Panics(func() {
		collection.GetByCertKey("Cert", "Key")
	})
	assert.Panics(func() {
		collection.GetAll()
	})
}

func TestCertificateDelete(t *testing.T) {
	assert := assert.New(t)
	collection := certsCollection()

	var certificate Certificate
	certificate.ID = new("first")
	certificate.Cert = new("firstCert")
	certificate.Key = new("firstKey")
	err := collection.Add(certificate)
	require.NoError(t, err)

	se, err := collection.Get("first")
	require.NoError(t, err)
	assert.NotNil(se)
	assert.Equal("firstCert", *se.Cert)

	err = collection.Delete(*se.ID)
	require.NoError(t, err)

	err = collection.Delete(*se.ID)
	require.Error(t, err)

	certificate.ID = new("first")
	certificate.Cert = new("firstCert")
	certificate.Key = new("firstKey")
	err = collection.Add(certificate)
	require.NoError(t, err)

	se, err = collection.Get("first")
	require.NoError(t, err)
	assert.NotNil(se)
	assert.Equal("firstCert", *se.Cert)

	require.Error(t, collection.DeleteByCertKey("", ""))

	require.Error(t, collection.DeleteByCertKey("foo", "bar"))

	err = collection.DeleteByCertKey(*se.Cert, *se.Key)
	require.NoError(t, err)

	err = collection.Delete("")
	require.Error(t, err)

	err = collection.Delete(*se.ID)
	require.Error(t, err)

	se, err = collection.Get("first")
	require.Error(t, err)
	assert.Nil(se)
}

func TestCertificateGetAll(t *testing.T) {
	assert := assert.New(t)
	collection := certsCollection()

	var certificate Certificate
	certificate.ID = new("first")
	certificate.Cert = new("firstCert")
	certificate.Key = new("firstKey")
	err := collection.Add(certificate)
	require.NoError(t, err)

	var certificate2 Certificate
	certificate2.ID = new("second")
	certificate2.Cert = new("secondCert")
	certificate2.Key = new("secondKey")
	err = collection.Add(certificate2)
	require.NoError(t, err)

	certificates, err := collection.GetAll()

	require.NoError(t, err)
	assert.Len(certificates, 2)
}
