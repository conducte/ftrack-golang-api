package ftrack

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-shadow/moment"
	uuid "github.com/satori/go.uuid"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	EncodeDatetimeFormat string = "YYYY-MM-DDTHH:mm:ss"
	ServerLocationId     string = "3a372bde-05bc-11e4-8908-20c9d081909b"
	DefaultApiEndpoint   string = "/api"
	EntityTypeKey        string = "__entity_type__"
)

var (
	once      sync.Once
	netClient *http.Client
)

func newNetClient(timeout time.Duration) *http.Client {
	once.Do(func() {
		var netTransport = &http.Transport{
			TLSHandshakeTimeout: 2 * time.Second,
		}
		netClient = &http.Client{
			Timeout:   timeout,
			Transport: netTransport,
		}
	})
	return netClient
}

type Session struct {
	ServerUrl         string
	ApiUser           string
	ApiKey            string
	ApiEndpoint       string
	ClientToken       string
	Timeout           time.Duration
	ServerVersion     string
	Initialized       bool
	Schemas           QuerySchemasResult
	SchemasMap        map[string]map[string]interface{}
	ServerInformation QueryInformationResult
	primaryKeysMap    map[string][]string
}

type SessionConfig struct {
	ApiUser     string
	ApiKey      string
	ServerUrl   string
	ApiEndpoint string
	ClientToken string
	Timeout     time.Duration
}

type callResultWrap struct {
	results     []interface{}
	operations  []interface{}
	identityMap map[string]map[string]interface{}
	session     *Session
}

func (call *callResultWrap) UnmarshalJSON(data []byte) error {
	var rawResults []json.RawMessage
	err := json.Unmarshal(data, &rawResults)
	if err != nil {
		return call.session.getServerError(data)
	}
	for i, raw := range rawResults {
		op := reflect.ValueOf(call.operations[i])

		resultFactory := op.MethodByName("ResultFactory")
		if !resultFactory.IsValid() {
			return errors.New(fmt.Sprintf("failed to get factory for %s with method ResultFactory", call.operations[i]))
		}
		result := resultFactory.Call([]reflect.Value{reflect.ValueOf(call.session)})
		if len(result) != 1 {
			return errors.New(fmt.Sprintf("factory returned invalid value %s must be single value", result))
		}
		decodeResult := result[0].MethodByName("DecodeResult")
		if !decodeResult.IsValid() {
			return errors.New(fmt.Sprintf("failed to get decode function for %s", result[0].Kind()))
		}
		err := json.Unmarshal(raw, result[0].Interface())
		if err != nil {
			return err
		}
		decodeResult.Call([]reflect.Value{
			reflect.ValueOf(call.session),
			reflect.ValueOf(call.identityMap),
		})
		call.results = append(call.results, result[0].Elem().Interface())
	}
	return err
}

type ErrorResponse struct {
	Content   string `json:"content"`
	Exception string `json:"exception"`
	ErrorCode int    `json:"error_code"`
}

func NewSession(config SessionConfig) (*Session, error) {
	if len(config.ServerUrl) == 0 || len(config.ApiUser) == 0 || len(config.ApiKey) == 0 {
		text := "SessionConfig with empty:"
		if len(config.ServerUrl) == 0 {
			text += " ServerUrl "
		}
		if len(config.ApiUser) == 0 {
			text += " ApiUser "
		}
		if len(config.ApiKey) == 0 {
			text += " ApiKey "
		}
		return nil, errors.New(text)
	}
	if len(config.ApiEndpoint) == 0 {
		config.ApiEndpoint = DefaultApiEndpoint
	}
	if len(config.ClientToken) == 0 {
		config.ClientToken = fmt.Sprintf("ftrack-golang-api--%s", uuid.Must(uuid.NewV4(), nil))
	}
	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second
	}
	session := Session{
		ApiUser:     config.ApiUser,
		ApiKey:      config.ApiKey,
		ServerUrl:   config.ServerUrl,
		ApiEndpoint: config.ApiEndpoint,
		ClientToken: config.ClientToken,
		Timeout:     config.Timeout,
		Initialized: false,
	}
	if err := session.initialize(nil); err != nil {
		return nil, err
	}
	return &session, nil
}

func (session *Session) initialize(serverInformationValues []string) error {
	var err error
	result, err := session.Call(
		NewQueryInformationOperation(serverInformationValues),
		NewQuerySchemasOperation(),
	)
	if err != nil {
		return err
	}
	session.ServerInformation = result[0].(QueryInformationResult)
	session.Schemas = result[1].(QuerySchemasResult)
	session.ServerVersion = session.ServerInformation["version"].(string)
	session.SchemasMap = map[string]map[string]interface{}{}
	session.primaryKeysMap = map[string][]string{}
	for _, schema := range session.Schemas {
		type_, ok := schema["id"]
		if !ok {
			return errors.New(fmt.Sprintf("Failed to init schema missing key 'id' in %s", schema))
		}
		typeName := reflect.ValueOf(type_).String()
		session.SchemasMap[typeName] = schema
		primaryKeys, ok := schema["primary_key"]
		if !ok {
			return errors.New(fmt.Sprintf("Failed to init schema missing key 'primary_key' in %s", schema))
		}
		slice, ok := primaryKeys.([]interface{})
		if !ok {
			return errors.New(fmt.Sprintf("Failed to init schema failed to cast 'primary_key' to []interface{}: %s", primaryKeys))
		}
		session.primaryKeysMap[typeName] = []string{}
		for _, pk := range slice {
			pk, ok := pk.(string)
			if !ok {
				return errors.New(fmt.Sprintf("Failed to init schema failed to cast 'primary_key' element to string: %s", pk))
			}
			session.primaryKeysMap[typeName] = append(session.primaryKeysMap[typeName], pk)
		}
	}
	session.Initialized = true
	return nil
}

func (session *Session) GetPrimaryKeyAttributes(entityType string) []string {
	if pks, ok := session.primaryKeysMap[entityType]; ok {
		return pks
	}
	return nil
}

func (session *Session) GetIdentifyingKey(entity map[string]interface{}) (string, error) {
	if entity == nil || !IsEntity(entity) {
		return "", errors.New("invalid entity")
	}
	var key string
	key += entity[EntityTypeKey].(string)
	if pks := session.GetPrimaryKeyAttributes(key); pks != nil {
		for _, pk := range pks {
			key += fmt.Sprintf(",%s", entity[pk])
		}
		return key, nil
	} else {
		return "", errors.New("invalid entity")
	}
}

func (session *Session) Encode(data interface{}) interface{} {
	value := reflect.ValueOf(data)
	momentKind := reflect.ValueOf(*moment.New()).Kind()
	switch value.Kind() {
	case reflect.Array | reflect.Slice:
		var out []interface{}
		for i := 0; i < value.Len(); i++ {
			out = append(out, session.Encode(value.Index(i).Interface()))
		}
		return out
	case reflect.Map:
		iter := value.MapRange()
		for iter.Next() {
			value.SetMapIndex(iter.Key(), reflect.ValueOf(session.Encode(iter.Value().Interface())))
		}
		return data
	case reflect.Ptr:
		if value.Elem().Kind() == momentKind {
			return map[string]interface{}{
				"__type__": "datetime",
				"value":    data.(*moment.Moment).Format(EncodeDatetimeFormat),
			}
		}
		return data
	default:
		return data
	}
}

func (session *Session) getErrorFromResponse(response ErrorResponse) error {
	switch response.Exception {
	case "ValidationError":
		return &ServerValidationError{
			Msg:       response.Content,
			ErrorCode: response.ErrorCode,
			Exception: response.Exception,
		}
	case "FTAuthenticationError", "PermissionError":
		return &ServerPermissionDeniedError{
			Msg:       response.Content,
			ErrorCode: response.ErrorCode,
			Exception: response.Exception,
		}
	default:
		return &ServerError{
			Msg:       response.Content,
			ErrorCode: response.ErrorCode,
			Exception: response.Exception,
		}
	}
}

func (session *Session) Decode(data interface{}, identityMap map[string]map[string]interface{}) interface{} {
	if data == nil {
		return data
	}
	value := reflect.ValueOf(data)
	if identityMap == nil {
		identityMap = map[string]map[string]interface{}{}
	}
	switch value.Kind() {
	case reflect.Array | reflect.Slice:
		for i := 0; i < value.Len(); i++ {
			value.Index(i).Set(reflect.ValueOf(session.Decode(value.Index(i).Interface(), identityMap)))
		}
		return data
	case reflect.Map:
		casted := data.(map[string]interface{})
		if IsDate(casted) {
			return session.decodeDateTime(casted)
		} else if IsEntity(casted) {
			return session.mergeEntity(casted, identityMap)
		}
		iter := value.MapRange()
		for iter.Next() {
			value.SetMapIndex(iter.Key(), reflect.ValueOf(session.Decode(iter.Value().Interface(), identityMap)))
		}
		return data
	default:
		return data
	}
}

func (session *Session) decodeDateTime(data map[string]interface{}) interface{} {
	m := moment.New().Moment(EncodeDatetimeFormat, data["value"].(string))
	if session.ServerInformation["is_timezone_support_enabled"].(bool) {
		return m.UTC().Format(EncodeDatetimeFormat)
	}
	return m.Format(EncodeDatetimeFormat)
}

func (session *Session) mergeEntity(entity interface{}, identityMap map[string]map[string]interface{}) interface{} {
	casted, ok := entity.(map[string]interface{})
	if !ok {
		return entity
	}
	key, err := session.GetIdentifyingKey(casted)
	if err != nil {
		return entity
	}
	if identityMap == nil {
		identityMap = map[string]map[string]interface{}{}
	}
	if _, ok := identityMap[key]; !ok {
		identityMap[key] = casted
	}
	merged := identityMap[key]
	for k, v := range casted {
		merged[k] = session.Decode(v, identityMap)
	}
	return merged
}

func (session *Session) encodeOperations(operations interface{}) ([]byte, error) {

	encoded := session.Encode(&operations)
	return json.Marshal(encoded)
}

func (session *Session) EnsurePopulated(data interface{}, keys []string) (map[string]interface{}, error) {
	entityType, err := GetEntityType(data)
	if err != nil {
		return nil, err
	}
	entity := data.(map[string]interface{})
	primaryKeys := session.GetPrimaryKeyAttributes(entityType)
	if primaryKeys == nil {
		return nil, errors.New(fmt.Sprintf("could't determine primary keys for entity type %s", entityType))
	}
	expression := fmt.Sprintf("select %s from %s where", strings.Join(keys, ", "), entityType)
	var criteria []string
	for _, k := range primaryKeys {
		criteria = append(criteria, fmt.Sprintf("%s is %s", k, entity[k]))
	}
	expression = fmt.Sprintf("%s %s", expression, strings.Join(criteria, " and "))
	response, err := session.Query(expression)
	if err != nil {
		return nil, err
	}
	if len(response.Data) == 0 {
		return nil, errors.New("no entity found")
	}
	if len(response.Data) > 1 {
		return nil, errors.New("multiple entity found")
	}
	for k, v := range response.Data[0] {
		entity[k] = v
	}
	return entity, nil
}

func (session *Session) GetSchema(schemaId string) map[string]interface{} {
	if schema, ok := session.SchemasMap[schemaId]; ok {
		return schema
	}
	return nil
}

func (session *Session) Query(expression string) (*QueryResult, error) {
	result, err := session.Call(NewQueryOperation(expression))
	if err != nil {
		return nil, err
	}
	casted := result[0].(QueryResult)
	return &casted, nil
}

func (session *Session) Create(entityType string, data map[string]interface{}) (*CreateResult, error) {
	operation := NewCreateOperation(entityType, data)
	create, err := session.Call(operation)
	if err != nil {
		return nil, err
	}
	casted := create[0].(CreateResult)
	return &casted, nil
}

func (session *Session) Update(entityType string, keys []string, data map[string]interface{}) (*UpdateResult, error) {
	operation := NewUpdateOperation(entityType, keys, data)
	create, err := session.Call(operation)
	if err != nil {
		return nil, err
	}
	casted := create[0].(UpdateResult)
	return &casted, nil
}

func (session *Session) Delete(entityType string, id []string) (*DeleteResult, error) {
	operation := NewDeleteOperation(entityType, id)
	create, err := session.Call(operation)
	if err != nil {
		return nil, err
	}
	casted := create[0].(DeleteResult)
	return &casted, nil
}

func (session *Session) GetComponentUrl(componentId uuid.UUID) string {
	return fmt.Sprintf(
		"%s/component/get?%s",
		session.ServerUrl,
		encodeUriParameters(
			uriParameter{"id", componentId.String()},
			uriParameter{"username", session.ApiUser},
			uriParameter{"apiKey", session.ApiKey},
		),
	)
}

func (session *Session) GetThumbnailUrl(componentId uuid.UUID, size int) string {
	return fmt.Sprintf(
		"%s/component/thumbnail?%s",
		session.ServerUrl,
		encodeUriParameters(
			uriParameter{"id", componentId.String()},
			uriParameter{"size", strconv.Itoa(size)},
			uriParameter{"username", session.ApiUser},
			uriParameter{"apiKey", session.ApiKey},
		),
	)
}

type CreateComponentOptions struct {
	Id         *uuid.UUID
	OnProgress *func(int)
	OnAborted  *func()
	FileType   *string
	FileSize   *int64
	FileName   *string

	componentLocationId uuid.UUID
}

func (options *CreateComponentOptions) setDefaults(file *os.File) error {
	stat, err := file.Stat()
	if err != nil {
		return err
	}

	if options.Id == nil {
		id := uuid.Must(uuid.NewV4(), nil)
		options.Id = &id
	}
	if options.OnAborted == nil {
		onAborted := func() {}
		options.OnAborted = &onAborted
	}
	if options.OnProgress == nil {
		onProgress := func(int) {}
		options.OnProgress = &onProgress
	}

	if options.FileType == nil {
		ext := filepath.Ext(NormalizeString(file.Name()))
		options.FileType = &ext
	}
	if options.FileSize == nil {
		size := stat.Size()
		options.FileSize = &size
	}
	if options.FileName == nil {
		fileName := NormalizeString(file.Name())
		base := strings.TrimSuffix(filepath.Base(fileName), filepath.Ext(file.Name()))
		options.FileName = &base
	}
	normalizedFileName := NormalizeString(*options.FileName)
	options.FileName = &normalizedFileName
	options.componentLocationId = uuid.Must(uuid.NewV4(), nil)
	return nil
}

type progressReader struct {
	io.Reader
	Reporter func(r int64)
}

func (pr *progressReader) Read(p []byte) (n int, err error) {
	n, err = pr.Reader.Read(p)
	pr.Reporter(int64(n))
	return
}

func (session *Session) CreateComponent(fileName string, options CreateComponentOptions) (result []CreateResult, err error) {
	file, err := os.Open(fileName)
	if err != nil {
		return
	}
	defer func() { _ = file.Close() }()
	if err := options.setDefaults(file); err != nil {
		return nil, err
	}
	results, err := session.Call(
		NewGetUploadMetadataOperation(
			fmt.Sprintf("%s%s", *options.FileName, *options.FileType),
			*options.FileSize, *options.Id,
		),
	)
	if err != nil {
		return
	}
	uploadMetadata := results[0].(GetUploadMetadataResult)
	results, err = session.Call(
		NewCreateOperation("FileComponent", map[string]interface{}{
			"id":        options.Id.String(),
			"name":      *options.FileName,
			"size":      *options.FileSize,
			"file_type": *options.FileType,
		}),
		NewCreateOperation("ComponentLocation", map[string]interface{}{
			"id":                  options.componentLocationId.String(),
			"component_id":        options.Id.String(),
			"location_id":         ServerLocationId,
			"resource_identifier": options.Id.String(),
		}),
	)
	if err != nil {
		return
	}
	result = append(result, results[0].(CreateResult))
	result = append(result, results[1].(CreateResult))
	client := http.Client{ // TODO: Use singleton?
		// TODO: Needed? Timeout: session.Timeout,
	}
	request, err := http.NewRequest(
		"PUT",
		uploadMetadata.Url,
		&progressReader{
			Reader: file,
			Reporter: func(c int64) {
				(*options.OnProgress)(int(float32(c) / float32(*options.FileSize) * 100))
			},
		},
	)
	if err != nil {
		return
	}
	for k, v := range uploadMetadata.Headers {
		if k == "Content-Length" {
			continue
		}
		request.Header.Set(k, v)
	}
	response, err := client.Do(request)
	if err == nil {
		defer response.Body.Close()
	}
	// TODO: add UploadError type?
	var uploadError = err
	if err == nil && response.StatusCode != http.StatusOK {
		text := response.Status
		body, err := ioutil.ReadAll(response.Body)
		if err == nil {
			text += "\n"
			text += string(body)
		}
		uploadError = errors.New(text)
	}
	if uploadError != nil {
		_, _ = session.Call(
			NewDeleteOperation("FileComponent", []string{options.Id.String()}),
			NewDeleteOperation("ComponentLocation", []string{options.componentLocationId.String()}),
		)
		return nil, uploadError
	}
	return
}

func (session *Session) getServerError(response []byte) error {
	var errorResponse ErrorResponse
	err := json.Unmarshal(response, &errorResponse)
	if err == nil {
		return session.getErrorFromResponse(errorResponse)
	}
	return &MalformedResponseError{Content: response}
}

func (session *Session) Call(operations ...interface{}) ([]interface{}, error) {
	response, err := session.call(operations...)
	if err != nil {
		return nil, err
	}
	wrap := callResultWrap{
		operations:  operations,
		identityMap: map[string]map[string]interface{}{},
		session:     session,
	}
	err = json.Unmarshal(response, &wrap)
	if err != nil {
		return nil, err
	}
	return wrap.results, nil
}

func (session *Session) call(operations ...interface{}) ([]byte, error) {
	url := fmt.Sprintf("%s%s", session.ServerUrl, session.ApiEndpoint)

	requestBody, err := session.encodeOperations(operations)

	if err != nil {
		return nil, err
	}

	request, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-type", "application/json")
	request.Header.Set("ftrack-api-key", session.ApiKey)
	request.Header.Set("ftrack-user", session.ApiUser)
	request.Header.Set("ftrack-Clienttoken", session.ClientToken)

	resp, err := newNetClient(session.Timeout).Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func IsEntity(entity interface{}) bool {
	if _, err := GetEntityType(entity); err == nil {
		return true
	}
	return false
}

func GetEntityType(entity interface{}) (string, error) {
	casted, ok := entity.(map[string]interface{})
	if !ok {
		return "", errors.New("not entity")
	}
	entityType, ok := casted[EntityTypeKey]
	if !ok {
		return "", errors.New("not entity")
	}
	entityTypeCasted, ok := entityType.(string)
	if !ok {
		return "", errors.New(fmt.Sprintf("entity type key '%s' contains invalid entity type '%s'", EntityTypeKey, entityType))
	}
	return entityTypeCasted, nil
}

func IsDate(data interface{}) bool {
	casted, ok := data.(map[string]interface{})
	if !ok {
		return false
	}
	if v, ok := casted["__type__"]; ok {
		if value := reflect.ValueOf(v); value.Kind() == reflect.String {
			return value.String() == "datetime"
		}
	}
	return false
}
