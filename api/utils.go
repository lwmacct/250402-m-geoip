package api

import (
	"reflect"
	"strings"
)

// getModelFields 从模型结构体中提取字段名称
func getModelFields(model interface{}) []string {
	fields := []string{}

	// 获取结构体类型
	t := reflect.TypeOf(model)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// 遍历所有字段
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// 处理嵌入式字段（如gorm.Model）
		if field.Anonymous {
			// 如果字段类型是结构体
			if field.Type.Kind() == reflect.Struct {
				// 递归获取嵌入式结构体的字段
				embeddedFields := getModelFields(reflect.New(field.Type).Elem().Interface())
				fields = append(fields, embeddedFields...)
			}
			continue
		}

		// 获取标签中的列名
		tagValue := field.Tag.Get("gorm")
		columnName := ""
		for _, tag := range strings.Split(tagValue, ";") {
			if strings.HasPrefix(tag, "column:") {
				columnName = strings.TrimPrefix(tag, "column:")
				break
			}
		}

		// 如果没有指定列名，使用字段名小写
		if columnName == "" {
			columnName = strings.ToLower(field.Name)
		}

		fields = append(fields, columnName)
	}

	return fields
}
