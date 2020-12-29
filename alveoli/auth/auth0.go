package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	jwtmiddleware "github.com/auth0/go-jwt-middleware"
	vespiary "github.com/vx-labs/vespiary/vespiary/api"

	"github.com/dgrijalva/jwt-go"
)

type Jwks struct {
	Keys []JSONWebKeys `json:"keys"`
}

type JSONWebKeys struct {
	Kty string   `json:"kty"`
	Kid string   `json:"kid"`
	Use string   `json:"use"`
	N   string   `json:"n"`
	E   string   `json:"e"`
	X5c []string `json:"x5c"`
}

type Profile struct {
	Sub                 string  `json:"sub"`
	Name                string  `json:"name"`
	GivenName           string  `json:"given_name"`
	FamilyName          string  `json:"family_name"`
	MiddleName          string  `json:"middle_name"`
	Nickname            string  `json:"nickname"`
	PreferredUsername   string  `json:"preferred_username"`
	Profile             string  `json:"profile"`
	Picture             string  `json:"picture"`
	Website             string  `json:"website"`
	Email               string  `json:"email"`
	EmailVerified       bool    `json:"email_verified"`
	Gender              string  `json:"gender"`
	Birthdate           string  `json:"birthdate"`
	Zoneinfo            string  `json:"zoneinfo"`
	Locale              string  `json:"locale"`
	PhoneNumber         string  `json:"phone_number"`
	PhoneNumberVerified bool    `json:"phone_number_verified"`
	Address             Address `json:"address"`
	UpdatedAt           string  `json:"updated_at"`
}

type Address struct {
	Country string `json:"country"`
}

func getPemCert(domain string, token *jwt.Token) (string, error) {
	cert := ""
	url := fmt.Sprintf("%s.well-known/jwks.json", domain)
	resp, err := http.Get(url)

	if err != nil {
		return cert, err
	}
	defer resp.Body.Close()

	var jwks = Jwks{}
	err = json.NewDecoder(resp.Body).Decode(&jwks)

	if err != nil {
		return cert, err
	}

	for k := range jwks.Keys {
		if token.Header["kid"] == jwks.Keys[k].Kid {
			cert = "-----BEGIN CERTIFICATE-----\n" + jwks.Keys[k].X5c[0] + "\n-----END CERTIFICATE-----"
		}
	}

	if cert == "" {
		err := errors.New("Unable to find appropriate key")
		return cert, err
	}

	return cert, nil
}

func auth0Middleware(domain, apiID string) *jwtmiddleware.JWTMiddleware {
	return jwtmiddleware.New(jwtmiddleware.Options{
		ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
			// Verify 'aud' claim
			aud := apiID
			checkAud := token.Claims.(jwt.MapClaims).VerifyAudience(aud, false)
			if !checkAud {
				return token, errors.New("Invalid audience")
			}
			// Verify 'iss' claim
			iss := fmt.Sprintf("https://%s/", domain)
			checkIss := token.Claims.(jwt.MapClaims).VerifyIssuer(iss, false)
			if !checkIss {
				return token, errors.New("Invalid issuer")
			}

			cert, err := getPemCert(iss, token)
			if err != nil {
				panic(err.Error())
			}

			result, _ := jwt.ParseRSAPublicKeyFromPEM([]byte(cert))
			return result, nil
		},
		SigningMethod: jwt.SigningMethodRS256,
	})

}

func (l *auth0Wrapper) ResolveUserEmail(header string) (string, error) {
	email := ""
	url := fmt.Sprintf("https://%s/userinfo", l.domain)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return email, err
	}
	req.Header.Add("Authorization", header)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return email, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("failed to resolve userinfo: got http status code %d", resp.StatusCode)
	}
	var profile = Profile{}
	err = json.NewDecoder(resp.Body).Decode(&profile)

	if err != nil {
		return email, err
	}
	return profile.Email, nil
}

func (l *auth0Wrapper) getTenant(r *http.Request) (string, error) {
	user := r.Context().Value("user")
	claim := user.(*jwt.Token).Claims.(jwt.MapClaims)
	tenant := claim["sub"].(string)
	return tenant, nil
}

type auth0Wrapper struct {
	domain         string
	apiID          string
	vespiaryClient vespiary.VespiaryClient
}

func (l *auth0Wrapper) Authenticate(ctx context.Context, token string) (string, error) {
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		// Verify 'aud' claim
		aud := l.apiID
		checkAud := token.Claims.(jwt.MapClaims).VerifyAudience(aud, false)
		if !checkAud {
			return token, errors.New("Invalid audience")
		}
		// Verify 'iss' claim
		iss := fmt.Sprintf("https://%s/", l.domain)
		checkIss := token.Claims.(jwt.MapClaims).VerifyIssuer(iss, false)
		if !checkIss {
			return token, errors.New("Invalid issuer")
		}
		cert, err := getPemCert(iss, token)
		if err != nil {
			return nil, err
		}
		return jwt.ParseRSAPublicKeyFromPEM([]byte(cert))
	})
	if err != nil {
		return "", err
	}
	claim := parsedToken.Claims.(jwt.MapClaims)
	tenant := claim["sub"].(string)
	return tenant, nil
}
func (l *auth0Wrapper) Validate(ctx context.Context, token string) (UserMetadata, error) {
	tenant, err := l.Authenticate(ctx, token)
	out, err := l.vespiaryClient.GetAccountByPrincipal(ctx, &vespiary.GetAccountByPrincipalRequest{
		Principal: tenant,
	})
	if err != nil {
		return UserMetadata{}, err
	}
	return UserMetadata{Principal: tenant, AccountID: out.Account.ID, Name: out.Account.Name}, nil
}

func Auth0(domain, apiId string, vespiaryClient vespiary.VespiaryClient) Provider {
	return &auth0Wrapper{
		domain:         domain,
		apiID:          apiId,
		vespiaryClient: vespiaryClient,
	}
}
