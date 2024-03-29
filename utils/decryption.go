package utils

import (
	"encoding/base64"
)

func DecryptBase64DynamicMap(data map[string]any) map[string]any {
	for key, value := range data {
		switch value := value.(type) {
		case map[string]any:
			data[key] = DecryptBase64DynamicMap(value)
		case map[string]string:
			data[key] = DecryptBase64StringMap(value)
		case []byte:
			data[key] = DecryptBase64(string(value))
		case []map[string]any:
			decryptedArray := []map[string]any{}
			for _, element := range value {
				decryptedArray = append(decryptedArray, DecryptBase64DynamicMap(element))
			}

			data[key] = decryptedArray
		case []any:
			decryptedArray := []any{}
			for _, element := range value {
				switch element := element.(type) {
				case map[string]any:
					decryptedArray = append(decryptedArray, DecryptBase64DynamicMap(element))
				case map[string]string:
					decryptedArray = append(decryptedArray, DecryptBase64StringMap(element))
				case string:
					decryptedArray = append(decryptedArray, DecryptBase64(element))
				case []byte:
					decryptedArray = append(decryptedArray, DecryptBase64(string(element)))
				default:
					decryptedArray = append(decryptedArray, element)
				}
			}

			data[key] = decryptedArray
		case []string:
			decryptedArray := []string{}
			for _, element := range value {
				decryptedArray = append(decryptedArray, DecryptBase64(element))
			}

			data[key] = decryptedArray
		case string:
			data[key] = DecryptBase64(value)
		}
	}
	return data
}

func DecryptBase64StringMap(data map[string]string) map[string]string {
	for key, value := range data {
		data[key] = DecryptBase64(value)
	}

	return data
}

func DecryptBase64(value string) string {
	decodedData, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return value
	}

	return string(decodedData)
}

func OperateOnDynamicMap(data map[string]any, operation func(string) string) map[string]any {
	for key, value := range data {
		switch value := value.(type) {
		case map[string]any:
			data[key] = OperateOnDynamicMap(value, operation)
		case map[string]string:
			data[key] = OperateOnStringMap(value, operation)
		case []map[string]any:
			decryptedArray := []map[string]any{}
			for _, element := range value {
				decryptedArray = append(decryptedArray, OperateOnDynamicMap(element, operation))
			}

			data[key] = decryptedArray
		case []byte:
			data[key] = operation(string(value))
		case []any:
			decryptedArray := []any{}
			for _, element := range value {
				switch element := element.(type) {
				case map[string]any:
					decryptedArray = append(decryptedArray, OperateOnDynamicMap(element, operation))
				case map[string]string:
					decryptedArray = append(decryptedArray, OperateOnStringMap(element, operation))
				case string:
					decryptedArray = append(decryptedArray, operation(element))
				case []byte:
					decryptedArray = append(decryptedArray, operation(string(element)))
				default:
					decryptedArray = append(decryptedArray, element)
				}
			}

			data[key] = decryptedArray
		case []string:
			decryptedArray := []string{}
			for _, element := range value {
				decryptedArray = append(decryptedArray, operation(element))
			}

			data[key] = decryptedArray
		case string:
			data[key] = operation(value)
		}
	}
	return data
}

func OperateOnStringMap(data map[string]string, operation func(string) string) map[string]string {
	for key, value := range data {
		data[key] = operation(value)
	}

	return data
}
