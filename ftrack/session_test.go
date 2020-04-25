package ftrack

import (
	"errors"
	"fmt"
	"github.com/go-shadow/moment"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"testing"
)

func mustEnvLookUp(t *testing.T, key string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		t.Fatalf("no key in env: %s", key)
	}
	return val
}

func deepGet(entity map[string]interface{}, path ...string) (interface{}, error) {
	for i, p := range path {
		value, ok := entity[p]
		if !ok {
			return nil, errors.New(fmt.Sprintf("deep get failed, stopped at %s", p))
		}
		if !(i+1 == len(path)) {
			entity = value.(map[string]interface{})
		} else {
			return value, nil
		}
	}
	return nil, errors.New(fmt.Sprintf("deep get failed"))
}

func mustDeepGet(t *testing.T, data interface{}, path ...string) interface{} {
	value, err := deepGet(data.(map[string]interface{}), path...)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func freshSession(t *testing.T) *Session {

	s, err := NewSession(SessionConfig{
		ApiKey:    mustEnvLookUp(t, "FTRACK_API_KEY"),
		ApiUser:   mustEnvLookUp(t, "FTRACK_API_USER"),
		ServerUrl: mustEnvLookUp(t, "FTRACK_SERVER"),
	})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestSession_Initialized(t *testing.T) {
	assert.Equal(t,
		freshSession(t).Initialized, true,
		"Should initialize the session automatically",
	)
	_, err := NewSession(SessionConfig{
		ApiKey:    "INVALID_API_KEY",
		ApiUser:   mustEnvLookUp(t, "FTRACK_API_USER"),
		ServerUrl: mustEnvLookUp(t, "FTRACK_SERVER"),
	})
	switch err.(type) {
	default:
		t.Fatal("Should reject invalid credentials with ServerError")
	case *ServerError:
	}
}

func TestSession_Call(t *testing.T) {
	session := freshSession(t)
	op1 := NewQueryOperation("select status.state.short from Task where status.state.short is NOT_STARTED limit 1")
	op2 := NewQueryOperation("select status.state.short from Task where status.state.short is NOT_STARTED limit 1")
	result, err := session.Call(op1, op2)
	if err != nil {
		t.Fatal(err)
	}
	t1 := result[0].(QueryResult).Data[0]
	t2 := result[0].(QueryResult).Data[0]
	s1, err := deepGet(t1, "status", "state", "short")
	if err != nil {
		t.Fatal(err)
	}
	s2, err := deepGet(t2, "status", "state", "short")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, s1, s2)
}

func TestSession_Query(t *testing.T) {
	session := freshSession(t)
	q, err := session.Query("select name from Task limit 1")
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, len(q.Data) == 1, "Should allow querying a Task, but empty")
	entityType, err := GetEntityType(q.Data[0])
	if err != nil {
		t.Fatal(err)
	}
	assert.Equalf(t, entityType, "Task", "Should allow querying a Task, but returns %s", entityType)
	q, err = session.Query("select version, asset.versions.version from AssetVersion where asset_id is_not None limit 1")
	if err != nil {
		t.Fatal(err)
	}
	versions := mustDeepGet(t, q.Data[0], "asset", "versions").([]interface{})
	version := q.Data[0]
	var nestedVersion map[string]interface{} = nil
	for _, v := range versions {
		if version["id"].(string) == v.(map[string]interface{})["id"].(string) {
			nestedVersion = v.(map[string]interface{})
		}
	}
	assert.NotNil(t, nestedVersion)
	assert.Equal(t, nestedVersion["version"], version["version"])
}

func TestSession_Decode(t *testing.T) {
	session := freshSession(t)
	iMap := map[string]map[string]interface{}{}
	data := session.Decode([]interface{}{
		map[string]interface{}{
			"id":          1,
			EntityTypeKey: "Task",
			"name":        "foo",
		},
		map[string]interface{}{
			"id":          1,
			EntityTypeKey: "Task",
		},
		map[string]interface{}{
			"id":          2,
			EntityTypeKey: "Task",
			"name":        "bar",
		},
	}, iMap).([]interface{})
	assert.Equal(t, "foo",
		mustDeepGet(t, data[0], "name"),
		"Should support merging 0-level nested data")
	assert.Equal(t, "foo",
		mustDeepGet(t, data[1], "name"),
		"Should support merging 0-level nested data")
	assert.Equal(t, "bar",
		mustDeepGet(t, data[2], "name"),
		"Should support merging 0-level nested data")
	data = session.Decode([]interface{}{
		map[string]interface{}{
			"id":          1,
			EntityTypeKey: "Task",
			"name":        "foo",
			"status": map[string]interface{}{
				EntityTypeKey: "Status",
				"id":          2,
				"name":        "In progress",
			},
		},
		map[string]interface{}{
			"id":          2,
			EntityTypeKey: "Task",

			"status": map[string]interface{}{
				EntityTypeKey: "Status",
				"id":          1,
				"name":        "Done",
			},
		},
		map[string]interface{}{
			"id":          3,
			EntityTypeKey: "Task",
			"name":        "bar",
			"status": map[string]interface{}{
				EntityTypeKey: "Status",
				"id":          1,
			},
		},
	}, iMap).([]interface{})
	assert.Equal(t, "In progress",
		mustDeepGet(t, data[0], "status", "name"),
		"Should support merging 1-level nested data")
	assert.Equal(t, "Done",
		mustDeepGet(t, data[1], "status", "name"),
		"Should support merging 1-level nested data")
	assert.Equal(t, "Done",
		mustDeepGet(t, data[2], "status", "name"),
		"Should support merging 1-level nested data")
	data = session.Decode([]interface{}{
		map[string]interface{}{
			"id":              1,
			"__entity_type__": "Task",
			"name":            "foo",
			"status": map[string]interface{}{
				"__entity_type__": "Status",
				"id":              1,
				"name":            "In progress",
				"state": map[string]interface{}{
					"__entity_type__": "State",
					"id":              1,
					"short":           "DONE",
				},
			},
		},
		map[string]interface{}{
			"id":              2,
			"__entity_type__": "Task",

			"status": map[string]interface{}{
				"__entity_type__": "Status",
				"id":              2,
				"name":            "Done",
				"state": map[string]interface{}{
					"__entity_type__": "State",
					"id":              2,
					"short":           "NOT_STARTED",
				},
			},
		},
		map[string]interface{}{
			"id":              3,
			"__entity_type__": "Task",
			"name":            "bar",
			"status": map[string]interface{}{
				"__entity_type__": "Status",
				"id":              3,
				"state": map[string]interface{}{
					"__entity_type__": "State",
					"id":              1,
				},
			},
		},
	}, iMap).([]interface{})
	assert.Equal(t, "DONE",
		mustDeepGet(t, data[0], "status", "state", "short"),
		"Should support merging 2-level nested data")
	assert.Equal(t, "NOT_STARTED",
		mustDeepGet(t, data[1], "status", "state", "short"),
		"Should support merging 2-level nested data")
	assert.Equal(t, "DONE",
		mustDeepGet(t, data[2], "status", "state", "short"),
		"Should support merging 2-level nested data")
}

func TestSession_Encode(t *testing.T) {
	session := freshSession(t)
	now := moment.New().Now()
	data := session.Encode(map[string]interface{}{"time": now})
	encoded, ok := data.(map[string]interface{})["time"].(map[string]interface{})
	if !ok {
		t.Fatal("time is not encoded!")
	}
	assert.Equal(t, encoded["__type__"], "datetime")
	assert.Equal(t, encoded["value"], now.Format("YYYY-MM-DDTHH:mm:ss"))
}

func TestSession_Create(t *testing.T) {
	session := freshSession(t)
	username := uuid.Must(uuid.NewV4(), nil).String()
	create, err := session.Create(
		"User",
		map[string]interface{}{"username": username},
	)
	if err != nil {
		t.Fatal(err)
	}
	entityType, ok := create.Data[EntityTypeKey]
	if !ok {
		t.Fatal("failed to get entityType")
	}
	assert.Equal(t, "User", entityType)
	id := create.Data["id"].(string)
	del, err := session.Delete(entityType.(string), []string{id})
	if err != nil {
		t.Log("Failed to delete user after create", err)
	}
	if !del.Data {
		t.Log("Failed to delete user after create")
	}
}

func TestSession_Update(t *testing.T) {
	session := freshSession(t)
	username := uuid.Must(uuid.NewV4(), nil).String()
	newUsername := uuid.Must(uuid.NewV4(), nil).String()
	create, err := session.Create(
		"User",
		map[string]interface{}{"username": username},
	)
	if err != nil {
		t.Fatal(err)
	}
	entityType, ok := create.Data[EntityTypeKey]
	if !ok {
		t.Fatal("failed to get entityType")
	}
	assert.Equal(t, "User", entityType)
	id := create.Data["id"].(string)
	update, updateErr := session.Update("User", []string{id}, map[string]interface{}{"username": newUsername})
	del, err := session.Delete(entityType.(string), []string{id})
	if err != nil {
		t.Log("Failed to delete user after create", err)
	}
	if !del.Data {
		t.Log("Failed to delete user after create")
	}
	if updateErr != nil {
		t.Fatal(err)
	}
	n := update.Data["username"]
	assert.Equal(t, newUsername, n)
}

func TestSession_Delete(t *testing.T) {
	session := freshSession(t)
	username := uuid.Must(uuid.NewV4(), nil).String()
	create, err := session.Create(
		"User",
		map[string]interface{}{"username": username},
	)
	if err != nil {
		t.Fatal(err)
	}
	entityType, ok := create.Data[EntityTypeKey]
	if !ok {
		t.Fatal("failed to get entityType")
	}
	assert.Equal(t, "User", entityType)
	id := create.Data["id"].(string)
	del, err := session.Delete(entityType.(string), []string{id})
	if err != nil {
		t.Fatal("Failed to delete user after create", err)
	}
	if !del.Data {
		t.Fatal("Failed to delete user after create")
	}
}

func TestSession_CreateComponent(t *testing.T) {
	content := []byte("temporary file's content")
	tmpFile, err := ioutil.TempFile("", "*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.Write(content); err != nil {
		tmpFile.Close()
		t.Fatal(err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatal(err)
	}
	session := freshSession(t)
	create, err := session.CreateComponent(tmpFile.Name(), CreateComponentOptions{})
	if err != nil {
		t.Fatal(err)
	}
	component, _ := session.EnsurePopulated(create[0].Data, []string{"file_type", "id"})
	componentLocation, _ := session.EnsurePopulated(create[1].Data, []string{"resource_identifier", "id"})
	defer session.Delete("FileComponent", []string{component["id"].(string)})
	defer session.Delete("ComponentLocation", []string{componentLocation["id"].(string)})
	resp, err := http.Get(session.GetComponentUrl(uuid.FromStringOrNil(component["id"].(string))))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	uploadedContent, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, content, uploadedContent, "Uploaded file is not the same as original!")

}

func TestSession_EnsurePopulated(t *testing.T) {
	session := freshSession(t)
	q, err := session.Query("select id from Task limit 1")
	if err != nil {
		t.Fatal(err)
	}
	p, err := session.EnsurePopulated(q.Data[0], []string{"name"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := p["name"]; !ok {
		t.Fatal("Not populated!")
	}
}

func TestSession_GetComponentUrl(t *testing.T) {
	session := freshSession(t)
	componentUrl := session.GetComponentUrl(uuid.Must(uuid.NewV4(), nil))
	_, err := url.Parse(componentUrl)
	assert.Nil(t, err)
}

func TestSession_GetIdentifyingKey(t *testing.T) {
	session := freshSession(t)
	q, err := session.Query("select name from Task limit 1")
	if err != nil {
		t.Fatal(err)
	}
	_, err = session.GetIdentifyingKey(q.Data[0])
	if err != nil {
		t.Fatal(err)
	}
}

func TestSession_GetPrimaryKeyAttributes(t *testing.T) {
	session := freshSession(t)
	q, err := session.Query("select name from Task limit 1")
	if err != nil {
		t.Fatal(err)
	}
	et, _ := GetEntityType(q.Data[0])
	pks := session.GetPrimaryKeyAttributes(et)
	if pks == nil || len(pks) == 0 {
		t.Fatal("No PrimaryKeyAttributes returned for entity", q.Data[0])
	}
}

func TestSession_GetSchema(t *testing.T) {
	session := freshSession(t)
	assert.NotNil(t, session.GetSchema("User"))
}

func TestSession_GetThumbnailUrl(t *testing.T) {
	session := freshSession(t)
	thumbnailUrl := session.GetThumbnailUrl(uuid.Must(uuid.NewV4(), nil), 300)
	_, err := url.Parse(thumbnailUrl)
	assert.Nil(t, err)
}

func TestSession_AsyncQuery(t *testing.T) {
	session := freshSession(t)
	ch1, ch2, ch3 :=
		session.AsyncQuery("select id from AssetVersion limit 1"),
		session.AsyncQuery("select id from AssetVersion limit 1"),
		session.AsyncQuery("select id from AssetVersion limit 1")
	q1, q2, q3 := <-ch1, <-ch2, <-ch3
	assert.Nil(t, q1.Err)
	assert.Nil(t, q2.Err)
	assert.Nil(t, q3.Err)
	select {
	case r := <-session.AsyncQuery("select id from AssetVersion limit 10"):
		assert.Nil(t, r.Err)
	case r := <-session.AsyncQuery("select id from AssetVersion limit 2"):
		assert.Nil(t, r.Err)
	case r := <-session.AsyncQuery("select id from AssetVersion limit 25"):
		assert.Nil(t, r.Err)
	}
}
