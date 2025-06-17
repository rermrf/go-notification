package batchSize

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestNewFixedStepAdjuster(t *testing.T) {
	testCases := []struct {
		name          string
		initialSize   int
		minSize       int
		maxSize       int
		adjustStep    int
		interval      time.Duration
		fastThreshold time.Duration
		slowThreshold time.Duration
		expectedSize  int
	}{
		{
			name:          "正常初始化",
			initialSize:   50,
			minSize:       10,
			maxSize:       100,
			adjustStep:    5,
			interval:      time.Second * 10,
			fastThreshold: 150 * time.Millisecond,
			slowThreshold: 200 * time.Millisecond,
			expectedSize:  50,
		},
		{
			name:          "出始值小于最小值",
			initialSize:   5,
			minSize:       10,
			maxSize:       100,
			adjustStep:    5,
			interval:      time.Second * 10,
			fastThreshold: 150 * time.Millisecond,
			slowThreshold: 200 * time.Millisecond,
			// 应当被调整为最小值
			expectedSize: 10,
		},
		{
			name:          "出始值大于最大值",
			initialSize:   200,
			minSize:       10,
			maxSize:       100,
			adjustStep:    5,
			interval:      time.Second * 10,
			fastThreshold: 150 * time.Millisecond,
			slowThreshold: 200 * time.Millisecond,
			// 应当被调整为最大值
			expectedSize: 100,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			adjuster := NewFixedStepAdjuster(tc.initialSize, tc.minSize, tc.maxSize, tc.adjustStep, tc.interval, tc.fastThreshold, tc.slowThreshold)

			firstSize, err := adjuster.Adjust(context.Background(), 175*time.Millisecond)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedSize, firstSize)
		})
	}
}

func TestAdjustBatchSize(t *testing.T) {
	t.Run("响应时间影响批次大小", func(t *testing.T) {
		t.Parallel()
		// 创建一个无间隔限制的调整器
		adjuster := NewFixedStepAdjuster(50, 10, 100, 10, 0, 150*time.Millisecond, 200*time.Millisecond)

		// 1. 响应时间在中间范围 - 保持不变
		size, err := adjuster.Adjust(t.Context(), 175*time.Millisecond)
		assert.NoError(t, err)
		assert.Equal(t, 50, size, "中间响应时间不应改变批次大小")

		// 2. 快速响应 - 增加批次大小
		size, err = adjuster.Adjust(t.Context(), 100*time.Millisecond)
		assert.NoError(t, err)
		assert.Equal(t, 60, size, "快速响应增加批次大小")

		// 3. 再次快速响应 - 继续增加
		size, err = adjuster.Adjust(t.Context(), 100*time.Millisecond)
		assert.NoError(t, err)
		assert.Equal(t, 70, size, "连续快速响应继续增加批次大小")

		//4. 慢速响应 - 减少批次大小
		size, err = adjuster.Adjust(t.Context(), 250*time.Millisecond) // 高于慢速阈值
		assert.NoError(t, err)
		assert.Equal(t, 60, size, "慢速响应应当减少批次大小")
	})

	t.Run("批次大小有边界限制", func(t *testing.T) {
		t.Parallel()

		adjuster := NewFixedStepAdjuster(50, 10, 100, 20, 0, 150*time.Millisecond, 200*time.Millisecond)
		// 1. 接近最大值时快时响应
		size, err := adjuster.Adjust(t.Context(), 100*time.Millisecond)
		assert.NoError(t, err)
		assert.Equal(t, 70, size, "应增加但是不超过最大值")

		size, err = adjuster.Adjust(t.Context(), 100*time.Millisecond)
		assert.NoError(t, err)
		assert.Equal(t, 90, size, "再次应增加但是不超过最大值")

		size, err = adjuster.Adjust(t.Context(), 100*time.Millisecond)
		assert.NoError(t, err)
		assert.Equal(t, 100, size, "超过最大边界限制，应为最大值")

		minAdjuster := NewFixedStepAdjuster(30, 10, 100, 20, 0, 150*time.Millisecond, 200*time.Millisecond)

		size, err = minAdjuster.Adjust(t.Context(), 250*time.Millisecond)
		assert.NoError(t, err)
		assert.Equal(t, 10, size, "降低")

		size, err = minAdjuster.Adjust(t.Context(), 250*time.Millisecond)
		assert.NoError(t, err)
		assert.Equal(t, 10, size, "再次降低，不超过最小值")
	})

	t.Run("调整间隔限制", func(t *testing.T) {
		// 创建一个带有间隔限制的调整器
		adjuster := NewFixedStepAdjuster(50, 10, 100, 10, 100*time.Millisecond, 150*time.Millisecond, 200*time.Millisecond)

		// 1. 首次挑战应当正常
		size, err := adjuster.Adjust(t.Context(), 100*time.Millisecond)
		assert.NoError(t, err)
		assert.Equal(t, 60, size, "首次应当正常调整")

		// 2. 紧接着调用不调整
		size, err = adjuster.Adjust(t.Context(), 100*time.Millisecond)
		assert.NoError(t, err)
		assert.Equal(t, 60, size, "在时间间隔内不调整")

		// 3. 时间间隔外，应当调整
		time.Sleep(150 * time.Millisecond)
		size, err = adjuster.Adjust(t.Context(), 100*time.Millisecond)
		assert.NoError(t, err)
		assert.Equal(t, 70, size, "时间间隔外，应当调整")
	})
}

func TestContinuousAdjustment(t *testing.T) {
	t.Run("连续调整", func(t *testing.T) {
		adjuster := NewFixedStepAdjuster(50, 10, 100, 10, 0, 150*time.Millisecond, 200*time.Millisecond)

		initialSize, _ := adjuster.Adjust(t.Context(), 175*time.Millisecond) // 中间值，不调整
		assert.Equal(t, 50, initialSize)

		// 连续增长直到上限
		sizes := []int{60, 70, 80, 90, 100, 100}
		for _, size := range sizes {
			adjustedSize, err := adjuster.Adjust(t.Context(), 100*time.Millisecond) // 快速响应
			assert.NoError(t, err)
			assert.Equal(t, size, adjustedSize)
		}

		// 连续下降直到下限
		sizes = []int{90, 80, 70, 60, 50, 40, 30, 20, 10, 10}
		for _, size := range sizes {
			adjustedSize, err := adjuster.Adjust(t.Context(), 250*time.Millisecond)
			assert.NoError(t, err)
			assert.Equal(t, size, adjustedSize)
		}
	})
}
