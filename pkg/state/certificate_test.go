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

	certificate.ID = kong.String("first")
	require.Error(t, collection.Add(certificate))

	certificate.Key = kong.String("firstKey")
	require.Error(t, collection.Add(certificate))

	certificate.Cert = kong.String("firstCert")
	err := collection.Add(certificate)
	require.NoError(t, err)

	// re-insert
	require.Error(t, collection.Add(certificate))
}

func TestCertificateGetUpdate(t *testing.T) {
	assert := assert.New(t)
	collection := certsCollection()

	var certificate Certificate
	certificate.Cert = kong.String("firstCert")
	certificate.Key = kong.String("firstKey")
	certificate.ID = kong.String("first")
	err := collection.Add(certificate)
	require.NoError(t, err)

	se, err := collection.GetByCertKey("firstCert", "firstKey")
	require.NoError(t, err)
	assert.NotNil(se)
	se.ID = nil
	require.Error(t, collection.Update(*se))

	se.ID = kong.String("first")
	se.Key = nil
	se.Cert = kong.String("firstCert-updated")
	err = collection.Update(*se)
	require.Error(t, err)

	se.Key = kong.String("firstKey-updated")
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
	cert.Cert = kong.String("my-cert")
	cert.Key = kong.String("my-key")
	cert.ID = kong.String("first")
	err := collection.Add(cert)
	require.NoError(t, err)

	c, err := collection.Get("first")
	require.NoError(t, err)
	assert.NotNil(c)
	c.Cert = kong.String("my-new-cert")

	c, err = collection.Get("first")
	require.NoError(t, err)
	assert.NotNil(c)
	assert.Equal("my-cert", *c.Cert)
}

func TestCertificatesInvalidType(t *testing.T) {
	assert := assert.New(t)
	collection := certsCollection()

	var upstream Upstream
	upstream.Name = kong.String("my-upstream")
	upstream.ID = kong.String("first")
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
			ID:   kong.String("id"),
			Cert: kong.String("Cert"),
			Key:  kong.String("Key"),
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
	certificate.ID = kong.String("first")
	certificate.Cert = kong.String("firstCert")
	certificate.Key = kong.String("firstKey")
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

	certificate.ID = kong.String("first")
	certificate.Cert = kong.String("firstCert")
	certificate.Key = kong.String("firstKey")
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
	certificate.ID = kong.String("first")
	certificate.Cert = kong.String("firstCert")
	certificate.Key = kong.String("firstKey")
	err := collection.Add(certificate)
	require.NoError(t, err)

	var certificate2 Certificate
	certificate2.ID = kong.String("second")
	certificate2.Cert = kong.String("secondCert")
	certificate2.Key = kong.String("secondKey")
	err = collection.Add(certificate2)
	require.NoError(t, err)

	certificates, err := collection.GetAll()

	require.NoError(t, err)
	assert.Len(certificates, 2)
}
