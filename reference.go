package admin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	_path "path"
	"strconv"
)

// Reference represents a specific location in Database
type Reference struct {
	database *Database
	path     string

	// queries
	startAt      interface{}
	endAt        interface{}
	orderBy      interface{}
	equalTo      interface{}
	limitToFirst int
	limitToLast  int
}

func marshalJSON(v interface{}) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func addQueryJSON(q url.Values, name string, value interface{}) error {
	s, err := marshalJSON(value)
	if err != nil {
		return err
	}
	q.Add(name, s)
	return nil
}

func addQueryInt(q url.Values, name string, value int) {
	s := strconv.Itoa(value)
	q.Add(name, s)
}

func (ref *Reference) url() (*url.URL, error) {
	u, err := url.Parse(ref.database.app.databaseURL + "/" + ref.path + ".json")
	if err != nil {
		return nil, err
	}
	if ref.database.app.tokenSource == nil {
		return u, nil
	}
	tk, err := ref.database.app.tokenSource.Token()
	if err != nil {
		return nil, err
	}
	token := tk.AccessToken
	q := u.Query()
	q.Add("access_token", token)

	if ref.startAt != nil {
		err = addQueryJSON(q, "startAt", ref.startAt)
		if err != nil {
			return nil, err
		}
	}
	if ref.endAt != nil {
		err = addQueryJSON(q, "endAt", ref.endAt)
		if err != nil {
			return nil, err
		}
	}
	if ref.orderBy != nil {
		err = addQueryJSON(q, "orderBy", ref.orderBy)
		if err != nil {
			return nil, err
		}
	}
	if ref.equalTo != nil {
		err = addQueryJSON(q, "equalTo", ref.equalTo)
		if err != nil {
			return nil, err
		}
	}
	if ref.limitToFirst != 0 {
		addQueryInt(q, "limitToFirst", ref.limitToFirst)
	}
	if ref.limitToLast != 0 {
		addQueryInt(q, "limitToLast", ref.limitToLast)
	}

	u.RawQuery = q.Encode()
	return u, nil
}

func (ref *Reference) invokeRequest(method string, body io.Reader) ([]byte, error) {
	url, err := ref.url()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(method, url.String(), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err := ref.database.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		var e struct {
			Error string `json:"error"`
		}
		err = json.Unmarshal(b, &e)
		if err != nil {
			e.Error = resp.Status
		}
		return nil, fmt.Errorf("firebasedatabase: %s", e.Error)
	}
	return bytes.TrimSpace(b), nil
}

// Set writes data to current location
func (ref *Reference) Set(value interface{}) error {
	buf := bytes.NewBuffer([]byte{})
	err := json.NewEncoder(buf).Encode(value)
	if err != nil {
		return err
	}
	_, err = ref.invokeRequest(http.MethodPut, buf)
	if err != nil {
		return err
	}
	return nil
}

// Push pushs data to current location
func (ref *Reference) Push(value interface{}) error {
	buf := bytes.NewBuffer([]byte{})
	err := json.NewEncoder(buf).Encode(value)
	if err != nil {
		return err
	}
	_, err = ref.invokeRequest(http.MethodPost, buf)
	if err != nil {
		return err
	}
	return nil
}

// Remove removes data from current location
func (ref *Reference) Remove() error {
	_, err := ref.invokeRequest(http.MethodDelete, nil)
	if err != nil {
		return err
	}
	return nil
}

// Key returns the last path of Reference
func (ref *Reference) Key() string {
	_, p := _path.Split(ref.path)
	return p
}

// Ref returns a copy
func (ref Reference) Ref() *Reference {
	return &ref
}

// Root returns the root location of database
func (ref *Reference) Root() *Reference {
	return &Reference{
		database: ref.database,
	}
}

// Child returns a Reference for relative path
func (ref Reference) Child(path string) *Reference {
	ref.path = _path.Join(ref.path, path)
	return &ref
}

// Parent returns the parent location of Reference
func (ref Reference) Parent() *Reference {
	ref.path, _ = _path.Split(ref.path)
	return &ref
}

// EndAt implements Query interface
func (ref Reference) EndAt(value interface{}) Query {
	ref.endAt = value
	return &ref
}

// StartAt implements Query interface
func (ref Reference) StartAt(value interface{}) Query {
	ref.startAt = value
	return &ref
}

// EqualTo implements Query interface
func (ref Reference) EqualTo(value interface{}) Query {
	ref.equalTo = value
	return &ref
}

// IsEqual implements Query interface
func (ref Reference) IsEqual(other interface{}) Query {
	panic(ErrNotImplement)
}

// LimitToFirst implements Query interface
func (ref Reference) LimitToFirst(limit int) Query {
	ref.limitToFirst = limit
	return &ref
}

// LimitToLast implements Query interface
func (ref Reference) LimitToLast(limit int) Query {
	ref.limitToLast = limit
	return &ref
}

// OrderByChild implements Query interface
func (ref Reference) OrderByChild(path interface{}) Query {
	ref.orderBy = path
	return &ref
}

// OrderByKey implements Query interface
func (ref Reference) OrderByKey() Query {
	ref.orderBy = "$key"
	return &ref
}

// OrderByPriority implements Query interface
func (ref Reference) OrderByPriority() Query {
	ref.orderBy = "$priority"
	return &ref
}

// OrderByValue implements Query interface
func (ref Reference) OrderByValue() Query {
	ref.orderBy = "$value"
	return &ref
}

// OnValue implements Query interface
func (ref *Reference) OnValue(event chan *DataSnapshot) CancelFunc {
	panic(ErrNotImplement)
}

// OnChildAdded implements Query interface
func (ref *Reference) OnChildAdded(event chan *ChildSnapshot) CancelFunc {
	panic(ErrNotImplement)
}

// OnChildRemoved implements Query interface
func (ref *Reference) OnChildRemoved(event chan *OldChildSnapshot) CancelFunc {
	panic(ErrNotImplement)
}

// OnChildChanged implements Query interface
func (ref *Reference) OnChildChanged(event chan *ChildSnapshot) CancelFunc {
	panic(ErrNotImplement)
}

// OnChildMoved implements Query interface
func (ref *Reference) OnChildMoved(event chan *ChildSnapshot) CancelFunc {
	panic(ErrNotImplement)
}

// OnceValue implements Query interface
func (ref *Reference) OnceValue() (*DataSnapshot, error) {
	// TODO: find from cached first
	b, err := ref.invokeRequest(http.MethodGet, nil)
	if err != nil {
		return nil, err
	}
	return &DataSnapshot{
		ref: ref,
		raw: b,
	}, nil
}

// OnceChildAdded implements Query interface
func (ref *Reference) OnceChildAdded() *ChildSnapshot {
	panic(ErrNotImplement)
}

// OnceChildRemove implements Query interface
func (ref *Reference) OnceChildRemove() *OldChildSnapshot {
	panic(ErrNotImplement)
}

// OnceChildChanged implements Query interface
func (ref *Reference) OnceChildChanged() *ChildSnapshot {
	panic(ErrNotImplement)
}

// OnceChildMoved implements Query interface
func (ref *Reference) OnceChildMoved() *ChildSnapshot {
	panic(ErrNotImplement)
}

func (ref *Reference) String() string {
	panic(ErrNotImplement)
}
