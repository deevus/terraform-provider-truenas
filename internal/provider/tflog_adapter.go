package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// TFLogAdapter bridges the client.Logger interface to Terraform's tflog.
type TFLogAdapter struct{}

func (TFLogAdapter) Debug(ctx context.Context, msg string, fields map[string]any) {
	tflog.Debug(ctx, msg, fields)
}
