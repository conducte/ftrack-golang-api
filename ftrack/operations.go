package ftrack

import (
	"fmt"
	uuid "github.com/satori/go.uuid"
)

type QueryOperation struct {
	Action     string `json:"action"`
	Expression string `json:"expression"`
}

type QueryResult struct {
	Action   string                   `json:"action"`
	Data     []map[string]interface{} `json:"data"`
	Metadata map[string]interface{}   `json:"metadata"`
}

func NewQueryOperation(expression string) QueryOperation {
	return QueryOperation{
		Action:     "query",
		Expression: expression,
	}
}

func (op QueryOperation) ResultFactory(session *Session) *QueryResult {
	return &QueryResult{}
}

func (r *QueryResult) DecodeResult(session *Session, identityMap map[string]map[string]interface{}) {
	session.Decode(r.Data, identityMap)
}

type CreateOperation struct {
	Action     string                 `json:"action"`
	EntityType string                 `json:"entity_type"`
	EntityData map[string]interface{} `json:"entity_data"`
}

type CreateResult struct {
	Action   string                 `json:"action"`
	Data     map[string]interface{} `json:"data"`
	Metadata map[string]interface{} `json:"metadata"`
}

func NewCreateOperation(entityType string, data map[string]interface{}) CreateOperation {
	op := CreateOperation{
		Action:     "create",
		EntityType: entityType,
		EntityData: map[string]interface{}{},
	}
	for k, v := range data {
		op.EntityData[k] = v
	}
	if _, ok := op.EntityData[EntityTypeKey]; !ok {
		op.EntityData[EntityTypeKey] = fmt.Sprintf("%s", entityType)
	}
	return op
}

func (op CreateOperation) ResultFactory(session *Session) *CreateResult {
	return &CreateResult{}
}

func (r *CreateResult) DecodeResult(session *Session, identityMap map[string]map[string]interface{}) {
	session.Decode(r.Data, identityMap)
}

type UpdateOperation struct {
	Action     string                 `json:"action"`
	EntityType string                 `json:"entity_type"`
	EntityData map[string]interface{} `json:"entity_data"`
	EntityKey  []string               `json:"entity_key"`
}

type UpdateResult CreateResult

func NewUpdateOperation(entityType string, keys []string, data map[string]interface{}) UpdateOperation {
	op := UpdateOperation{
		Action:     "update",
		EntityType: entityType,
		EntityKey:  keys,
		EntityData: map[string]interface{}{},
	}
	for k, v := range data {
		op.EntityData[k] = v
	}
	if _, ok := op.EntityData[EntityTypeKey]; !ok {
		op.EntityData[EntityTypeKey] = entityType
	}
	return op
}

func (op UpdateOperation) ResultFactory(session *Session) *UpdateResult {
	return &UpdateResult{}
}

func (r *UpdateResult) DecodeResult(session *Session, identityMap map[string]map[string]interface{}) {
	session.Decode(r.Data, identityMap)
}

type DeleteOperation struct {
	Action     string   `json:"action"`
	EntityType string   `json:"entity_type"`
	EntityKey  []string `json:"entity_key"`
}

type DeleteResult struct {
	Action string `json:"action"`
	Data   bool   `json:"data"`
}

func NewDeleteOperation(entityType string, keys []string) DeleteOperation {
	return DeleteOperation{
		Action:     "delete",
		EntityType: entityType,
		EntityKey:  keys,
	}
}

func (op DeleteOperation) ResultFactory(session *Session) *DeleteResult {
	return &DeleteResult{}
}

func (r *DeleteResult) DecodeResult(session *Session, identityMap map[string]map[string]interface{}) {
}

type QueryInformationOperation struct {
	Action string   `json:"action"`
	Values []string `json:"values"`
}

type QueryInformationResult map[string]interface{}

func NewQueryInformationOperation(values []string) QueryInformationOperation {
	contains := func(a []string, e string) bool {
		for _, a := range a {
			if a == e {
				return true
			}
		}
		return false
	}
	if values != nil && !contains(values, "is_timezone_support_enabled") {
		values = append(values, "is_timezone_support_enabled")
	} else {
		values = []string{
			"is_timezone_support_enabled",
		}
	}
	return QueryInformationOperation{
		Action: "query_server_information",
		Values: values,
	}
}

func (op QueryInformationOperation) ResultFactory(session *Session) *QueryInformationResult {
	return &QueryInformationResult{}
}

func (r *QueryInformationResult) DecodeResult(session *Session, identityMap map[string]map[string]interface{}) {
}

type QuerySchemasOperation struct {
	Action string `json:"action"`
}

type QuerySchemasResult []map[string]interface{}

func NewQuerySchemasOperation() QuerySchemasOperation {
	return QuerySchemasOperation{
		Action: "query_schemas",
	}
}

func (op QuerySchemasOperation) ResultFactory(session *Session) *QuerySchemasResult {
	return &QuerySchemasResult{}
}

func (r *QuerySchemasResult) DecodeResult(session *Session, identityMap map[string]map[string]interface{}) {
}

type GetUploadMetadataOperation struct {
	Action      string    `json:"action"`
	FileName    string    `json:"file_name"`
	FileSize    int64     `json:"file_size"`
	ComponentId uuid.UUID `json:"component_id"`
}

type GetUploadMetadataResult struct {
	Url     string            `json:"url"`
	Headers map[string]string `json:"headers"`
}

func NewGetUploadMetadataOperation(fileName string, fileSize int64, componentId uuid.UUID) GetUploadMetadataOperation {
	return GetUploadMetadataOperation{
		Action:      "get_upload_metadata",
		FileName:    fileName,
		FileSize:    fileSize,
		ComponentId: componentId,
	}
}

func (op GetUploadMetadataOperation) ResultFactory(session *Session) *GetUploadMetadataResult {
	return &GetUploadMetadataResult{}
}

func (r *GetUploadMetadataResult) DecodeResult(session *Session, identityMap map[string]map[string]interface{}) {
}
