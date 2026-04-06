package internal

import (
	"path/filepath"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/openotters/agentfile/spec"
)

type LayerInfo struct {
	Title     string
	MediaType string
	Size      int64
	Digest    string
}

type Info struct {
	Name        string
	Path        string
	Description string
	UsagePath   string
	Layers      []LayerInfo
}

func (i Info) BinPath() string {
	return filepath.Join(i.Path, i.Name)
}

func Inspect(manifest v1.Manifest) Info {
	info := Info{
		Path:      spec.DefaultBinPath,
		UsagePath: spec.DefaultUsagePath,
	}

	if v, ok := manifest.Annotations[spec.AnnotationBinName]; ok && v != "" {
		info.Name = v
	}

	if v, ok := manifest.Annotations[spec.AnnotationBinPath]; ok && v != "" {
		info.Path = v
	}

	if v, ok := manifest.Annotations[spec.AnnotationBinDescription]; ok {
		info.Description = v
	}

	if v, ok := manifest.Annotations[spec.AnnotationBinUsage]; ok && v != "" {
		info.UsagePath = v
	}

	for _, l := range manifest.Layers {
		info.Layers = append(info.Layers, LayerInfo{
			Title:     l.Annotations[v1.AnnotationTitle],
			MediaType: l.MediaType,
			Size:      l.Size,
			Digest:    l.Digest.String(),
		})
	}

	return info
}
