package acquisition

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/product/sourcing"
)

func browserV2StagingOperation(t *testing.T) (sourcing.AcquisitionOperation, sourcing.PublicationCommand) {
	op, command := browserStagingOperation(t)
	command.Producer.Version = "v2"
	command.Envelope.RawReference.ReferenceType = "browser_capture_evidence_v2"
	m := command.Envelope.RawReference.Metadata
	m["contract_version"], m["parser_version"] = "src2b-browser-capture-v2", "1688-browser-dom/v3"
	input, _ := json.Marshal([]string{"src2b-browser-capture-v2", "browser_capture", op.Source.URL, op.CaptureSHA256})
	op.Fingerprint = digest(input)
	op.Command = &command
	return op, command
}

func TestBrowserCaptureV2CommandDeterminesOneValidator(t *testing.T) {
	op, command := browserV2StagingOperation(t)
	require.True(t, validIdentity(op))
	require.True(t, validCommand(op, command))
	for _, part := range []string{"producer", "contract", "parser", "fingerprint"} {
		t.Run(part, func(t *testing.T) {
			op, command := browserV2StagingOperation(t)
			switch part {
			case "producer":
				command.Producer.Version = "v1"
			case "contract":
				command.Envelope.RawReference.Metadata["contract_version"] = sourcing.AcquisitionContractVersion
			case "parser":
				command.Envelope.RawReference.Metadata["parser_version"] = sourcing.BrowserCaptureParserVersion
			case "fingerprint":
				op.Fingerprint, _ = sourcing.BrowserAcquisitionFingerprint(op.Source, op.CaptureSHA256)
			}
			require.False(t, validCommand(op, command))
		})
	}
}

func TestBrowserCaptureFrozenV1CommandHash(t *testing.T) {
	_, command := browserStagingOperation(t)
	raw, err := json.Marshal(command)
	require.NoError(t, err)
	require.Equal(t, "6899a1539a7e0f82496766e7ee00f5683e6150db1d17695b22f0ebd9b75864fc", digest(raw))
}
