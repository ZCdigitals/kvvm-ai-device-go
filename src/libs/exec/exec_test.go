package exec

import (
	"testing"
	"time"
)

func TestExec(t *testing.T) {
	ex := NewExec("sleep", "60")

	t.Run("should start right", func(t *testing.T) {
		err := ex.Start()
		if err != nil {
			t.Errorf("start error %v", err)
		}
		defer ex.Stop()

		// wait
		time.Sleep(100 * time.Millisecond)
	})
}
