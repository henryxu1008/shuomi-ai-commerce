package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os/exec"
	"strings"
)

type formalBuildImage struct {
	Component string `json:"component"`
	ImageID   string `json:"imageId"`
	Target    string `json:"target"`
}
type formalBuildReceipt struct {
	SchemaVersion int                `json:"schemaVersion"`
	SourceCommit  string             `json:"sourceCommit"`
	SourceTree    string             `json:"sourceTree"`
	Images        []formalBuildImage `json:"images"`
}
type formalImage struct{ ID, Revision, Tree, Target string }
type formalBuildIdentity struct{ ReceiptPath, ReceiptSHA256, Commit, Tree, ImageID, Target string }
type formalStartupProof struct {
	Version, Commit, Tree, ImageID, SourceSystemID, TestSystemID, SchemaFingerprint string
	RuntimeVerified, Isolated                                                       bool
}

func validateFormalReleaseBinding(m manifest, anchor formalAnchor) error {
	if !validGitID(m.ApprovedBinaryRevision) || !validGitID(m.ApprovedBinaryTree) || m.ApprovedBinaryRevision == anchor.OriginalBuild.Commit || anchor.CandidateBuild.Commit != m.ApprovedBinaryRevision || anchor.CandidateBuild.Tree != m.ApprovedBinaryTree {
		return errors.New("candidate image is not the explicitly approved binary revision")
	}
	return nil
}

func validateFormalStartup(anchor formalAnchor, m manifest, p formalStartupProof) error {
	if err := validateFormalReleaseBinding(m, anchor); err != nil {
		return err
	}
	if p.Version != "chat-isolated-startup-v1" || p.Commit != m.ApprovedBinaryRevision || p.Tree != m.ApprovedBinaryTree || p.ImageID != anchor.CandidateBuild.ImageID || p.SourceSystemID != anchor.Target.SystemIdentifier || p.TestSystemID == "" || p.TestSystemID == p.SourceSystemID || p.SchemaFingerprint != "55110ab58b16be50277d212dd85c3360757ba9432ef55cab7637191ea259fbe8" || !p.RuntimeVerified || !p.Isolated {
		return errors.New("candidate isolated startup evidence mismatch")
	}
	return nil
}

func validGitID(s string) bool {
	if len(s) != 40 || s != strings.ToLower(s) {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}
func validateFormalBuild(r formalBuildReceipt, live formalImage, commit, tree, image, target string) error {
	if r.SchemaVersion != 1 || !validGitID(commit) || !validGitID(tree) || !strings.HasPrefix(image, "sha256:") || !validDigest(strings.TrimPrefix(image, "sha256:")) || target == "" || r.SourceCommit != commit || r.SourceTree != tree || live.ID != image || live.Revision != commit || live.Tree != tree || live.Target != target {
		return errors.New("build receipt does not match approved image and source")
	}
	matches := 0
	for _, i := range r.Images {
		if i.Component == "Backend" && i.ImageID == image && i.Target == target {
			matches++
		}
	}
	if matches != 1 {
		return errors.New("backend build receipt entry unavailable or ambiguous")
	}
	return nil
}

func inspectFormalImage(ctx context.Context, id string) (formalImage, error) {
	if !strings.HasPrefix(id, "sha256:") || !validDigest(strings.TrimPrefix(id, "sha256:")) {
		return formalImage{}, errors.New("invalid image identity")
	}
	out, err := exec.CommandContext(ctx, "docker", "image", "inspect", "--format", `{{.Id}}|{{index .Config.Labels "org.opencontainers.image.revision"}}|{{index .Config.Labels "org.opencontainers.image.tree"}}|{{index .Config.Labels "io.shuomi.build.backend-target"}}`, id).Output()
	if err != nil {
		return formalImage{}, errors.New("image inspection unavailable")
	}
	p := strings.Split(strings.TrimSpace(string(out)), "|")
	if len(p) != 4 {
		return formalImage{}, errors.New("invalid image observation")
	}
	return formalImage{ID: p[0], Revision: p[1], Tree: p[2], Target: p[3]}, nil
}

func verifyFormalBuildReceipt(ctx context.Context, b formalBuildIdentity, inspect func(context.Context, string) (formalImage, error)) error {
	var receipt formalBuildReceipt
	raw, err := loadProtected(ctx, b.ReceiptPath, &receipt)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(raw)
	if !validDigest(b.ReceiptSHA256) || hex.EncodeToString(sum[:]) != b.ReceiptSHA256 {
		return errors.New("image build receipt digest mismatch")
	}
	live, err := inspect(ctx, b.ImageID)
	if err != nil {
		return err
	}
	return validateFormalBuild(receipt, live, b.Commit, b.Tree, b.ImageID, b.Target)
}

func verifyFormalStartupProof(ctx context.Context, anchor formalAnchor, m manifest) error {
	var p formalStartupProof
	raw, err := loadProtected(ctx, anchor.CandidateStartupProofPath, &p)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(raw)
	if !validDigest(anchor.CandidateStartupProofSHA256) || hex.EncodeToString(sum[:]) != anchor.CandidateStartupProofSHA256 {
		return errors.New("candidate isolated startup record digest mismatch")
	}
	return validateFormalStartup(anchor, m, p)
}
