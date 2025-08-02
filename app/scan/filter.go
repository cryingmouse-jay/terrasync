package scan

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"terrasync/object"
	"time"
)

// Condition 表示单个过滤条件
type Condition struct {
	Property string      // 属性名(name, size, modified等)
	Operator string      // 运算符(==, !=, >, <, >=, <=, in)
	Value    interface{} // 值
}

// ConditionFilter 实现Filter接口，用于基于多个条件过滤文件
type ConditionFilter struct {
	conditions []Condition
}

// NewConditionFilter 创建一个新的条件过滤器
func NewConditionFilter(conditions []string) (*ConditionFilter, error) {
	filter := &ConditionFilter{}
	for _, condStr := range conditions {
		cond, err := parseCondition(condStr)
		if err != nil {
			return nil, fmt.Errorf("解析条件失败: %s, 错误: %v", condStr, err)
		}
		filter.conditions = append(filter.conditions, cond)
	}
	return filter, nil
}

func (f *ConditionFilter) IsSatisfied(fileInfo object.FileInfo) bool {
	for _, cond := range f.conditions {
		if !f.matchCondition(fileInfo, cond) {
			return false // 任何一个条件不满足则跳过
		}
	}
	return true // 所有条件都满足则不跳过
}

// 解析单个条件字符串
func parseCondition(condStr string) (Condition, error) {
	// 支持的运算符，按优先级排序
	operators := []string{">=", "<=", "==", "!=", "in", "like", ">", "<"}

	opRegex := regexp.MustCompile(`\s*(` + strings.Join(operators, "|") + `)\s*`)

	// 查找运算符位置
	matches := opRegex.FindStringSubmatch(condStr)
	if len(matches) == 0 {
		return Condition{}, fmt.Errorf("找不到有效的运算符: %s", condStr)
	}

	// 分割属性名和值
	operator := matches[1]
	parts := opRegex.Split(condStr, 2)
	if len(parts) != 2 {
		return Condition{}, fmt.Errorf("条件格式错误: %s", condStr)
	}

	property := strings.TrimSpace(parts[0])
	valueStr := strings.TrimSpace(parts[1])

	// 解析值根据属性类型
	var value interface{}
	var err error

	switch strings.ToLower(property) {
	case "name", "type", "path":
		// 字符串类型(去除引号)
		value = strings.Trim(valueStr, "'\"")
	case "size":
		// 大小类型(支持K, M, G)
		value, err = parseSize(valueStr)
	case "modified":
		// 时间类型(小时)
		value, err = parseDuration(valueStr)
	default:
		return Condition{}, fmt.Errorf("不支持的属性: %s", property)
	}

	if err != nil {
		return Condition{}, err
	}

	return Condition{
		Property: strings.ToLower(property),
		Operator: strings.ToLower(operator),
		Value:    value,
	}, nil
}

// 解析大小字符串(如: 100, 10K, 2M, 3G)
func parseSize(sizeStr string) (int64, error) {
	sizeStr = strings.TrimSpace(sizeStr)
	multipliers := map[string]int64{
		"k": 1 << 10,
		"m": 1 << 20,
		"g": 1 << 30,
		"t": 1 << 40,
	}

	// 提取数字和单位
	re := regexp.MustCompile(`^([0-9.]+)([kmgKMGtT]?)$`)
	matches := re.FindStringSubmatch(sizeStr)
	if len(matches) < 2 {
		return 0, fmt.Errorf("无效的大小格式: %s", sizeStr)
	}

	// 解析数值
	num, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, err
	}

	// 应用单位乘数
	unit := strings.ToLower(matches[2])
	multiplier := int64(1)
	if unit != "" {
		multiplier = multipliers[unit]
		if multiplier == 0 {
			return 0, fmt.Errorf("无效的单位: %s", unit)
		}
	}

	return int64(num * float64(multiplier)), nil
}

// 解析时间字符串(如: 0.5, 24 表示小时)
func parseDuration(durStr string) (time.Duration, error) {
	durStr = strings.TrimSpace(durStr)
	hours, err := strconv.ParseFloat(durStr, 64)
	if err != nil {
		return 0, err
	}
	return time.Duration(hours * float64(time.Hour)), nil
}

// 匹配单个条件
func (f *ConditionFilter) matchCondition(fileInfo object.FileInfo, cond Condition) bool {
	now := time.Now()

	switch cond.Property {
	case "name":
		fileName := filepath.Base(fileInfo.Key())
		return matchString(fileName, cond.Operator, cond.Value.(string))
	case "path":
		filePath := fileInfo.Key()
		return matchString(filePath, cond.Operator, cond.Value.(string))
	case "size":
		fileSize := fileInfo.Size()
		return matchNumber(fileSize, cond.Operator, cond.Value.(int64))
	case "modified":
		modifiedTime := fileInfo.MTime()
		duration := cond.Value.(time.Duration)
		return matchTime(modifiedTime, cond.Operator, now.Add(-duration))
	case "type":
		fileType := "file"
		if fileInfo.IsDir() {
			fileType = "dir"
		}
		return matchString(fileType, cond.Operator, cond.Value.(string))
	default:
		return false
	}
}

// 字符串匹配
func matchString(value string, operator string, target string) bool {
	switch operator {
	case "==":
		return value == target
	case "!=":
		return value != target
	case "in":
		return strings.Contains(strings.ToLower(value), strings.ToLower(target))
	case "like":
		// 将like模式转换为正则表达式
		// % 匹配任意数量的字符
		// _ 匹配单个字符
		regexPattern := strings.Replace(target, "%", ".*", -1)
		regexPattern = strings.Replace(regexPattern, "_", ".", -1)
		regexPattern = "^" + regexPattern + "$"
		matched, err := regexp.MatchString(regexPattern, value)
		if err != nil {
			return false
		}
		return matched
	default:
		return false
	}
}

// 数字匹配
func matchNumber(value int64, operator string, target int64) bool {
	switch operator {
	case "==":
		return value == target
	case "!=":
		return value != target
	case ">":
		return value > target
	case "<":
		return value < target
	case ">=":
		return value >= target
	case "<=":
		return value <= target
	default:
		return false
	}
}

// 时间匹配
func matchTime(value time.Time, operator string, target time.Time) bool {
	switch operator {
	case ">":
		return value.After(target)
	case "<":
		return value.Before(target)
	case ">=":
		return !value.Before(target)
	case "<=":
		return !value.After(target)
	case "==":
		return value.Equal(target)
	case "!=":
		return !value.Equal(target)
	default:
		return false
	}
}
