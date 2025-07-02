package putil

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	mrand "math/rand"
	"os"
	"time"
)

// --------------------------------------------------
var mr *mrand.Rand

// [start, end)
func GetRand(start, end int) int {
	if mr == nil {
		mr = mrand.New(mrand.NewSource((time.Now().UnixNano())))
	}
	return mr.Intn(end-start) + start
}

func GetRandStr(n int) string {
	b := make([]byte, n/2+1)
	_, err := rand.Read(b)
	if err != nil {
		return ""
	}

	s := ""
	for _, v := range b {
		s += fmt.Sprintf("%02x", v)
	}
	s = s[:n]
	return s
}

// --------------------------------------------------
// Md5Sum calculates md5 value of some strings.
func Md5Sum(input ...string) string {
	h := md5.New()
	for _, v := range input {
		io.WriteString(h, v)
	}
	sliceCipherStr := h.Sum(nil)
	sMd5 := hex.EncodeToString(sliceCipherStr)
	return sMd5
}

// GetFileMd5 gets file's md5.
func GetFileMd5(fi string) (string, error) {
	f, err := os.Open(fi)
	if err != nil {
		return "", err
	}
	defer f.Close()

	return GetFileMd5Stream(f)
}

// GetFileMd5Stream gets file's md5 by io.Reader
func GetFileMd5Stream(f io.Reader) (string, error) {
	md := md5.New()
	_, err := io.Copy(md, f)
	if err != nil {
		return "", err
	}
	md5 := hex.EncodeToString(md.Sum(nil))
	return md5, nil
}

func PKCS7Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func PKCS7UnPadding(origData []byte) []byte {
	if len(origData) == 0 {
		return origData
	}
	length := len(origData)
	unpadding := int(origData[length-1])
	if length-unpadding < 0 {
		// 1：一开始发现这里有点问题的，但是为了不越界崩溃，先这样处理
		// 2：后来分析得知，如果aes解析失败，这里就相当于是乱码，不符合pkcs7规范
		return origData
	}
	return origData[:(length - unpadding)]
}

func AesEncrypt(origData, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	origData = PKCS7Padding(origData, blockSize)
	// fmt.Println("origData:", origData)
	// ivs := GetRandStr(blockSize)
	// fmt.Printf("iv[%v] : %v", blockSize, ivs)
	// iv := []byte(ivs)

	// iv理论上要用随机数，但是需要拼接在密文中传递，解密方需要用一致的iv，所以暂时没有搞
	iv := key[:blockSize]
	blockMode := cipher.NewCBCEncrypter(block, iv)
	crypted := make([]byte, len(origData))
	blockMode.CryptBlocks(crypted, origData)
	return crypted, nil
}

func AesDecrypt(crypted, key []byte) ([]byte, error) {
	if len(crypted)%len(key) != 0 {
		return nil, errors.New("crypted size don't match key key")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()

	// ivs := GetRandStr(blockSize)
	// fmt.Printf("iv[%v] : %v", blockSize, ivs)
	// iv := []byte(ivs)
	fmt.Println("blockSize:", blockSize) //16

	iv := key[:blockSize]
	blockMode := cipher.NewCBCDecrypter(block, iv)
	origData := make([]byte, len(crypted))
	blockMode.CryptBlocks(origData, crypted)
	// fmt.Println("origData:", origData)
	origData = PKCS7UnPadding(origData)
	return origData, nil
}

func AesEncryptToBase64(origData, key []byte) (string, error) {
	encryptedBytes, err := AesEncrypt(origData, key)
	if err != nil {
		return "", err
	}
	b64 := base64.StdEncoding.EncodeToString(encryptedBytes)
	return b64, nil
}
func AesDecryptFromBase64(b64 string, key []byte) ([]byte, error) {
	encryptedBytes, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, err
	}
	origData, err := AesDecrypt(encryptedBytes, key)
	if err != nil {
		return nil, err
	}
	return origData, nil
}

func RsaGetPriKey(priKeyBytes []byte) *rsa.PrivateKey {
	priBlock, _ := pem.Decode(priKeyBytes)
	if priBlock == nil {
		return nil
	}

	priKey, err := x509.ParsePKCS1PrivateKey(priBlock.Bytes)
	if err != nil {
		return nil
	}
	return priKey
}

func RsaGetPubKey(pubKeyBytes []byte) *rsa.PublicKey {
	pubBlock, _ := pem.Decode(pubKeyBytes)
	if pubBlock == nil {
		return nil
	}

	pubKey, err := x509.ParsePKCS1PublicKey(pubBlock.Bytes)
	if err != nil {
		return nil
	}

	return pubKey
}

func RsaEncryptToBase64(pubKey *rsa.PublicKey, msg string) string {
	encryptedMsg, err := rsa.EncryptPKCS1v15(rand.Reader, pubKey, []byte(msg))
	if err != nil {
		return ""
	}

	encryptedMsg64 := base64.StdEncoding.EncodeToString(encryptedMsg)
	return encryptedMsg64
}

func RsaDecryptFromBase64(priKey *rsa.PrivateKey, encryptedMsg64 string) string {
	encryptedMsg2, err := base64.StdEncoding.DecodeString(encryptedMsg64)
	if err != nil {
		return ""
	}

	orginMsg, err := rsa.DecryptPKCS1v15(rand.Reader, priKey, encryptedMsg2)
	if err != nil {
		return ""
	}
	return string(orginMsg)
}

// Crc32 returns file's crc32 string.
func Crc32(src string) (string, error) {
	//Initialize an empty return string now in case an error has to be returned
	var cRC32String string

	//Open the fhe file located at the given path and check for errors
	file, err := os.Open(src)
	if err != nil {
		return cRC32String, err
	}

	//Tell the program to close the file when the function returns
	defer file.Close()

	//Create the table with the given polynomial
	tablePolynomial := crc32.MakeTable(crc32.IEEE)

	//Open a new hash interface to write the file to
	hash := crc32.New(tablePolynomial)

	//Copy the file in the interface
	if _, err := io.Copy(hash, file); err != nil {
		return cRC32String, err
	}

	//Generate the hash
	hashInBytes := hash.Sum(nil)[:]

	//Encode the hash to a string
	cRC32String = hex.EncodeToString(hashInBytes)

	//Return the output
	return cRC32String, nil
}

// --------------------------------------------------
func HmacSha256(data, secret string) []byte {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))

	return h.Sum(nil)
}

func GetSha256(data []byte) string {
	hash := sha256.New()
	hash.Write(data)
	bytes := hash.Sum(nil)
	return hex.EncodeToString(bytes)
}
