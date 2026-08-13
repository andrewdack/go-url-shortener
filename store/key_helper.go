package store

import "fmt"

// ownerKey generates the Redis key for storing the owner (user ID) of a given short URL.
func ownerKey(shortUrl string) string {
	return fmt.Sprintf("owner:%s", shortUrl)
}

// clicksKey generates the Redis key for storing the click count of a given short URL.
func clicksKey(shortUrl string) string {
	return fmt.Sprintf("clicks:%s", shortUrl)
}
