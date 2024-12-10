package ssf

import (
	"github.com/emanor-okta/go-scim/types"
)

type SecEvtTokenJWTHeader struct {
	Kid string `json:"kid,omitempty"`
	Typ string `json:"typ,omitempty"`
	Alg string `json:"alg,omitempty"`
}

type Reason struct {
	En string `json:"en,omitempty"`
}

type Subject struct {
	Format string `json:"format,omitempty"`
	Email  string `json:"email,omitempty"`
	Sub    string `json:"sub,omitempty"`
	User   struct {
		Format string `json:"format,omitempty"`
		Email  string `json:"email,omitempty"`
		Iss    string `json:"iss,omitempty"`
	} `json:"user,omitempty"`
	Device struct {
		Format string `json:"format,omitempty"`
		Email  string `json:"email,omitempty"`
		Iss    string `json:"iss,omitempty"`
	} `json:"device,omitempty"`
	Tenant struct {
		Format string `json:"format,omitempty"`
		Email  string `json:"email,omitempty"`
		Iss    string `json:"iss,omitempty"`
	} `json:"tenant,omitempty"`
}

type EvtAttributes struct {
	Current_ip            string `json:"current_ip,omitempty"`
	Current_user_agent    string `json:"current_user_agent,omitempty"`
	Event_timestamp       int64  `json:"event_timestamp,omitempty"`
	Initiating_entity     string `json:"initiating_entity,omitempty"`
	Current_level         string `json:"current_level,omitempty"`
	Previous_level        string `json:"previous_level,omitempty"`
	Current_ip_address    string `json:"current_ip_address,omitempty"`
	Previous_ip_address   string `json:"previous_ip_address,omitempty"`
	Last_known_ip         string `json:"last_known_ip,omitempty"`
	Last_known_user_agent string `json:"last_known_user_agent,omitempty"`
	New_value             string `json:"new-value,omitempty"`
	Reason_admin          Reason `json:"reason_admin,omitempty"`
	Reason_user           Reason `json:"reason_user,omitempty"`
	// Subject               map[string]string `json:"subject,omitempty"`
	Subject Subject `json:"subject,omitempty"`
}

// https://openid.net/specs/openid-caep-specification-1_0.html
type Events struct {
	DeviceRiskChange struct {
		EvtAttributes
	} `json:"https://schemas.okta.com/secevent/okta/event-type/device-risk-change,omitempty"`
	IpChange struct {
		EvtAttributes
	} `json:"https://schemas.okta.com/secevent/okta/event-type/ip-change,omitempty"`
	UserRiskChange struct {
		EvtAttributes
	} `json:"https://schemas.okta.com/secevent/okta/event-type/user-risk-change,omitempty"`
	DeviceComplianceChange struct {
		EvtAttributes
	} `json:"https://schemas.openid.net/secevent/caep/event-type/device-compliance-change,omitempty"`
	SessionRevoked struct {
		EvtAttributes
	} `json:"https://schemas.openid.net/secevent/caep/event-type/session-revoked,omitempty"`
	IdentifierChanged struct {
		EvtAttributes
	} `json:"https://schemas.openid.net/secevent/risc/event-type/identifier-changed,omitempty"`
	CredentialChanged struct {
		Credential_type string `json:"credential_type,omitempty"`
		Change_type     string `json:"change_type,omitempty"`
		Friendly_name   string `json:"friendly_name,omitempty"`
		X509_issuer     string `json:"x509_issuer,omitempty"`
		X509_serial     string `json:"x509_serial,omitempty"`
		Fido2_aaguid    string `json:"fido2_aaguid,omitempty"`
		EvtAttributes
	} `json:"https://schemas.openid.net/secevent/caep/event-type/credential-change,omitempty"`
}

type SecEvtTokenJWTBody struct {
	Iss    string `json:"iss,omitempty"`
	Aud    string `json:"aud,omitempty"`
	Jti    string `json:"jti,omitempty"`
	Iat    int64  `json:"iat,omitempty"`
	Events Events `json:"events,omitempty"`
}

// type OauthConfig struct {
// 	Issuer       string `json:"issuer,omitempty"`
// 	ClientId     string `json:"client_id,omitempty"`
// 	ClientSecret string `json:"client_secret,omitempty"`
// 	Scopes       string `json:"scopes,omitempty"`
// 	RedirectURI  string `json:"redirect_url,omitempty"`
// }

// type TokenReponse struct {
// 	AccessToken string `json:"access_token,omitempty"`
// 	IdToken     string `json:"id_token,omitempty"`
// }

type SsfReceiverAppData struct {
	types.TokenReponse
	types.OauthConfig
	Authenticated,
	ForceReAuth bool
	Username,
	UUID string
}
