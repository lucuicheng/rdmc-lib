package main

import (
	"fmt"
	"log"
	"time"

	"github.com/dgrijalva/jwt-go"
)

var (
	// 定义密钥
	secretKey = []byte("LP+BAwEBB0xpY2Vuc2UB/4IAAQMBBERhdGEBCgABAVIB/4QAAQFTAf+EAAAACv+DBQEC/4YAAAD+AV7/ggH/8nsiaXNzdWVEYXRlIjoiMjAyMy0wNy0xMlQxMTo0ODo1OC41NzU1NiswODowMCIsImVtYWlsIjoidGVzdEBleGFtcGxlLmNvbSIsInVzZXIiOiJhZG1pbkBsb2NhbGhvc3QiLCJjb21wYW55IjoiQWN0aXZlSU8iLCJhZ2VudENvdW50IjoxMCwidGFza0NvdW50IjoxNSwidmFsaWRpdHlQZXJpb2QiOiIyMDIzLTA3LTIyVDExOjQ4OjU4LjU3NTU2KzA4OjAwIiwiZXhwaXJhdGlvbkRhdGUiOiIyMDIzLTA3LTE3VDE0OjQyOjM5WiJ9ATECqeb5Pc0z4t4QCs/vxZGt9V54co8RFOBbkCfyZrvlF4Q3vp/qnYXS5gyuJyQWMneNATEChay7kD/FJiJXMer2y0i7DVNxrriOwrFnzMi20jSBz9IxP9GLyDZWLgcz9yfw2tgdAA==")
)

func generateToken() (string, error) {
	// 创建声明
	claims := jwt.MapClaims{
		"sub": "admin@localhost",
		"exp": time.Now().Add(time.Hour * 24).Unix(), // 1 day
	}

	// 创建 token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 签名并获取 token 字符串
	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func validateToken(tokenString string) (*jwt.MapClaims, error) {
	// 解析 token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// 校验签名算法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	// 校验 token 是否有效
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return &claims, nil
	}

	return nil, fmt.Errorf("Invalid token")
}

func main() {
	// 生成 token
	tokenString, err := generateToken()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Generated token:", tokenString)

	// 校验 token
	claims, err := validateToken(tokenString)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Valid token. Claims:", claims)
}
