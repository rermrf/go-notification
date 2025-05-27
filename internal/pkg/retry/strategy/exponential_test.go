package strategy

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestNewExponentialBackoffRetryStrategy_New(t *testing.T) {
	// 这个方法的作用是确保测试在并发环境下运行
	//t.Parallel()

	testCases := []struct {
		name            string
		initialInterval time.Duration
		maxInterval     time.Duration
		maxRetries      int32
		want            *ExponentialBackoffRetryStrategy
		wantErr         error
	}{
		{
			name:            "no error",
			initialInterval: 2 * time.Second,
			maxInterval:     2 * time.Minute,
			maxRetries:      5,
			want: func() *ExponentialBackoffRetryStrategy {
				s := NewExponentialBackoffRetryStrategy(2*time.Second, 2*time.Minute, 5)
				return s
			}(),
			wantErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Parallel()
		s := NewExponentialBackoffRetryStrategy(tc.initialInterval, tc.maxInterval, tc.maxRetries)
		assert.Equal(t, tc.want, s, tc.name)
	}
}

func TestExponentialBackoffRetryStrategy_Next(t *testing.T) {
	testCases := []struct {
		name     string
		ctx      context.Context
		strategy *ExponentialBackoffRetryStrategy

		wantIntervals []time.Duration
	}{
		{
			name: "如果重试次数达到最大值，则停止重试",
			ctx:  t.Context(),
			strategy: func() *ExponentialBackoffRetryStrategy {
				s := NewExponentialBackoffRetryStrategy(1*time.Second, 4*time.Second, 5)
				return s
			}(),
			wantIntervals: []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second, 4 * time.Second, 4 * time.Second},
		},
		{
			name: "初始间隔超过最大间隔",
			ctx:  t.Context(),
			strategy: func() *ExponentialBackoffRetryStrategy {
				return NewExponentialBackoffRetryStrategy(5*time.Second, 3*time.Second, 3)
			}(),
			wantIntervals: []time.Duration{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			intervals := make([]time.Duration, 0)
			for {
				if interval, ok := tc.strategy.Next(); ok {
					intervals = append(intervals, interval)
				} else {
					break
				}
			}
			assert.Equal(t, tc.wantIntervals, intervals)
		})
	}
}

// 指数退避重试策略子测试函数，无限重试
func TestExponentialBackoffRetryStrategy_Next_Infinitely(t *testing.T) {
	//t.Parallel()
	//t.Run("最大重试次数为0", func(t *testing.T) {
	//	t.Parallel()
	//	testNextInfiniteRetry(t, 0)
	//})
	//
	//t.Run("最大重试次数为负数", func(t *testing.T) {
	//	t.Parallel()
	//	testNextInfiniteRetry(t, -1)
	//})
}

//func testNextInfiniteRetry(t *testing.T, i int) {
//
//}
