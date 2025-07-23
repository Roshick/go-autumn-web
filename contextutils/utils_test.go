package contextutils

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestStruct struct {
	Name  string
	Value int
}

func TestWithValue(t *testing.T) {
	ctx := context.Background()
	testValue := TestStruct{Name: "test", Value: 42}

	newCtx := WithValue(ctx, testValue)

	assert.NotNil(t, newCtx)
	assert.NotEqual(t, ctx, newCtx)
}

func TestGetValue(t *testing.T) {
	t.Run("value exists", func(t *testing.T) {
		ctx := context.Background()
		testValue := TestStruct{Name: "test", Value: 42}

		ctxWithValue := WithValue(ctx, testValue)
		result := GetValue[TestStruct](ctxWithValue)

		require.NotNil(t, result)
		assert.Equal(t, testValue, *result)
	})

	t.Run("value does not exist", func(t *testing.T) {
		ctx := context.Background()
		result := GetValue[TestStruct](ctx)

		assert.Nil(t, result)
	})

	t.Run("different types", func(t *testing.T) {
		ctx := context.Background()
		testString := "hello"
		testInt := 123

		ctxWithString := WithValue(ctx, testString)
		ctxWithBoth := WithValue(ctxWithString, testInt)

		stringResult := GetValue[string](ctxWithBoth)
		intResult := GetValue[int](ctxWithBoth)

		require.NotNil(t, stringResult)
		require.NotNil(t, intResult)
		assert.Equal(t, testString, *stringResult)
		assert.Equal(t, testInt, *intResult)
	})
}

func TestMustGetValue(t *testing.T) {
	t.Run("value exists", func(t *testing.T) {
		ctx := context.Background()
		testValue := TestStruct{Name: "test", Value: 42}

		ctxWithValue := WithValue(ctx, testValue)
		result := MustGetValue[TestStruct](ctxWithValue)

		assert.Equal(t, testValue, result)
	})

	t.Run("value does not exist - should panic", func(t *testing.T) {
		ctx := context.Background()

		assert.Panics(t, func() {
			MustGetValue[TestStruct](ctx)
		})
	})
}

func TestGenericTypeSupport(t *testing.T) {
	ctx := context.Background()

	// Test with various types
	testCases := []struct {
		name  string
		value interface{}
	}{
		{"string", "test string"},
		{"int", 42},
		{"bool", true},
		{"struct", TestStruct{Name: "test", Value: 100}},
		{"slice", []string{"a", "b", "c"}},
		{"map", map[string]int{"key": 123}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			switch v := tc.value.(type) {
			case string:
				ctxWithValue := WithValue(ctx, v)
				result := GetValue[string](ctxWithValue)
				require.NotNil(t, result)
				assert.Equal(t, v, *result)
			case int:
				ctxWithValue := WithValue(ctx, v)
				result := GetValue[int](ctxWithValue)
				require.NotNil(t, result)
				assert.Equal(t, v, *result)
			case bool:
				ctxWithValue := WithValue(ctx, v)
				result := GetValue[bool](ctxWithValue)
				require.NotNil(t, result)
				assert.Equal(t, v, *result)
			case TestStruct:
				ctxWithValue := WithValue(ctx, v)
				result := GetValue[TestStruct](ctxWithValue)
				require.NotNil(t, result)
				assert.Equal(t, v, *result)
			case []string:
				ctxWithValue := WithValue(ctx, v)
				result := GetValue[[]string](ctxWithValue)
				require.NotNil(t, result)
				assert.Equal(t, v, *result)
			case map[string]int:
				ctxWithValue := WithValue(ctx, v)
				result := GetValue[map[string]int](ctxWithValue)
				require.NotNil(t, result)
				assert.Equal(t, v, *result)
			}
		})
	}
}
