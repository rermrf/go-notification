package ringbuffer

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewTimeDurationRingBuffer(t *testing.T) {
	testCases := []struct {
		name     string
		capacity int
		wantErr  bool
	}{
		{
			name:     "有效容量",
			capacity: 10,
			wantErr:  false,
		},
		{
			name:     "零容量",
			capacity: 0,
			wantErr:  true,
		},
		{
			name:     "负容量",
			capacity: -10,
			wantErr:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rb, err := NewTimeDurationRingBuffer(tc.capacity)
			if tc.wantErr {
				assert.Error(t, err)
				assert.Nil(t, rb)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, rb)
				assert.Equal(t, tc.capacity, rb.Cap())
				assert.Equal(t, 0, rb.Len())
			}
		})
	}
}
