package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/dgrijalva/jwt-go"
)

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

func userEmail(domain, header string) (string, error) {
	email := ""
	url := fmt.Sprintf("%suserinfo", domain)
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

	var profile = Profile{}
	err = json.NewDecoder(resp.Body).Decode(&profile)

	if err != nil {
		return email, err
	}
	return profile.Email, nil
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

	for k, _ := range jwks.Keys {
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
