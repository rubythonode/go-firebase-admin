package admin

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
)

// FirebaseAuth type
type FirebaseAuth struct {
	app       *FirebaseApp
	keysMutex *sync.RWMutex
	keys      map[string]*rsa.PublicKey
	keysExp   time.Time
}

const keysEndpoint = "https://www.googleapis.com/robot/v1/metadata/x509/securetoken@system.gserviceaccount.com"

func newFirebaseAuth(app *FirebaseApp) *FirebaseAuth {
	return &FirebaseAuth{
		app:       app,
		keysMutex: &sync.RWMutex{},
	}
}

// CreateCustomToken creates custom token
func (auth *FirebaseAuth) CreateCustomToken(userID string, claims interface{}) (string, error) {
	if auth.app.jwtConfig == nil {
		return "", fmt.Errorf("firebaseauth: Service account needed for create custom token")
	}
	now := time.Now()
	payload := &Claims{
		Issuer:    auth.app.jwtConfig.Email,
		Subject:   auth.app.jwtConfig.Email,
		Audience:  "https://identitytoolkit.googleapis.com/google.identity.identitytoolkit.v1.IdentityToolkit",
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(time.Hour).Unix(),
		UserID:    userID,
		Claims:    claims,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, payload)
	return token.SignedString(auth.app.jwtConfig.PrivateKey)
}

// VerifyIDToken verifies idToken
func (auth *FirebaseAuth) VerifyIDToken(idToken string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(idToken, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("firebaseauth: Firebase ID token has incorrect algorithm. Expected \"RSA\" but got \"%#v\"", token.Header["alg"])
		}
		kid := token.Header["kid"].(string)
		if kid == "" {
			return nil, fmt.Errorf("firebaseauth: Firebase ID token has no \"kid\" claim")
		}
		key := auth.selectKey(kid)
		if key == nil {
			return nil, fmt.Errorf("firebaseauth: Firebase ID token has \"kid\" claim which does not correspond to a known public key. Most likely the ID token is expired, so get a fresh token from your client app and try again")
		}
		return key, nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		if !claims.verifyAudience(auth.app.projectID) {
			return nil, fmt.Errorf("firebaseauth: Firebase ID token has incorrect \"aud\" (audience) claim. Expected \"%s\" but got \"%s\"", auth.app.projectID, claims.Audience)
		}
		if !claims.verifyIssuer("https://securetoken.google.com/" + auth.app.projectID) {
			return nil, fmt.Errorf("firebaseauth: Firebase ID token has incorrect \"iss\" (issuer) claim. Expected \"https://securetoken.google.com/%s\" but got \"%s\"", auth.app.projectID, claims.Issuer)
		}
		if claims.Subject == "" {
			return nil, fmt.Errorf("firebaseauth: Firebase ID token has an empty string \"sub\" (subject) claim")
		}
		if len(claims.Subject) > 128 {
			return nil, fmt.Errorf("firebaseauth: Firebase ID token has \"sub\" (subject) claim longer than 128 characters")
		}

		claims.UserID = claims.Subject

		return claims, nil
	}
	return nil, fmt.Errorf("firebaseauth: invalid token")
}

func (auth *FirebaseAuth) fetchKeys() error {
	auth.keysMutex.Lock()
	defer auth.keysMutex.Unlock()
	resp, err := http.Get(keysEndpoint)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	auth.keysExp, _ = time.Parse(time.RFC1123, resp.Header.Get("Expires"))

	m := map[string]string{}
	if err = json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return err
	}
	ks := map[string]*rsa.PublicKey{}
	for k, v := range m {
		p, _ := jwt.ParseRSAPublicKeyFromPEM([]byte(v))
		if p != nil {
			ks[k] = p
		}
	}
	auth.keys = ks
	return nil
}

func (auth *FirebaseAuth) selectKey(kid string) *rsa.PublicKey {
	auth.keysMutex.RLock()
	if auth.keysExp.IsZero() || auth.keysExp.Before(time.Now()) || len(auth.keys) == 0 {
		auth.keysMutex.RUnlock()
		if err := auth.fetchKeys(); err != nil {
			return nil
		}
		auth.keysMutex.RLock()
	}
	defer auth.keysMutex.RUnlock()
	return auth.keys[kid]
}