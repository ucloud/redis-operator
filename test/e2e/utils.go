package e2e

import (
	"fmt"
	"math/rand"
	"net"
	"time"

	"github.com/onsi/ginkgo"
)

var letters = []rune("abcdefghijklmnopqrstuvwxyz1234567890")

// RandString create a random string
func RandString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Logf write log to ginkgo output
func Logf(format string, a ...interface{}) {
	t := time.Now().Format(time.RFC3339)
	l := fmt.Sprintf(format, a...)
	fmt.Fprintf(ginkgo.GinkgoWriter, "%s %s\n", t, l)
}

// IPEqual reports whether ip and x are the same IP address
func IPEqual(s, t string) bool {
	sIP := net.ParseIP(s)
	tIP := net.ParseIP(t)

	return sIP.Equal(tIP)
}
