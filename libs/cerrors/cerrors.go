package cerrors

import (
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/xclient"
	"runtime/debug"
)

func WithPanicRecover(function func()) {
	defer func() {
		if r := recover(); r != nil {
			xlog.Errorf("Recovered from panic: %v | stack trace: %v", r, string(debug.Stack()))
			xclient.GetGlobalDiscordClient().SendWebhookMessage("Recovered from panic" + fmt.Sprintf("Recovered from panic: %v | stack trace: %v", r, string(debug.Stack())))
		}
	}()
	function()
}

func ReturnFirstError(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}
