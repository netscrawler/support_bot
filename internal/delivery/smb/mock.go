package smb

import (
	"context"
	"fmt"
	models "support_bot/internal/models/report"
)

type UnimplementedSMBSenderServer struct {}

func (UnimplementedSMBSenderServer) Upload(ctx context.Context, remote string, fileData ...models.ReportData) error {
	return fmt.Errorf("smb sender disabled")
}
