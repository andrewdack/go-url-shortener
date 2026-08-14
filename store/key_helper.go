package store

import "fmt"

// clicksKey generates the Redis key for storing the click count of a given short URL.
func clicksKey(shortUrl string) string {
	return fmt.Sprintf("clicks:%s", shortUrl)
}
