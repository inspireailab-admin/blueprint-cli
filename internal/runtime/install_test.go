package runtime

import (
	"testing"
)

// llamaCppB9775Assets mirrors the exact asset name list published in the
// ggml-org/llama.cpp b9775 release. We use it as a fixture: the asset picker
// has to land on the right plain-CPU build for each (os, arch) without picking
// CUDA / Vulkan / SYCL / OpenVINO / OpenCL / ROCm or HIP variants.
//
// If llama.cpp ever changes their asset naming convention this fixture is
// where you'll feel it first.
var llamaCppB9775Assets = []ghAsset{
	{Name: "cudart-llama-bin-win-cuda-12.4-x64.zip"},
	{Name: "cudart-llama-bin-win-cuda-13.3-x64.zip"},
	{Name: "llama-b9775-bin-android-arm64.tar.gz"},
	{Name: "llama-b9775-bin-macos-arm64.tar.gz"},
	{Name: "llama-b9775-bin-macos-x64.tar.gz"},
	{Name: "llama-b9775-bin-ubuntu-arm64.tar.gz"},
	{Name: "llama-b9775-bin-ubuntu-openvino-2026.2-x64.tar.gz"},
	{Name: "llama-b9775-bin-ubuntu-rocm-7.2-x64.tar.gz"},
	{Name: "llama-b9775-bin-ubuntu-s390x.tar.gz"},
	{Name: "llama-b9775-bin-ubuntu-sycl-fp16-x64.tar.gz"},
	{Name: "llama-b9775-bin-ubuntu-sycl-fp32-x64.tar.gz"},
	{Name: "llama-b9775-bin-ubuntu-vulkan-arm64.tar.gz"},
	{Name: "llama-b9775-bin-ubuntu-vulkan-x64.tar.gz"},
	{Name: "llama-b9775-bin-ubuntu-x64.tar.gz"},
	{Name: "llama-b9775-bin-win-cpu-arm64.zip"},
	{Name: "llama-b9775-bin-win-cpu-x64.zip"},
	{Name: "llama-b9775-bin-win-cuda-12.4-x64.zip"},
	{Name: "llama-b9775-bin-win-cuda-13.3-x64.zip"},
	{Name: "llama-b9775-bin-win-hip-radeon-x64.zip"},
	{Name: "llama-b9775-bin-win-opencl-adreno-arm64.zip"},
	{Name: "llama-b9775-bin-win-openvino-2026.2-x64.zip"},
	{Name: "llama-b9775-bin-win-sycl-x64.zip"},
	{Name: "llama-b9775-bin-win-vulkan-x64.zip"},
	{Name: "llama-b9775-ui.tar.gz"},
	{Name: "llama-b9775-xcframework.zip"},
}

func TestPickAsset_AllSupportedPlatforms(t *testing.T) {
	cases := []struct {
		os       string
		arch     string
		expected string
	}{
		{"darwin", "amd64", "llama-b9775-bin-macos-x64.tar.gz"},
		{"darwin", "arm64", "llama-b9775-bin-macos-arm64.tar.gz"},
		{"linux", "amd64", "llama-b9775-bin-ubuntu-x64.tar.gz"},
		{"linux", "arm64", "llama-b9775-bin-ubuntu-arm64.tar.gz"},
		{"windows", "amd64", "llama-b9775-bin-win-cpu-x64.zip"},
		{"windows", "arm64", "llama-b9775-bin-win-cpu-arm64.zip"},
	}
	for _, c := range cases {
		t.Run(c.os+"-"+c.arch, func(t *testing.T) {
			got, err := pickAssetFor(llamaCppB9775Assets, c.os, c.arch)
			if err != nil {
				t.Fatalf("pickAssetFor(%s,%s): %v", c.os, c.arch, err)
			}
			if got.Name != c.expected {
				t.Errorf("pickAssetFor(%s,%s) = %q, want %q", c.os, c.arch, got.Name, c.expected)
			}
		})
	}
}

func TestPickAsset_UnsupportedPlatform(t *testing.T) {
	_, err := pickAssetFor(llamaCppB9775Assets, "plan9", "amd64")
	if err == nil {
		t.Fatal("pickAssetFor(plan9, amd64): want error, got nil")
	}
}

func TestPickAsset_EmptyAssets(t *testing.T) {
	_, err := pickAssetFor(nil, "linux", "amd64")
	if err == nil {
		t.Fatal("pickAssetFor(nil,...): want error, got nil")
	}
}

func TestHasAccelerator(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"llama-b9775-bin-win-cpu-x64.zip", false},
		{"llama-b9775-bin-macos-arm64.tar.gz", false},
		{"llama-b9775-bin-ubuntu-x64.tar.gz", false},

		{"llama-b9775-bin-win-cuda-12.4-x64.zip", true},
		{"llama-b9775-bin-win-cuda-13.3-x64.zip", true},
		{"llama-b9775-bin-win-vulkan-x64.zip", true},
		{"llama-b9775-bin-win-hip-radeon-x64.zip", true},
		{"llama-b9775-bin-win-sycl-x64.zip", true},
		{"llama-b9775-bin-win-opencl-adreno-arm64.zip", true},
		{"llama-b9775-bin-win-openvino-2026.2-x64.zip", true},
		{"llama-b9775-bin-ubuntu-rocm-7.2-x64.tar.gz", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := hasAccelerator(c.name)
			if got != c.want {
				t.Errorf("hasAccelerator(%q) = %v, want %v", c.name, got, c.want)
			}
		})
	}
}

func TestIsRuntimeFile(t *testing.T) {
	cases := []struct {
		name string
		path string
		want bool
	}{
		// Top-level flat layout (Windows zip)
		{"top-level binary", "llama-server.exe", true},
		{"top-level DLL", "ggml.dll", true},
		{"top-level dylib", "libllama.dylib", true},

		// Nested in /bin/ (Mac / Linux tar)
		{"build bin layout", "build/bin/llama-server", true},
		{"plain bin layout", "bin/llama-server", true},

		// Things we explicitly don't want
		{"path traversal", "../etc/passwd", false},
		{"source code", "src/main.cpp", false},
		{"include dir", "include/llama.h", false},
		{"docs", "docs/build.md", false},

		// Edge cases
		{"empty", "", true}, // top-level no-slash; benign — won't match a real bin
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := isRuntimeFile(c.path)
			if got != c.want {
				t.Errorf("isRuntimeFile(%q) = %v, want %v", c.path, got, c.want)
			}
		})
	}
}

func TestPlatformTagFor(t *testing.T) {
	cases := []struct {
		os, arch    string
		wantPlatStr string
		wantArchStr string
	}{
		{"darwin", "arm64", "macos-arm64", ""},
		{"darwin", "amd64", "macos-x64", ""},
		{"linux", "amd64", "ubuntu", "x64"},
		{"linux", "arm64", "ubuntu-arm64", ""},
		{"windows", "amd64", "win", "x64"},
		{"windows", "arm64", "win", "arm64"},
		{"plan9", "amd64", "", ""},
	}
	for _, c := range cases {
		t.Run(c.os+"-"+c.arch, func(t *testing.T) {
			plat, arch := platformTagFor(c.os, c.arch)
			if plat != c.wantPlatStr || arch != c.wantArchStr {
				t.Errorf("platformTagFor(%s,%s) = (%q,%q), want (%q,%q)",
					c.os, c.arch, plat, arch, c.wantPlatStr, c.wantArchStr)
			}
		})
	}
}
