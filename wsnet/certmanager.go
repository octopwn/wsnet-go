package wsnet

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"os"
	"time"
)

// generateCACertificate generates a self-signed CA certificate and key
func generateCACertificate(certFile, keyFile string) error {
    log.Println("Generating CA certificate and key...")

    // Generate CA private key
    caPriv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
    if err != nil {
        return err
    }

    // Generate CA certificate template
    serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
    if err != nil {
        return err
    }

    notBefore := time.Now()
    notAfter := notBefore.Add(365 * 24 * time.Hour)

    template := x509.Certificate{
        SerialNumber: serialNumber,
        Subject: pkix.Name{
            Organization: []string{"Example CA"},
        },
        NotBefore:             notBefore,
        NotAfter:              notAfter,
        KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
        IsCA:                  true,
        BasicConstraintsValid: true,
    }

    // Self-sign the CA certificate
    derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &caPriv.PublicKey, caPriv)
    if err != nil {
        return err
    }

    // Save the CA certificate to certFile
    certOut, err := os.Create(certFile)
    if err != nil {
        return err
    }
    defer certOut.Close()
    if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
        return err
    }

    // Save the CA private key to keyFile
    keyOut, err := os.Create(keyFile)
    if err != nil {
        return err
    }
    defer keyOut.Close()
    caPrivBytes, err := x509.MarshalECPrivateKey(caPriv)
    if err != nil {
        return err
    }
    if err := pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: caPrivBytes}); err != nil {
        return err
    }

    log.Println("CA certificate and key generated successfully.")
    return nil
}

// generateCertificate generates a self-signed TLS certificate
func generateCertificate(certFile, keyFile string) error {
	log.Println("Generating TLS certificate and key...")

	// Generate a private key
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return err
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Example Organization"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	// Self-sign the certificate
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return err
	}

	// Save the certificate to certFile
	certOut, err := os.Create(certFile)
	if err != nil {
		return err
	}
	defer certOut.Close()
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return err
	}

	// Save the private key to keyFile
	keyOut, err := os.Create(keyFile)
	if err != nil {
		return err
	}
	defer keyOut.Close()
	privBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return err
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: privBytes}); err != nil {
		return err
	}

	log.Println("TLS certificate and key generated successfully.")
	return nil
}

// loadTLSConfig loads the TLS configuration from certificate and key files
func loadTLSConfig(certFile, keyFile string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
	}, nil
}