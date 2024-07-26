package utils

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"

	"github.com/google/uuid"
)

func GenerateUUID() string {
	return uuid.NewString()
}

func ConvertHeaderMap(m interface{}) map[string][]string {
	r := map[string][]string{}
	for k, v := range m.(map[string]interface{}) {
		r[k] = []string{}
		for _, v2 := range v.([]interface{}) {
			r[k] = append(r[k], v2.(string))
		}
	}
	return r
}

/*
	JWT Utils
*/

func GetKeyForIDFromIssuer(id, issuer string) (jwk.Key, bool) {
	if strings.Contains(issuer, "/oauth2/") {
		// Custom AS
		issuer = issuer + "/v1/keys"
	} else {
		// Org AS
		issuer = issuer + "/oauth2/v1/keys"
	}

	set, err := jwk.Fetch(context.Background(), issuer)
	if err != nil {
		log.Printf("getKeyForIDFromIssuer: failed to fetch JWK keys from: %s, error: %+v\n", issuer, err)
		return nil, false
	} else {
		key, ok := set.LookupKeyID(id)
		if !ok {
			log.Printf("getKeyForIDFromIssuer: failed to find key: %s\n", id)
			return nil, false
		}
		return key, true
	}
}

func VerifyJwt(jwtBytes []byte, key jwk.Key, alg jwa.KeyAlgorithm) bool {
	_, err := jwt.Parse(jwtBytes, jwt.WithKey(alg, key))
	if err != nil {
		fmt.Printf("verifyJwt: failed to verify JWT: %s, error: %s\n", string(jwtBytes), err)
		return false
	}

	return true
}
