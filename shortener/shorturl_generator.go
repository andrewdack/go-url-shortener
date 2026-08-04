// Package shortener provides functionality to generate a short link out of original link.
package shortener

import (
	"crypto/sha256"
	"fmt"
	"math/big"
	"os"

	"github.com/itchyny/base58-go"
)

func sha2560f(input string) []byte {
	algorithm := sha256.New()
	algorithm.Write([]byte(input))
	return algorithm.Sum(nil)
}

func toBase58EncodedString(bytes []byte) string {
	// use the base58 encoding used by bitcoin
	encoding := base58.BitcoinEncoding

	// Use base58 encoding to get encoded string as byte slice
	encoded, err := encoding.Encode(bytes)

	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	// return string byte slice as a string
	return string(encoded)
}

// GenerateShortLink creates a shortlink string based on hashing the initial link and userId together.
func GenerateShortLink(initialLink string, userId string) string {
	// use sha256 hash on concatenated initial link and user ID
	// Returns a []byte of exactly 32 bytes (256 bits)
	urlHashBytes := sha2560f(initialLink + userId)

	// Derive a big integer number from the hash bytes generated from the hashing
	// This is lossy -> 32 bytes represents number way bigger than uint64 can hold
	// So .Uint64 effecitvely keeps only the low 64 bits and discards the rest of the hash
	generatedNumber := new(big.Int).SetBytes(urlHashBytes).Uint64()
	
	// Turn the uin64 into a string and convert decimal string into bytes
	// Convert bytes to base58 encoded string
	finalString := toBase58EncodedString([]byte(fmt.Sprintf("%d", generatedNumber)))

	// Return only the first 8 characters of the Base58 string and return that as the short link
	// This increases collision risk but with a low traffic and low users should not be a worry
	return finalString[:8]
}