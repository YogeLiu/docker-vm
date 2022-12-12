package utils

import (
	"testing"

	"github.com/emirpasic/gods/maps/linkedhashmap"
	"github.com/stretchr/testify/assert"
)

func TestCache_Add(t *testing.T) {
	cache := NewCache(10)
	type fields struct {
		cache *Cache
	}
	type args struct {
		key   Key
		value interface{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name:   "TestCache_Add",
			fields: fields{cache: cache},
			args:   args{key: "testKey1", value: "testVal1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.fields.cache
			c.Add(tt.args.key, tt.args.value)
			c.Add(tt.args.key, tt.args.value)
			_, ok := c.Get(tt.args.key)
			assert.Equal(t, true, ok, "tt.args.key should exist")
			_, ok = c.Get("test_wrong")
			assert.Equal(t, false, ok, "test_wrong should not exist")
			assert.Equal(t, 1, c.Len(), "cache len should be 1")
		})
	}
}

func TestCache_Clear(t *testing.T) {
	cache := NewCache(10)
	type fields struct {
		cache *Cache
	}
	type args struct {
		key   Key
		value interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantNum int
	}{
		{
			name:    "TestCache_Clear",
			fields:  fields{cache: cache},
			args:    args{key: "testKey1", value: "testVal1"},
			wantNum: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.fields.cache
			c.Add(tt.args.key, tt.args.value)
			c.Clear()
			assert.Equal(t, 0, c.Len(), "cache len should be 0")
		})
	}
}

func TestCache_GetOldest_RemoveOldest(t *testing.T) {

	cache := NewCache(10)

	testKey1 := "testKey1"
	testKey2 := "testKey2"
	testVal1 := "testVal1"
	testVal2 := "testVal2"

	cache.Add(testKey1, testVal1)
	cache.Add(testKey2, testVal2)

	type fields struct {
		cache *Cache
	}
	type args struct {
		key   Key
		value interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantVal string
	}{
		{
			name:    "TestCache_GetOldest_RemoveOldest",
			fields:  fields{cache: cache},
			wantVal: testVal1,
		},
		{
			name:    "TestCache_GetOldest_RemoveOldest",
			fields:  fields{cache: cache},
			wantVal: testVal2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.fields.cache
			gotVal := c.GetOldest().(string)
			assert.Equal(t, tt.wantVal, gotVal, "oldest val not match")
			c.RemoveOldest()
		})
	}
}

//func TestCache_Len(t *testing.T) {
//	type fields struct {
//		MaxEntries int
//		OnEvicted  func(key Key, value interface{})
//		ll         *list.List
//		cache      map[interface{}]*list.Element
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		want   int
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Cache{
//				MaxEntries: tt.fields.MaxEntries,
//				OnEvicted:  tt.fields.OnEvicted,
//				ll:         tt.fields.ll,
//				cache:      tt.fields.cache,
//			}
//			if got := c.Len(); got != tt.want {
//				t.Errorf("Len() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestCache_Remove(t *testing.T) {
//	type fields struct {
//		MaxEntries int
//		OnEvicted  func(key Key, value interface{})
//		ll         *list.List
//		cache      map[interface{}]*list.Element
//	}
//	type args struct {
//		key Key
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Cache{
//				MaxEntries: tt.fields.MaxEntries,
//				OnEvicted:  tt.fields.OnEvicted,
//				ll:         tt.fields.ll,
//				cache:      tt.fields.cache,
//			}
//			c.Remove(tt.args.key)
//		})
//	}
//}
//
//func TestCache_RemoveOldest(t *testing.T) {
//	type fields struct {
//		MaxEntries int
//		OnEvicted  func(key Key, value interface{})
//		ll         *list.List
//		cache      map[interface{}]*list.Element
//	}
//	tests := []struct {
//		name   string
//		fields fields
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Cache{
//				MaxEntries: tt.fields.MaxEntries,
//				OnEvicted:  tt.fields.OnEvicted,
//				ll:         tt.fields.ll,
//				cache:      tt.fields.cache,
//			}
//			c.RemoveOldest()
//		})
//	}
//}
//
//func TestCache_removeElement(t *testing.T) {
//	type fields struct {
//		MaxEntries int
//		OnEvicted  func(key Key, value interface{})
//		ll         *list.List
//		cache      map[interface{}]*list.Element
//	}
//	type args struct {
//		e *list.Element
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Cache{
//				MaxEntries: tt.fields.MaxEntries,
//				OnEvicted:  tt.fields.OnEvicted,
//				ll:         tt.fields.ll,
//				cache:      tt.fields.cache,
//			}
//			c.removeElement(tt.args.e)
//		})
//	}
//}

func TestNewCache(t *testing.T) {
	type args struct {
		maxEntries int
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "TestNewCache",
			args: args{maxEntries: 10},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = NewCache(tt.args.maxEntries)
		})
	}
}

func TestIterator(t *testing.T) {

	hmap := linkedhashmap.New()
	hmap.Put("key0", "val0")
	hmap.Put("key1", "val1")
	hmap.Put("key2", "val2")

	type args struct {
		key string
		val string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "TestNewCache",
			args: args{key: "key0", val: "val0"},
		},
		{
			name: "TestNewCache",
			args: args{key: "key1", val: "val1"},
		},
		{
			name: "TestNewCache",
			args: args{key: "key2", val: "val2"},
		},
	}
	hmapIt := hmap.Iterator()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hmapIt.Next()
			val := hmapIt.Value().(string)
			if !assert.Equal(t, val, tt.args.val) {
				t.Errorf("TestIterator, want %v, got %v", tt.args.val, val)
			}
		})
	}
}
