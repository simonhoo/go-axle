package goaxle

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"
)

type KeyRing struct {
	// Identifier is the name given to this KeyRing.  Modification not supported.
	Identifier string `json:"-"`

	// The time this keyring was created.
	// Use of this field is discouraged, use ParseCreatedAt.
	CreatedAt float64 `json:"createdAt,omitempty"`

	// The time this keyring was updated.
	// Use of this field is discouraged, use ParseUpdatedAt.
	UpdatedAt float64 `json:"updatedAt,omitempty"`

	// address where this keyring is located
	axleAddress string
	// do need to create a new keyring on save?
	createOnSave bool
}

// NewKeyRing creates a new KeyRing object with defaults.
func NewKeyRing(axleAddress string, identifier string) (out *KeyRing) {
	out = &KeyRing{
		Identifier:   identifier,
		axleAddress:  axleAddress,
		createOnSave: true,
	}
	return out
}

// Create / Update this KeyRing on the ApiAxle server.
// To modify an existing KeyRing, be sure to retrieve it with GetKeyRing, otherwise
// the library will attempt to create a new KeyRing of the same name.
func (this *KeyRing) Save() (err error) {
	reqAddress := fmt.Sprintf(
		"%s%skeyring/%s",
		this.axleAddress,
		VERSION_ENDPOINT,
		url.QueryEscape(this.Identifier),
	)

	// update the updatedAt timestamp
	this.UpdatedAt = float64(time.Now().UnixNano() / (1000 * 1000))
	marshalled, err := json.Marshal(this)
	if err != nil {
		return fmt.Errorf("Unable to marshal KeyRing: %s", err.Error())
	}

	httpMethod := "POST"
	if !this.createOnSave {
		httpMethod = "PUT"
		// TODO: why have an last updated field if you can't update it?
		return fmt.Errorf("Unable to update key rings, it's not yet supported")
	}

	body, err := doHttpRequest(httpMethod, reqAddress, marshalled)
	if err != nil {
		return err
	}

	if !this.createOnSave {
		err = populateKeyRingFromResponse(&this, body, []string{"results", "new"})
	} else {
		err = populateKeyRingFromResponse(&this, body, []string{"results"})
	}

	if err != nil {
		return err
	}

	this.createOnSave = false

	return nil
}

// GetKeyRing retrieves an existing api object from the server.
func GetKeyRing(axleAddress string, identifier string) (out *KeyRing, err error) {

	reqAddress := fmt.Sprintf("%s%skeyring/%s", axleAddress, VERSION_ENDPOINT, url.QueryEscape(identifier))
	body, err := doHttpRequest("GET", reqAddress, nil)
	if err != nil {
		return nil, err
	}

	// unmarshal into our new keyRing object
	keyRing := NewKeyRing(axleAddress, identifier)
	err = populateKeyRingFromResponse(&keyRing, body, []string{"results"})
	if err != nil {
		return nil, err
	}
	keyRing.createOnSave = false

	return keyRing, err
}

// populateKeyRingFromResponse updates the provided KeyRing pointer with the fields
// provided in the response map.
func populateKeyRingFromResponse(keyRing **KeyRing, body []byte, detailsLocation []string) (err error) {
	response := make(map[string]interface{})
	err = json.Unmarshal(body, &response)
	if err != nil {
		return fmt.Errorf(
			"Unable to unmarshal response: %s",
			err.Error(),
		)
	}

	// navigate to the correct spot in the response to read from
	for _, key := range detailsLocation {
		resultsInterface, exists := response[key]
		if !exists {
			return fmt.Errorf(
				"Response map did not contain expected key: %s",
				key,
			)
		}
		var isValidCast bool
		response, isValidCast = resultsInterface.(map[string]interface{})
		if !isValidCast {
			return fmt.Errorf(
				"key %s did not contain map",
				key,
			)
		}
	}
	// making use of json to populate the object
	jsonvalue, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("Unable to decode keyring in response: %s", err.Error())
	}
	err = json.Unmarshal(jsonvalue, keyRing)
	if err != nil {
		return fmt.Errorf("Unable to decode keyring in response: %s", err.Error())
	}
	return nil
}

// String provides a JSON-like formated representation of this KeyRing object
func (this *KeyRing) String() string {
	out, err := json.MarshalIndent(this, "", "    ")
	if err != nil {
		return "<nil>"
	}
	reqAddress := fmt.Sprintf(
		"%s%skeyring/%s",
		this.axleAddress,
		VERSION_ENDPOINT,
		url.QueryEscape(this.Identifier),
	)
	return fmt.Sprintf("KeyRing - %s: %s", reqAddress, string(out))
}

// DeleteKeyRing removes the identified KeyRing.  Any existing objects represting this
// KeyRing will error on Save().
func DeleteKeyRing(axleAddress string, identifier string) (err error) {
	reqAddress := fmt.Sprintf("%s%skeyring/%s", axleAddress, VERSION_ENDPOINT, url.QueryEscape(identifier))

	body, err := doHttpRequest("DELETE", reqAddress, nil)
	if err != nil {
		return err
	}

	responseMap := make(map[string]interface{})
	err = json.Unmarshal(body, &responseMap)
	if err != nil {
		return fmt.Errorf(
			"Unable to unmarshal response from %s: %s",
			reqAddress,
			err.Error(),
		)
	}

	// in this case, our result is what is contained in the "results" keyring
	resultsInterface, exists := responseMap["results"]
	if !exists {
		return fmt.Errorf("Missing response from %s", reqAddress)
	}
	succeeded, isValidCast := resultsInterface.(bool)
	if !isValidCast {
		return fmt.Errorf(
			"Unable to extract response object from %s",
			reqAddress,
		)
	}

	if !succeeded {
		return fmt.Errorf("Delete of KeyRing at %s failed", reqAddress)
	}

	return nil
}

// Associate a key with a KEYRING.
func (this *KeyRing) LinkKey(keyIdentifier string) (key *Key, err error) {
	return KeyRingLinkKey(this.axleAddress, this.Identifier, keyIdentifier)
}

// Associate a key with a KEYRING.
func KeyRingLinkKey(axleAddress string, keyRingIdentifier string, keyIdentifier string) (key *Key, err error) {

	reqAddress := fmt.Sprintf(
		"%s%skeyring/%s/linkkey/%s",
		axleAddress,
		VERSION_ENDPOINT,
		url.QueryEscape(keyRingIdentifier),
		url.QueryEscape(keyIdentifier),
	)

	body, err := doHttpRequest("PUT", reqAddress, []byte("{}"))
	if err != nil {
		return nil, err
	}

	key = NewKey(axleAddress, keyIdentifier)
	err = populateKeyFromResponse(&key, body, []string{"results"})
	if err != nil {
		return nil, err
	}
	key.createOnSave = false

	return key, nil
}

// UnlinkKey disassociates the provided key with this KeyRing.
func (this *KeyRing) UnlinkKey(keyIdentifier string) (key *Key, err error) {
	return KeyRingUnlinkKey(this.axleAddress, this.Identifier, keyIdentifier)
}

// UnlinkKey disassociates the provided key with this API.
func KeyRingUnlinkKey(axleAddress string, keyRingIdentifier string, keyIdentifier string) (key *Key, err error) {
	reqAddress := fmt.Sprintf(
		"%s%skeyring/%s/unlinkkey/%s",
		axleAddress,
		VERSION_ENDPOINT,
		url.QueryEscape(keyRingIdentifier),
		url.QueryEscape(keyIdentifier),
	)

	body, err := doHttpRequest("PUT", reqAddress, []byte("{}"))
	if err != nil {
		return nil, err
	}

	key = NewKey(axleAddress, keyIdentifier)
	err = populateKeyFromResponse(&key, body, []string{"results"})
	if err != nil {
		return nil, err
	}
	key.createOnSave = false

	return key, nil
}

// List keys belonging to an KEYRING.
func (this *KeyRing) Keys(from int, to int) (keys []*Key, err error) {
	return KeyRingKeys(this.axleAddress, this.Identifier, from, to)
}

// List keys belonging to an KEYRING.
func KeyRingKeys(axleAddress string, identifier string, from int, to int) (keys []*Key, err error) {

	reqAddress := fmt.Sprintf(
		"%s%skeyring/%s/keys?resolve=true&from=%d&to=%d",
		axleAddress,
		VERSION_ENDPOINT,
		url.QueryEscape(identifier),
		from,
		to,
	)

	return doKeysRequest(reqAddress, axleAddress)
}

// Get stats for an keyring
func (this *KeyRing) Stats(from time.Time, to time.Time, granularity Granularity) (stats map[HitType]map[time.Time]map[int]int, err error) {
	return KeyRingStats(this.axleAddress, this.Identifier, from, to, "", "", granularity)
}

// Get stats for an keyring
func (this *KeyRing) StatsForKey(from time.Time, to time.Time, forkey string, granularity Granularity) (stats map[HitType]map[time.Time]map[int]int, err error) {
	return KeyRingStats(this.axleAddress, this.Identifier, from, to, forkey, "", granularity)
}

// Get stats for an keyring
func (this *KeyRing) StatsForApi(from time.Time, to time.Time, forapi string, granularity Granularity) (stats map[HitType]map[time.Time]map[int]int, err error) {
	return KeyRingStats(this.axleAddress, this.Identifier, from, to, "", forapi, granularity)
}

// Get stats for an keyring
func KeyRingStats(axleAddress string, keyRingIdentifier string, from time.Time, to time.Time, forapi string, forkey string, granularity Granularity) (stats map[HitType]map[time.Time]map[int]int, err error) {

	reqAddress := fmt.Sprintf(
		"%s%skeyring/%s/stats?from=%d&to=%d&granularity=%s",
		axleAddress,
		VERSION_ENDPOINT,
		url.QueryEscape(keyRingIdentifier),
		from.Unix(),
		to.Unix(),
		granularity,
	)

	if forkey != "" {
		reqAddress += "&forkey=" + url.QueryEscape(forkey)
	}
	if forapi != "" {
		reqAddress += "&forapi=" + url.QueryEscape(forapi)
	}

	return doStatsRequest(reqAddress)
}

// List all KEYRINGs.
func KeyRings(axleAddress string, from int, to int) (out []*KeyRing, err error) {
	reqAddress := fmt.Sprintf(
		"%s%skeyrings?resolve=true&from=%d&to=%d",
		axleAddress,
		VERSION_ENDPOINT,
		from,
		to,
	)

	body, err := doHttpRequest("GET", reqAddress, nil)
	if err != nil {
		return nil, err
	}

	response := make(map[string]interface{})
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, fmt.Errorf(
			"Unable to unmarshal response: %s",
			err.Error(),
		)
	}
	response, validCast := response["results"].(map[string]interface{})
	if !validCast {
		return nil, fmt.Errorf(
			"Unable to unmarshal response: %s",
			err.Error(),
		)
	}
	out = make([]*KeyRing, len(response))
	x := 0
	for identifier, value := range response {
		keyring := NewKeyRing(axleAddress, identifier)
		jsonvalue, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("Unable to decode keyring in response: %s", err.Error())
		}
		err = json.Unmarshal(jsonvalue, keyring)
		if err != nil {
			return nil, fmt.Errorf("Unable to decode keyring in response: %s", err.Error())
		}
		keyring.createOnSave = false
		out[x] = keyring
		x++
	}

	return out, nil
}

/* ex: set noexpandtab: */
