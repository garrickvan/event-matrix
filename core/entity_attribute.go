// Copyright 2025 eventmatrix.cn
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package core

import (
	"strings"

	"github.com/garrickvan/event-matrix/utils/jsonx"
	"github.com/spf13/cast"
)

type EntityAttribute struct {
	ID           string `json:"id" gorm:"primaryKey"`
	EntityID     string `json:"entityId" gorm:"index"`
	Name         string `json:"name"`
	Code         string `json:"code" gorm:"index"`
	FieldType    string `json:"fieldType"`
	ValueSource  string `json:"valueSource"`
	DefaultValue string `json:"defaultValue"`
	Unique       bool   `json:"unique"`
	Indexed      bool   `json:"indexed"`
	IsSecrecy    bool   `json:"isSecrecy"` // 保密字段查询时不返回
	UpdatedAt    int64  `json:"updatedAt"`
	CreatedAt    int64  `json:"createdAt"`
	DeletedAt    int64  `json:"deletedAt" gorm:"index"`
	DeletedBy    string `json:"deletedBy"`
	Creator      string `json:"creator"`
}

type FIELD_TYPE string

const (
	ID_FIELD_TYPE        FIELD_TYPE = "id"
	REF_FIELD_TYPE       FIELD_TYPE = "ref"
	STRING_FIELD_TYPE    FIELD_TYPE = "string"
	TEXT_FIELD_TYPE      FIELD_TYPE = "text"
	INT8_FIELD_TYPE      FIELD_TYPE = "int8"
	INT32_FIELD_TYPE     FIELD_TYPE = "int32"
	INT64_FIELD_TYPE     FIELD_TYPE = "int64"
	FLOAT32_FIELD_TYPE   FIELD_TYPE = "float32"
	FLOAT64_FIELD_TYPE   FIELD_TYPE = "float64"
	BOOLEAN_FIELD_TYPE   FIELD_TYPE = "boolean"
	DATETIME_FIELD_TYPE  FIELD_TYPE = "datetime"
	CONSTANT_FIELD_TYPE  FIELD_TYPE = "constant"
	UID_FIELD_TYPE       FIELD_TYPE = "uid"
	URL_FIELD_TYPE       FIELD_TYPE = "url"
	EMAIL_FIELD_TYPE     FIELD_TYPE = "email"
	PHONE_FIELD_TYPE     FIELD_TYPE = "phone"
	CUSTOM_FIELD_TYPE    FIELD_TYPE = "custom"
	AND_QUERY_FIELD_TYPE FIELD_TYPE = "and_query"
	OR_QUERY_FIELD_TYPE  FIELD_TYPE = "or_query"
	ORDER_BY_FIELD_TYPE  FIELD_TYPE = "order_by"
)

func (e *EntityAttribute) GetDefaultVal() interface{} {
	if e.DefaultValue == "" {
		return nil
	}

	switch FIELD_TYPE(e.FieldType) {
	case ID_FIELD_TYPE, REF_FIELD_TYPE, STRING_FIELD_TYPE, TEXT_FIELD_TYPE, UID_FIELD_TYPE, URL_FIELD_TYPE, EMAIL_FIELD_TYPE, PHONE_FIELD_TYPE:
		return e.DefaultValue
	case INT8_FIELD_TYPE, INT32_FIELD_TYPE, INT64_FIELD_TYPE:
		return cast.ToInt64(e.DefaultValue)
	case FLOAT32_FIELD_TYPE, FLOAT64_FIELD_TYPE:
		return cast.ToFloat64(e.DefaultValue)
	case BOOLEAN_FIELD_TYPE:
		return cast.ToBool(e.DefaultValue)
	case DATETIME_FIELD_TYPE:
		return cast.ToInt64(e.DefaultValue)
	default:
		return e.DefaultValue
	}
}

func (e *EntityAttribute) FixValue(v interface{}) interface{} {
	return FixAttributeValue(v, e.FieldType)
}

func FixAttributeValue(v interface{}, typz string) interface{} {
	if v == nil {
		return nil
	}

	switch FIELD_TYPE(typz) {
	case ID_FIELD_TYPE, REF_FIELD_TYPE, STRING_FIELD_TYPE, TEXT_FIELD_TYPE, UID_FIELD_TYPE, URL_FIELD_TYPE, EMAIL_FIELD_TYPE, PHONE_FIELD_TYPE:
		return cast.ToString(v)
	case INT8_FIELD_TYPE, INT32_FIELD_TYPE, INT64_FIELD_TYPE:
		return cast.ToInt64(v)
	case FLOAT32_FIELD_TYPE, FLOAT64_FIELD_TYPE:
		return cast.ToFloat64(v)
	case BOOLEAN_FIELD_TYPE:
		return cast.ToBool(v)
	case DATETIME_FIELD_TYPE:
		return cast.ToInt64(v)
	default:
		return v
	}
}

func FindAttrFromArray(code string, attrs []EntityAttribute) *EntityAttribute {
	if attrs == nil {
		return nil
	}

	for _, attr := range attrs {
		if strings.EqualFold(attr.Code, code) {
			return &attr
		}
	}
	return nil
}

func NewEntityAttributeFromJson(v string) *EntityAttribute {
	var data EntityAttribute
	err := jsonx.UnmarshalFromBytes([]byte(v), &data)
	if err != nil {
		return &EntityAttribute{}
	}
	return &data
}

func NewEntityAttributeFromMap(v interface{}) *EntityAttribute {
	data, ok := v.(map[string]interface{})
	if !ok {
		return &EntityAttribute{}
	}

	return &EntityAttribute{
		ID:           cast.ToString(data["id"]),
		EntityID:     cast.ToString(data["entityId"]),
		Name:         cast.ToString(data["name"]),
		Code:         cast.ToString(data["code"]),
		FieldType:    cast.ToString(data["fieldType"]),
		ValueSource:  cast.ToString(data["valueSource"]),
		DefaultValue: cast.ToString(data["defaultValue"]),
		Unique:       cast.ToBool(data["unique"]),
		Indexed:      cast.ToBool(data["indexed"]),
		IsSecrecy:    cast.ToBool(data["isSecrecy"]),
		UpdatedAt:    cast.ToInt64(data["updatedAt"]),
		CreatedAt:    cast.ToInt64(data["createdAt"]),
		DeletedAt:    cast.ToInt64(data["deletedAt"]),
		DeletedBy:    cast.ToString(data["deletedBy"]),
		Creator:      cast.ToString(data["creator"]),
	}
}

func (e *EntityAttribute) Clone() *EntityAttribute {
	if e == nil {
		return &EntityAttribute{}
	}
	return &EntityAttribute{
		ID:           e.ID,
		EntityID:     e.EntityID,
		Name:         e.Name,
		Code:         e.Code,
		FieldType:    e.FieldType,
		ValueSource:  e.ValueSource,
		DefaultValue: e.DefaultValue,
		Unique:       e.Unique,
		Indexed:      e.Indexed,
		IsSecrecy:    e.IsSecrecy,
		UpdatedAt:    e.UpdatedAt,
		CreatedAt:    e.CreatedAt,
		DeletedAt:    e.DeletedAt,
		DeletedBy:    e.DeletedBy,
		Creator:      e.Creator,
	}
}
