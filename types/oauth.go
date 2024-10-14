package types

import (
	"net/http"
	"time"
)

type OauthConfig struct {
	Issuer       string `json:"issuer,omitempty"`
	ClientId     string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
	Scopes       string `json:"scopes,omitempty"`
	RedirectURI  string `json:"redirect_url,omitempty"`
	ExtraParams  string
}

/*
holds state of active OAuth transactions for callback
*/
type OauthTransactionState struct {
	OauthConfig   OauthConfig
	AuthorizeTime time.Time
	State         string
	// RestoreURL    string
	Callback func(http.ResponseWriter, *http.Request, TokenReponse)
}

type TokenReponse struct {
	AccessToken string `json:"access_token,omitempty"`
	IdToken     string `json:"id_token,omitempty"`
}
