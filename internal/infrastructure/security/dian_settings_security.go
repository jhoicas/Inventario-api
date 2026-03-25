package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const defaultDIANBasePath = "storage/private/dian"

type AesGCMEncryptor struct {
	key []byte
}

func NewAesGCMEncryptor(secret string) (*AesGCMEncryptor, error) {
	if strings.TrimSpace(secret) == "" {
		return nil, errors.New("secret vacío")
	}
	hash := sha256.Sum256([]byte(secret))
	return &AesGCMEncryptor{key: hash[:]}, nil
}

func (e *AesGCMEncryptor) Encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("crear cipher AES: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("crear GCM: %w", err)
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("generar nonce: %w", err)
	}

	cipherText := aead.Seal(nil, nonce, []byte(plaintext), nil)
	combined := append(nonce, cipherText...)
	return base64.StdEncoding.EncodeToString(combined), nil
}

func (e *AesGCMEncryptor) Decrypt(ciphertext string) (string, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("crear cipher AES: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("crear GCM: %w", err)
	}

	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("decodificar ciphertext: %w", err)
	}
	nonceSize := aead.NonceSize()
	if len(raw) < nonceSize {
		return "", errors.New("ciphertext inválido")
	}
	nonce, data := raw[:nonceSize], raw[nonceSize:]
	plain, err := aead.Open(nil, nonce, data, nil)
	if err != nil {
		return "", fmt.Errorf("descifrar ciphertext: %w", err)
	}
	return string(plain), nil
}

type DIANCertificateFileStore struct {
	basePath string
}

func NewDIANCertificateFileStore(basePath string) *DIANCertificateFileStore {
	if strings.TrimSpace(basePath) == "" {
		basePath = defaultDIANBasePath
	}
	return &DIANCertificateFileStore{basePath: basePath}
}

func (s *DIANCertificateFileStore) Save(companyID, environment, originalFileName string, content []byte) (storedPath string, storedName string, err error) {
	if companyID == "" {
		return "", "", errors.New("company_id vacío")
	}
	if len(content) == 0 {
		return "", "", errors.New("archivo vacío")
	}

	envPart := strings.ToLower(strings.TrimSpace(environment))
	if envPart == "" {
		envPart = "test"
	}

	dir := filepath.Join(s.basePath, companyID, envPart)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", "", fmt.Errorf("crear directorio de certificado: %w", err)
	}

	stamp := time.Now().UTC().Format("20060102T150405.000000000")
	fileName := fmt.Sprintf("cert_%s.p12", stamp)
	fullPath := filepath.Join(dir, fileName)

	if err := os.WriteFile(fullPath, content, 0o600); err != nil {
		return "", "", fmt.Errorf("guardar certificado: %w", err)
	}

	return fullPath, fileName, nil
}
