package scan

import (
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockFileInfo 是FileInfo接口的模拟实现
type MockFileInfo struct {
	mock.Mock
	key       string
	_size     int64
	_mtime    time.Time
	_isDir    bool
	_fileMode os.FileMode
	_atime    time.Time
	_ctime    time.Time
}

// BenchmarkConditionFilter_IsSatisfied 基准测试IsSatisfied方法的性能
func BenchmarkConditionFilter_IsSatisfied(b *testing.B) {
	// 创建包含==、in和>条件的过滤器
	conditions := []string{
		"name == 'test.txt'",
		"name in 'test'",
		"size > 100",
	}
	filter, err := NewConditionFilter(conditions)
	if err != nil {
		b.Fatalf("创建过滤器失败: %v", err)
	}

	// 创建测试文件信息
	fileInfo := &MockFileInfo{
		key:    "test.txt",
		_size:  200,
		_mtime: time.Now(),
		_isDir: false,
	}

	// 重置计时器
	b.ResetTimer()

	// 循环执行IsSatisfied方法b.N次
	for i := 0; i < b.N; i++ {
		filter.IsSatisfied(fileInfo)
	}
}

func (m *MockFileInfo) Key() string {
	return m.key
}

func (m *MockFileInfo) Size() int64 {
	return m._size
}

func (m *MockFileInfo) MTime() time.Time {
	return m._mtime
}

func (m *MockFileInfo) ATime() time.Time {
	return m._atime
}

func (m *MockFileInfo) CTime() time.Time {
	return m._ctime
}

func (m *MockFileInfo) Perm() os.FileMode {
	return m._fileMode
}

func (m *MockFileInfo) IsDir() bool {
	return m._isDir
}

func (m *MockFileInfo) IsSymlink() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockFileInfo) IsRegular() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockFileInfo) IsSticky() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockFileInfo) Get(offset, limit int64) (io.ReadCloser, error) {
	args := m.Called(offset, limit)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockFileInfo) Delete() error {
	args := m.Called()
	return args.Error(0)
}

// TestParseCondition 测试解析条件字符串
func TestParseCondition(t *testing.T) {
	cases := []struct {
		name      string
		condStr   string
		expected  Condition
		expectErr bool
	}{{
		name:    "字符串条件",
		condStr: "name == 'test.txt'",
		expected: Condition{
			Property: "name",
			Operator: "==",
			Value:    "test.txt",
		},
		expectErr: false,
	}, {
		name:    "like条件",
		condStr: "name like 'main%'",
		expected: Condition{
			Property: "name",
			Operator: "like",
			Value:    "main%",
		},
		expectErr: false,
	}, {
		name:    "数字条件",
		condStr: "size > 100",
		expected: Condition{
			Property: "size",
			Operator: ">",
			Value:    int64(100),
		},
		expectErr: false,
	}, {
		name:    "带单位的数字条件",
		condStr: "size <= 10K",
		expected: Condition{
			Property: "size",
			Operator: "<=",
			Value:    int64(10240), // 10 * 1024
		},
		expectErr: false,
	}, {
		name:    "时间条件",
		condStr: "modified < 24",
		expected: Condition{
			Property: "modified",
			Operator: "<",
			Value:    24 * time.Hour,
		},
		expectErr: false,
	}, {
		name:    "类型条件",
		condStr: "type == 'file'",
		expected: Condition{
			Property: "type",
			Operator: "==",
			Value:    "file",
		},
		expectErr: false,
	}, {
		name:      "无效的运算符",
		condStr:   "name contains 'test'",
		expectErr: true,
	}, {
		name:      "无效的属性",
		condStr:   "invalid > 100",
		expectErr: true,
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cond, err := parseCondition(tc.condStr)
			if tc.expectErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.expected.Property, cond.Property)
			assert.Equal(t, tc.expected.Operator, cond.Operator)
			assert.Equal(t, tc.expected.Value, cond.Value)
		})
	}
}

// TestParseSize 测试解析大小字符串
func TestParseSize(t *testing.T) {
	cases := []struct {
		name      string
		sizeStr   string
		expected  int64
		expectErr bool
	}{{
		name:      "纯数字",
		sizeStr:   "100",
		expected:  100,
		expectErr: false,
	}, {
		name:      "K单位",
		sizeStr:   "10K",
		expected:  10 * 1024,
		expectErr: false,
	}, {
		name:      "M单位",
		sizeStr:   "2M",
		expected:  2 * 1024 * 1024,
		expectErr: false,
	}, {
		name:      "G单位",
		sizeStr:   "3G",
		expected:  3 * 1024 * 1024 * 1024,
		expectErr: false,
	}, {
		name:      "带小数点",
		sizeStr:   "1.5K",
		expected:  1536, // 1.5 * 1024
		expectErr: false,
	}, {
		name:      "无效格式",
		sizeStr:   "10X",
		expectErr: true,
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			size, err := parseSize(tc.sizeStr)
			if tc.expectErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, size)
		})
	}
}

// TestMatchLike 测试like操作符的匹配功能
func TestMatchLike(t *testing.T) {
	cases := []struct {
		name     string
		value    string
		pattern  string
		expected bool
	}{{
		name:     "前缀匹配",
		value:    "main.go",
		pattern:  "main%",
		expected: true,
	}, {
		name:     "后缀匹配",
		value:    "app.main",
		pattern:  "%main",
		expected: true,
	}, {
		name:     "中间匹配",
		value:    "app.main.go",
		pattern:  "%main%",
		expected: true,
	}, {
		name:     "精确匹配",
		value:    "main",
		pattern:  "main",
		expected: true,
	}, {
		name:     "单字符匹配",
		value:    "main.go",
		pattern:  "main._o",
		expected: true,
	}, {
		name:     "不匹配",
		value:    "test.go",
		pattern:  "main%",
		expected: false,
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := matchString(tc.value, "like", tc.pattern)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestParseDuration 测试解析时间字符串
func TestParseDuration(t *testing.T) {
	cases := []struct {
		name      string
		durStr    string
		expected  time.Duration
		expectErr bool
	}{{
		name:      "整数小时",
		durStr:    "24",
		expected:  24 * time.Hour,
		expectErr: false,
	}, {
		name:      "小数小时",
		durStr:    "0.5",
		expected:  30 * time.Minute,
		expectErr: false,
	}, {
		name:      "无效格式",
		durStr:    "abc",
		expectErr: true,
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dur, err := parseDuration(tc.durStr)
			if tc.expectErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, dur)
		})
	}
}

// TestMatchString 测试字符串匹配
func TestMatchString(t *testing.T) {
	cases := []struct {
		name     string
		value    string
		operator string
		target   string
		expected bool
	}{{
		name:     "等于匹配",
		value:    "test.txt",
		operator: "==",
		target:   "test.txt",
		expected: true,
	}, {
		name:     "不等于匹配",
		value:    "test.txt",
		operator: "!=",
		target:   "example.txt",
		expected: true,
	}, {
		name:     "包含匹配",
		value:    "test.txt",
		operator: "in",
		target:   "test",
		expected: true,
	}, {
		name:     "大小写不敏感包含匹配",
		value:    "Test.txt",
		operator: "in",
		target:   "test",
		expected: true,
	}, {
		name:     "不包含匹配",
		value:    "example.txt",
		operator: "in",
		target:   "test",
		expected: false,
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := matchString(tc.value, tc.operator, tc.target)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestMatchNumber 测试数字匹配
func TestMatchNumber(t *testing.T) {
	cases := []struct {
		name     string
		value    int64
		operator string
		target   int64
		expected bool
	}{{
		name:     "等于匹配",
		value:    100,
		operator: "==",
		target:   100,
		expected: true,
	}, {
		name:     "不等于匹配",
		value:    100,
		operator: "!=",
		target:   200,
		expected: true,
	}, {
		name:     "大于匹配",
		value:    200,
		operator: ">",
		target:   100,
		expected: true,
	}, {
		name:     "小于匹配",
		value:    50,
		operator: "<",
		target:   100,
		expected: true,
	}, {
		name:     "大于等于匹配",
		value:    100,
		operator: ">=",
		target:   100,
		expected: true,
	}, {
		name:     "小于等于匹配",
		value:    100,
		operator: "<=",
		target:   100,
		expected: true,
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := matchNumber(tc.value, tc.operator, tc.target)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestMatchTime 测试时间匹配
func TestMatchTime(t *testing.T) {
	now := time.Now()
	past := now.Add(-24 * time.Hour)
	future := now.Add(24 * time.Hour)

	cases := []struct {
		name     string
		value    time.Time
		operator string
		target   time.Time
		expected bool
	}{{
		name:     "大于匹配",
		value:    future,
		operator: ">",
		target:   now,
		expected: true,
	}, {
		name:     "小于匹配",
		value:    past,
		operator: "<",
		target:   now,
		expected: true,
	}, {
		name:     "大于等于匹配",
		value:    now,
		operator: ">=",
		target:   now,
		expected: true,
	}, {
		name:     "小于等于匹配",
		value:    now,
		operator: "<=",
		target:   now,
		expected: true,
	}, {
		name:     "等于匹配",
		value:    now,
		operator: "==",
		target:   now,
		expected: true,
	}, {
		name:     "不等于匹配",
		value:    past,
		operator: "!=",
		target:   now,
		expected: true,
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := matchTime(tc.value, tc.operator, tc.target)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestConditionFilter_IsSatisfied 测试条件过滤器
func TestConditionFilter_IsSatisfied(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name       string
		conditions []string
		fileInfo   *MockFileInfo
		expected   bool
	}{{
		name: "单条件匹配-文件名",
		conditions: []string{
			"name == 'test.txt'",
		},
		fileInfo: &MockFileInfo{
			key: "test.txt",
		},
		expected: true,
	}, {
		name: "单条件不匹配-文件名",
		conditions: []string{
			"name == 'test.txt'",
		},
		fileInfo: &MockFileInfo{
			key: "example.txt",
		},
		expected: false,
	}, {
		name: "多条件匹配-文件名和大小",
		conditions: []string{
			"name == 'test.txt'",
			"size > 100",
		},
		fileInfo: &MockFileInfo{
			key:   "test.txt",
			_size: 200,
		},
		expected: true,
	}, {
		name: "多条件不匹配-大小不满足",
		conditions: []string{
			"name == 'test.txt'",
			"size > 100",
		},
		fileInfo: &MockFileInfo{
			key:   "test.txt",
			_size: 50,
		},
		expected: false,
	}, {
		name: "时间条件匹配",
		conditions: []string{
			"modified > 24",
		},
		fileInfo: &MockFileInfo{
			_mtime: now.Add(-12 * time.Hour), // 12小时前，小于24小时
		},
		expected: true,
	}, {
		name: "时间条件不匹配",
		conditions: []string{
			"modified > 24",
		},
		fileInfo: &MockFileInfo{
			_mtime: now.Add(-36 * time.Hour), // 36小时前，大于24小时
		},
		expected: false,
	}, {
		name: "类型条件匹配-文件",
		conditions: []string{
			"type == 'file'",
		},
		fileInfo: &MockFileInfo{
			_isDir: false,
		},
		expected: true,
	}, {
		name: "类型条件匹配-目录",
		conditions: []string{
			"type == 'dir'",
		},
		fileInfo: &MockFileInfo{
			_isDir: true,
		},
		expected: true,
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			filter, err := NewConditionFilter(tc.conditions)
			assert.NoError(t, err)
			result := filter.IsSatisfied(tc.fileInfo)
			assert.Equal(t, tc.expected, result)
		})
	}
}
