package utils

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// ImportExportError 导入导出错误类型
type ImportExportError struct {
	Message string
	Line    int
	Field   string
}

func (e *ImportExportError) Error() string {
	if e.Line > 0 && e.Field != "" {
		return fmt.Sprintf("第%d行字段'%s'错误: %s", e.Line, e.Field, e.Message)
	}
	if e.Line > 0 {
		return fmt.Sprintf("第%d行错误: %s", e.Line, e.Message)
	}
	return e.Message
}

// ImportConfig 导入配置
type ImportConfig struct {
	FileType    string // "csv", "excel", "json"
	HasHeader   bool   // 是否有表头
	StartRow    int    // 数据开始行 (excel从1开始)
	SheetName   string // Excel工作表名称
	DateFormat  string // 日期格式
	TimeFormat  string // 时间格式
}

// ExportConfig 导出配置
type ExportConfig struct {
	FileType    string            // "csv", "excel", "json"
	FileName    string            // 文件名
	Headers     []string          // 表头
	FieldMap    map[string]string // 字段映射: struct字段名 -> 导出列名
	DateFormat  string            // 日期格式
	TimeFormat  string            // 时间格式
}

// ImportResult 导入结果
type ImportResult struct {
	TotalRows   int                 // 总行数
	SuccessRows int                 // 成功行数
	FailedRows  int                 // 失败行数
	Errors      []*ImportExportError // 错误详情
	Data        interface{}         // 导入的数据
}

// ImportData 通用数据导入函数
// dataPtr: 指向目标切片数据的指针，如 &[]User{}
// file: 上传的文件
// config: 导入配置
func ImportData(dataPtr interface{}, file *multipart.FileHeader, config *ImportConfig) (*ImportResult, error) {
	if dataPtr == nil || reflect.ValueOf(dataPtr).Kind() != reflect.Ptr {
		return nil, &ImportExportError{Message: "dataPtr必须是指向切片的指针"}
	}

	sliceType := reflect.TypeOf(dataPtr).Elem()
	if sliceType.Kind() != reflect.Slice {
		return nil, &ImportExportError{Message: "dataPtr必须指向切片类型"}
	}

	elemType := sliceType.Elem()
	if elemType.Kind() != reflect.Struct {
		return nil, &ImportExportError{Message: "切片元素必须是结构体类型"}
	}

	// 打开文件
	fileReader, err := file.Open()
	if err != nil {
		return nil, &ImportExportError{Message: "打开文件失败: " + err.Error()}
	}
	defer fileReader.Close()

	result := &ImportResult{
		TotalRows:   0,
		SuccessRows: 0,
		FailedRows:  0,
		Errors:      make([]*ImportExportError, 0),
	}

	switch strings.ToLower(config.FileType) {
	case "csv":
		err = importFromCSV(dataPtr, fileReader, config, result)
	case "excel":
		err = importFromExcel(dataPtr, fileReader, config, result)
	case "json":
		err = importFromJSON(dataPtr, fileReader, config, result)
	default:
		err = &ImportExportError{Message: "不支持的文件类型: " + config.FileType}
	}

	return result, err
}

// ExportData 通用数据导出函数
// data: 要导出的数据切片
// config: 导出配置
func ExportData(data interface{}, config *ExportConfig) ([]byte, error) {
	if data == nil {
		return nil, &ImportExportError{Message: "导出数据不能为空"}
	}

	dataValue := reflect.ValueOf(data)
	if dataValue.Kind() != reflect.Slice {
		return nil, &ImportExportError{Message: "导出数据必须是切片类型"}
	}

	if dataValue.Len() == 0 {
		return nil, &ImportExportError{Message: "导出数据为空"}
	}

	switch strings.ToLower(config.FileType) {
	case "csv":
		return exportToCSV(data, config)
	case "excel":
		return exportToExcel(data, config)
	case "json":
		return exportToJSON(data, config)
	default:
		return nil, &ImportExportError{Message: "不支持的文件类型: " + config.FileType}
	}
}

// importFromCSV CSV导入实现
func importFromCSV(dataPtr interface{}, reader io.Reader, config *ImportConfig, result *ImportResult) error {
	csvReader := csv.NewReader(reader)
	csvReader.FieldsPerRecord = -1 // 允许字段数量不一致

	lineNum := 0
	for {
		lineNum++
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			result.Errors = append(result.Errors, &ImportExportError{
				Line:    lineNum,
				Message: "读取CSV行失败: " + err.Error(),
			})
			result.FailedRows++
			continue
		}

		// 跳过表头
		if config.HasHeader && lineNum == 1 {
			continue
		}

		// 跳过开始行之前的数据
		if lineNum < config.StartRow {
			continue
		}

		result.TotalRows++
		if err := parseCSVRecord(dataPtr, record, lineNum, config, result); err != nil {
			result.FailedRows++
		} else {
			result.SuccessRows++
		}
	}

	return nil
}

// importFromExcel Excel导入实现
func importFromExcel(dataPtr interface{}, reader io.Reader, config *ImportConfig, result *ImportResult) error {
	file, err := excelize.OpenReader(reader)
	if err != nil {
		return &ImportExportError{Message: "打开Excel文件失败: " + err.Error()}
	}
	defer file.Close()

	sheetName := config.SheetName
	if sheetName == "" {
		sheetName = file.GetSheetName(0)
	}

	rows, err := file.GetRows(sheetName)
	if err != nil {
		return &ImportExportError{Message: "读取Excel工作表失败: " + err.Error()}
	}

	for i, row := range rows {
		lineNum := i + 1

		// 跳过表头
		if config.HasHeader && lineNum == 1 {
			continue
		}

		// 跳过开始行之前的数据
		if lineNum < config.StartRow {
			continue
		}

		result.TotalRows++
		if err := parseCSVRecord(dataPtr, row, lineNum, config, result); err != nil {
			result.FailedRows++
		} else {
			result.SuccessRows++
		}
	}

	return nil
}

// importFromJSON JSON导入实现
func importFromJSON(dataPtr interface{}, reader io.Reader, config *ImportConfig, result *ImportResult) error {
	dataValue := reflect.ValueOf(dataPtr).Elem()
	elemType := dataValue.Type().Elem()

	var dataSlice []interface{}
	if err := json.NewDecoder(reader).Decode(&dataSlice); err != nil {
		return &ImportExportError{Message: "解析JSON失败: " + err.Error()}
	}

	for i, item := range dataSlice {
		lineNum := i + 1
		result.TotalRows++

		// 将interface{}转换为目标结构体
		itemJSON, err := json.Marshal(item)
		if err != nil {
			result.Errors = append(result.Errors, &ImportExportError{
				Line:    lineNum,
				Message: "序列化数据失败: " + err.Error(),
			})
			result.FailedRows++
			continue
		}

		newElem := reflect.New(elemType).Interface()
		if err := json.Unmarshal(itemJSON, newElem); err != nil {
			result.Errors = append(result.Errors, &ImportExportError{
				Line:    lineNum,
				Message: "反序列化数据失败: " + err.Error(),
			})
			result.FailedRows++
			continue
		}

		dataValue.Set(reflect.Append(dataValue, reflect.ValueOf(newElem).Elem()))
		result.SuccessRows++
	}

	return nil
}

// parseCSVRecord 解析CSV/Excel记录到结构体
func parseCSVRecord(dataPtr interface{}, record []string, lineNum int, config *ImportConfig, result *ImportResult) error {
	dataValue := reflect.ValueOf(dataPtr).Elem()
	elemType := dataValue.Type().Elem()
	newElem := reflect.New(elemType).Elem()

	for i := 0; i < newElem.NumField(); i++ {
		field := newElem.Field(i)
		fieldType := elemType.Field(i)

		// 跳过非导出字段
		if !field.CanSet() {
			continue
		}

		// 检查记录长度
		if i >= len(record) {
			break
		}

		value := strings.TrimSpace(record[i])
		if value == "" {
			continue
		}

		if err := setFieldValue(field, fieldType.Type, value, config); err != nil {
			result.Errors = append(result.Errors, &ImportExportError{
				Line:    lineNum,
				Field:   fieldType.Name,
				Message: err.Error(),
			})
			return err
		}
	}

	dataValue.Set(reflect.Append(dataValue, newElem))
	return nil
}

// setFieldValue 设置字段值
func setFieldValue(field reflect.Value, fieldType reflect.Type, value string, config *ImportConfig) error {
	switch fieldType.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intVal, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("整数转换失败: %s", err.Error())
		}
		field.SetInt(intVal)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintVal, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return fmt.Errorf("无符号整数转换失败: %s", err.Error())
		}
		field.SetUint(uintVal)
	case reflect.Float32, reflect.Float64:
		floatVal, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("浮点数转换失败: %s", err.Error())
		}
		field.SetFloat(floatVal)
	case reflect.Bool:
		boolVal, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("布尔值转换失败: %s", err.Error())
		}
		field.SetBool(boolVal)
	case reflect.Struct:
		if fieldType == reflect.TypeOf(time.Time{}) {
			// 尝试多种日期格式
			var timeVal time.Time
			var err error

			if config.DateFormat != "" {
				timeVal, err = time.Parse(config.DateFormat, value)
			} else {
				// 默认格式
				formats := []string{
					"2006-01-02",
					"2006/01/02",
					"2006-01-02 15:04:05",
					"2006/01/02 15:04:05",
					time.RFC3339,
				}

				for _, format := range formats {
					timeVal, err = time.Parse(format, value)
					if err == nil {
						break
					}
				}
			}

			if err != nil {
				return fmt.Errorf("日期时间转换失败: %s", err.Error())
			}
			field.Set(reflect.ValueOf(timeVal))
		}
	default:
		return fmt.Errorf("不支持的字段类型: %s", fieldType.Kind())
	}

	return nil
}

// exportToCSV CSV导出实现
func exportToCSV(data interface{}, config *ExportConfig) ([]byte, error) {
	var buf strings.Builder
	csvWriter := csv.NewWriter(&buf)

	// 写入表头
	if len(config.Headers) > 0 {
		if err := csvWriter.Write(config.Headers); err != nil {
			return nil, err
		}
	}

	dataValue := reflect.ValueOf(data)
	for i := 0; i < dataValue.Len(); i++ {
		record := make([]string, 0)
		elem := dataValue.Index(i)

		for j := 0; j < elem.NumField(); j++ {
			field := elem.Field(j)
			fieldType := elem.Type().Field(j)

			// 跳过非导出字段
			if !field.CanInterface() {
				continue
			}

			value := formatFieldValue(field, fieldType.Type, config)
			record = append(record, value)
		}

		if err := csvWriter.Write(record); err != nil {
			return nil, err
		}
	}

	csvWriter.Flush()
	return []byte(buf.String()), nil
}

// exportToExcel Excel导出实现
func exportToExcel(data interface{}, config *ExportConfig) ([]byte, error) {
	file := excelize.NewFile()
	sheetName := "Sheet1"

	// 写入表头
	if len(config.Headers) > 0 {
		for i, header := range config.Headers {
			cell, _ := excelize.CoordinatesToCellName(i+1, 1)
			file.SetCellValue(sheetName, cell, header)
		}
	}

	dataValue := reflect.ValueOf(data)
	for i := 0; i < dataValue.Len(); i++ {
		rowNum := i + 2 // 从第2行开始
		elem := dataValue.Index(i)
		colNum := 1

		for j := 0; j < elem.NumField(); j++ {
			field := elem.Field(j)
			fieldType := elem.Type().Field(j)

			// 跳过非导出字段
			if !field.CanInterface() {
				continue
			}

			cell, _ := excelize.CoordinatesToCellName(colNum, rowNum)
			value := formatFieldValue(field, fieldType.Type, config)
			file.SetCellValue(sheetName, cell, value)
			colNum++
		}
	}

	buffer, err := file.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// exportToJSON JSON导出实现
func exportToJSON(data interface{}, config *ExportConfig) ([]byte, error) {
	return json.MarshalIndent(data, "", "  ")
}

// formatFieldValue 格式化字段值
func formatFieldValue(field reflect.Value, fieldType reflect.Type, config *ExportConfig) string {
	if !field.IsValid() {
		return ""
	}

	switch fieldType.Kind() {
	case reflect.String:
		return field.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(field.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(field.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(field.Float(), 'f', -1, 64)
	case reflect.Bool:
		return strconv.FormatBool(field.Bool())
	case reflect.Struct:
		if fieldType == reflect.TypeOf(time.Time{}) {
			timeVal := field.Interface().(time.Time)
			if config.DateFormat != "" {
				return timeVal.Format(config.DateFormat)
			}
			return timeVal.Format("2006-01-02 15:04:05")
		}
		// 其他结构体转为JSON字符串
		if jsonBytes, err := json.Marshal(field.Interface()); err == nil {
			return string(jsonBytes)
		}
	}

	return fmt.Sprintf("%v", field.Interface())
}