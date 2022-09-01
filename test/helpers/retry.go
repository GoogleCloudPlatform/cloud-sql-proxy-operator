// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package helpers

import (
	"fmt"
	"time"

	"github.com/go-logr/logr"
)

func RetryUntilSuccess(t TestLogger, attempts int, sleep time.Duration, f func() error) (err error) {
	for i := 0; i < attempts; i++ {
		if i > 0 {
			t.Logf("retrying after attempt %d, %v", i, err)
			time.Sleep(sleep)
			sleep *= 2
		}
		err = f()
		if err == nil {
			return nil
		}
	}
	return fmt.Errorf("after %d attempts, last error: %s", attempts, err)
}

type TestLogger interface {
	Logf(format string, args ...interface{})
}

type TestSetupLogger struct {
	logr.Logger
}

func (l *TestSetupLogger) Logf(format string, args ...interface{}) {
	l.Logger.Info(fmt.Sprintf(format, args...))
}
