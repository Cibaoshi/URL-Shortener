package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net/url"
	"strings"
)

func GenerateShortCode() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 6)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[n.Int64()]
	}
	return string(b)
}

func ValidateURL(rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", fmt.Errorf("пустой URL")
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("некорректный формат")
	}

	if u.Scheme == "" && u.Host == "" {
		if !strings.Contains(rawURL, ".") {
			return "", fmt.Errorf("это не похоже на адрес сайта (нет точки)")
		}
		rawURL = "https://" + rawURL
		u, _ = url.Parse(rawURL)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("поддерживается только http и https")
	}
	if u.Host == "" {
		return "", fmt.Errorf("отсутствует домен")
	}
	if !strings.Contains(u.Host, ".") && u.Host != "localhost" {
		return "", fmt.Errorf("некорректное доменное имя")
	}
	if u.Host == "localhost:8080" {
		return "", fmt.Errorf("нельзя сокращать ссылки этого сервиса")
	}

	return rawURL, nil
}
