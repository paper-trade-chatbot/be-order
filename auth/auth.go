//go:build test
// +build test

package auth

import (
	"crypto/rsa"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/paper-trade-chatbot/be-order/config"
	"github.com/paper-trade-chatbot/be-order/logging"
	"github.com/paper-trade-chatbot/be-order/models"
)

// const (	var (
// 	privKeyPath = "credential/jwt_dev.key"     // openssl genrsa -out app.rsa keysize		pvtBase64 = config.GetString("JWT_PVT_BASE64")
// 	pubKeyPath  = "credential/jwt_dev.key.pub" // openssl rsa -in app.rsa -pubout > app.rsa.pub		pubBase64 = config.GetString("JWT_PUB_BASE64")
// )

var (
	pvtBase64    = config.GetString("JWT_PVT_BASE64")
	pubBase64    = config.GetString("JWT_PUB_BASE64")
	isProduction = config.GetBool("PRODUCTION_ENVIRONMENT")
	projectName  = config.GetString("PROJECT_NAME")
)

func loadData(p string) ([]byte, error) {
	return []byte(p), nil
}

var (
	verifyKey *rsa.PublicKey
	signKey   *rsa.PrivateKey
)

type ClaimSet struct {
	jwt.StandardClaims
}

func init() {

	if isProduction == false {

		signBytes, err := base64.StdEncoding.DecodeString(pvtBase64)
		// signBytes, err := ioutil.ReadFile(privKeyPath)
		if err != nil {
			logging.Error(ctx, "%v", err)
		}
		signKey, err = jwt.ParseRSAPrivateKeyFromPEM(signBytes)
		if err != nil {
			logging.Error(ctx, "%v", err)
		}
		verifyBytes, err := base64.StdEncoding.DecodeString(pubBase64)
		// verifyBytes, err := ioutil.ReadFile(pubKeyPath)
		if err != nil {
			logging.Error(ctx, "%v", err)
		}
		verifyKey, err = jwt.ParseRSAPublicKeyFromPEM(verifyBytes)
		if err != nil {
			logging.Error(ctx, "%v", err)
		}
	} else {

		signBytes, err := loadData(pvtBase64)

		if err != nil {
			logging.Error(ctx, "%v", err)
		}
		signKey, err = jwt.ParseRSAPrivateKeyFromPEM(signBytes)
		if err != nil {
			logging.Error(ctx, "%v", err)
		}
		verifyBytes, err := loadData(pubBase64)

		if err != nil {
			logging.Error(ctx, "%v", err)
		}
		verifyKey, err = jwt.ParseRSAPublicKeyFromPEM(verifyBytes)
		if err != nil {
			logging.Error(ctx, "%v", err)
		}
	}
}

func IssueToken(userId string) (string, error) {

	u, err := uuid.NewRandom()
	claims := ClaimSet{
		jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Hour * time.Duration(10000)).Unix(),
			Issuer:    projectName,
			Id:        u.String(),
			Subject:   userId,
			IssuedAt:  time.Now().Unix(),
			Audience:  userId,
		},
	}

	t := jwt.NewWithClaims(jwt.GetSigningMethod("RS256"), claims)
	tokenString, err := t.SignedString(signKey)
	if err != nil {
		return "", err
	} else {
		return tokenString, nil
	}
}

// Validate Toekn & Claims
func ValidateJwt(ctx *gin.Context) (*models.Me, error) {

	token := ctx.Request.Header.Get("Authorization")
	claims, err := validateToken(token)
	if err != nil {
		logging.Error(ctx, "err %v", err)
		return nil, err
	}
	logging.Info(ctx, "claims %v", claims)
	u, err := validateClaims(claims)

	if err != nil {
		logging.Error(ctx, "err %v", err)
		return nil, err
	}
	ctx.Set("user", u)
	ctx.Set("userId", u.UserId)
	return u, nil
}

// Validate Token
func validateToken(token string) (*ClaimSet, error) {

	trimedToken := strings.Replace(token, " ", "", -1)
	t, err := jwt.ParseWithClaims(trimedToken, &ClaimSet{}, func(token *jwt.Token) (interface{}, error) {
		// since we only use the one private key to sign the tokens,
		// we also only use its public counter part to verify
		return verifyKey, nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := t.Claims.(*ClaimSet); ok && t.Valid {
		return claims, nil
	} else {
		return nil, errors.New("token not valid")
	}
}

// Validate issuer & Set User to context
func validateClaims(claims *ClaimSet) (*models.Me, error) {

	issuer := claims.StandardClaims.Issuer

	// check issuer
	if issuer != projectName {
		return nil, errors.New("wrong issuer")
	}

	// check userId in db
	userId := claims.StandardClaims.Subject
	u, err := models.GetMe(userId)
	if err != nil {
		return nil, err
	}
	return u, nil
}
