package state

import (
	"testing"

	"github.com/kong/go-kong/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func caCertsCollection() *CACertificatesCollection {
	return state().CACertificates
}

func TestCACertificateInsert(t *testing.T) {
	collection := caCertsCollection()

	var caCert CACertificate
	require.Error(t, collection.Add(caCert))
	caCert.ID = kong.String("first")
	require.Error(t, collection.Add(caCert))
	caCert.Cert = kong.String("firstCert")
	require.NoError(t, collection.Add(caCert))

	// re-inesrt
	require.Error(t, collection.Add(caCert))
}

func TestCACertificateGetUpdate(t *testing.T) {
	assert := assert.New(t)
	collection := caCertsCollection()

	var caCert CACertificate

	require.Error(t, collection.Update(caCert))

	caCert.Cert = kong.String("firstCert")
	caCert.ID = kong.String("first")
	require.Error(t, collection.Update(caCert))

	err := collection.Add(caCert)
	require.NoError(t, err)

	se, err := collection.Get("")
	require.Error(t, err)
	assert.Nil(se)

	se, err = collection.Get("firstCert")
	require.NoError(t, err)
	assert.NotNil(se)
	se.Cert = kong.String("firstCert-updated")
	err = collection.Update(*se)
	require.NoError(t, err)

	se, err = collection.Get("firstCert-updated")
	require.NoError(t, err)
	assert.NotNil(se)
	assert.Equal("firstCert-updated", *se.Cert)

	se, err = collection.Get("not-present")
	assert.Equal(ErrNotFound, err)
	assert.Nil(se)
}

func TestCACertInvalidType(t *testing.T) {
	assert := assert.New(t)
	collection := caCertsCollection()

	var cert Certificate
	cert.Cert = kong.String("my-cert")
	cert.ID = kong.String("first")
	txn := collection.db.Txn(true)
	txn.Insert(caCertTableName, &cert)
	txn.Commit()

	assert.Panics(func() {
		collection.Get("my-cert")
	})
	assert.Panics(func() {
		collection.GetAll()
	})
}

func TestCACertificateDelete(t *testing.T) {
	assert := assert.New(t)
	collection := caCertsCollection()

	require.Error(t, collection.Delete(""))

	var caCert CACertificate
	caCert.ID = kong.String("first")
	caCert.Cert = kong.String("firstCert")
	err := collection.Add(caCert)
	require.NoError(t, err)

	se, err := collection.Get("first")
	require.NoError(t, err)
	assert.NotNil(se)
	assert.Equal("firstCert", *se.Cert)

	err = collection.Delete(*se.ID)
	require.NoError(t, err)

	err = collection.Delete(*se.ID)
	require.Error(t, err)

	caCert.ID = kong.String("first")
	caCert.Cert = kong.String("firstCert")
	err = collection.Add(caCert)
	require.NoError(t, err)

	se, err = collection.Get("first")
	require.NoError(t, err)
	assert.NotNil(se)
	assert.Equal("firstCert", *se.Cert)

	err = collection.Delete(*se.Cert)
	require.NoError(t, err)

	err = collection.Delete(*se.ID)
	require.Error(t, err)
}

func TestCACertificateGetAll(t *testing.T) {
	assert := assert.New(t)
	collection := caCertsCollection()

	var caCert CACertificate
	caCert.ID = kong.String("first")
	caCert.Cert = kong.String("firstCert")
	err := collection.Add(caCert)
	require.NoError(t, err)

	var certificate2 CACertificate
	certificate2.ID = kong.String("second")
	certificate2.Cert = kong.String("secondCert")
	err = collection.Add(certificate2)
	require.NoError(t, err)

	certificates, err := collection.GetAll()

	require.NoError(t, err)
	assert.Len(certificates, 2)
}
