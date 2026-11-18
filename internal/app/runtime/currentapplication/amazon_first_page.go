package currentapplication

import (
	"context"
	"errors"
	"io/fs"
	"path"

	acq "task-processor/internal/product/dataacquisition"
)

// AmazonFirstPageFiles opens a single administrator-frozen approval. The key
// remains in the read-only directory until an approved Submit owns the claim.
type AmazonFirstPageFiles struct {
	Approval acq.AmazonFirstPageApproval
	files    fs.FS
}

func (cfg *Config) validateAmazonFirstPage() error {
	if cfg.AmazonFirstPageInputDir == nil {
		return nil
	}
	dir := *cfg.AmazonFirstPageInputDir
	if !path.IsAbs(dir) || dir == "/" || path.Clean(dir) != dir ||
		(cfg.SheinUSTrialInputDir != nil && dir == *cfg.SheinUSTrialInputDir) ||
		cfg.MarketResearch == nil || cfg.DataServices == nil || !cfg.DataServices.ExecutionEnabled {
		return errors.New("Amazon first-page trial requires its private directory, Market Research and live Data Services")
	}
	return nil
}

func loadAmazonFirstPageFiles(files fs.FS) (*AmazonFirstPageFiles, error) {
	input, err := readTrialFile[struct {
		Version  string                      `json:"version"`
		Approval acq.AmazonFirstPageApproval `json:"approval"`
	}](files, "approval.json", trialControlFileLimit)
	if err != nil || input.Version != "amazon-zenrows-first-page@v1" || input.Approval.Validate() != nil {
		return nil, acq.ErrUnavailable
	}
	return &AmazonFirstPageFiles{Approval: input.Approval, files: files}, nil
}

func (f *AmazonFirstPageFiles) ReadKey(ctx context.Context) (string, error) {
	if ctx.Err() != nil {
		return "", acq.ErrUnavailable
	}
	key, err := readTrialFile[struct {
		Binding string `json:"binding"`
		Key     string `json:"key"`
	}](f.files, "key.json", trialKeyFileLimit)
	if err != nil || key.Binding != f.Approval.Binding() {
		return "", acq.ErrUnavailable
	}
	return key.Key, nil
}
