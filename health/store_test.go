package health

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAsyncHealthStore(t *testing.T) {
	store := NewAsyncHealthStore()
	assert.NotNil(t, store)
	assert.NotNil(t, store.results)
}

func TestAsyncHealthStore_UpdateStatus(t *testing.T) {
	t.Run("Update with Error", func(t *testing.T) {
		store := NewAsyncHealthStore()
		err := fmt.Errorf("test error")
		store.UpdateStatus("testCheck", err)

		retrievedErr := store.GetStatus("testCheck")
		assert.Equal(t, err, retrievedErr)
	})

	t.Run("Update with Nil", func(t *testing.T) {
		store := NewAsyncHealthStore()
		store.UpdateStatus("testCheck", nil)

		retrievedErr := store.GetStatus("testCheck")
		assert.NoError(t, retrievedErr) // Or assert.Nil(t, retrievedErr)
		assert.Nil(t, retrievedErr)
	})
	t.Run("Update Overwrites", func(t *testing.T) {
		store := NewAsyncHealthStore()
		err1 := fmt.Errorf("first error")
		store.UpdateStatus("testCheck", err1)

		err2 := fmt.Errorf("second error")
		store.UpdateStatus("testCheck", err2)

		retrievedErr := store.GetStatus("testCheck")
		assert.Equal(t, err2, retrievedErr)
		assert.NotEqual(t, err1, retrievedErr)
	})
}

func TestAsyncHealthStore_GetStatus(t *testing.T) {
	t.Run("Get Non-Existent Check", func(t *testing.T) {
		store := NewAsyncHealthStore()
		err := store.GetStatus("nonExistentCheck")
		assert.Nil(t, err) // No error if not found
	})

	t.Run("Get Existing Check", func(t *testing.T) {
		store := NewAsyncHealthStore()
		expectedErr := fmt.Errorf("test error")
		store.UpdateStatus("testCheck", expectedErr)

		retrievedErr := store.GetStatus("testCheck")
		assert.Equal(t, expectedErr, retrievedErr)
	})
}
