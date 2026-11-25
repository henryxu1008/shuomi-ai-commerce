# 商品图片交付规格离线证据

[#1643 设计候选](product-image-delivery-specs.md) · 2026-11-25 · Go 1.26.5（Windows amd64），仓库已缓存的 `golang.org/x/image v0.44.0`。全部文件来自本地几何图形。两个黄色矩形标记受裁切影响的内容。没有业务数据或外部调用。

| 文件 | 加工参数 | 解码格式与尺寸 | 比例 | 实测字节数 | SHA-256 |
| --- | --- | --- | --- | ---: | --- |
| `source-square.png` | 方图输入 | PNG · 1280 × 1280 | 1:1 | 7,769 | `cd76a0c6bee0193c9f372e84c7febcfd64089cb3c1e020f8f18c28964a61f2c2` |
| `source-wide.png` | 横图输入 | PNG · 1600 × 1440 | 10:9 | 12,018 | `384f9749c750b5f7e10e725ec9389d61309865d700e3e7315db11b7ea368b67a` |
| `shopify-square.png` | 方图等比缩小；PNG BestCompression | PNG · 1024 × 1024 | 1:1 | 10,484 | `42ee98c48c4dcc1b94c797728aaf6ed004d56076b94eb84b0b874c0807d5da31` |
| `detail-padded.jpg` | 横图全幅缩至 960 × 864；上下各补 208px 白底；JPEG 90 | JPEG · 960 × 1280 | 3:4 | 101,267 | `18cf9f84dd9cacb28b568d95c82fd987f2e5305eee95e1d230009999eeba20ef` |
| `detail-cropped-preview.jpg` | 横图居中保留 `x ∈ [260,1340)，y ∈ [0,1440)`，缩至 960 × 1280；JPEG 90 | JPEG · 960 × 1280 | 3:4 | 104,190 | `859cef9baff83222f6dade41d313ee2b6d7d1bf0b7af73565d11e4dd635b34ea` |

落盘后重新打开并完整解码，校验尺寸、格式和成品 ≤ 3,145,728 字节。补白预览保留两侧完整标记；裁切预览截掉两侧内容。另将方图文件截断至一半，头信息仍可读取，完整解码按预期拒绝。文件大小结果只适用于这次合成输入。生产校验、超限压缩尝试、真实资产、采用事务、正式消费者、店铺接收及用户验收均未执行。

## 复现

在该仓库根目录运行。将下方代码保存到新建的临时目录，例如 `issue-1643-offline` 下的 `verify.go`。同一 Go 和 `x/image` 版本下可比对字节摘要。依赖已缓存时，`GOPROXY=off` 可离线重跑。

```powershell
$out = Join-Path $env:TEMP 'issue-1643-offline'
$env:GOPROXY = 'off'
go run -mod=readonly (Join-Path $out 'verify.go') $out
```

```go
package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"log"
	"os"
	"path/filepath"

	scale "golang.org/x/image/draw"
)

const limit = 3 << 20

func fixture(w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, color.NRGBA{uint8(215 + x%31), uint8(216 + y%29), 237, 255})
		}
	}
	for _, block := range []struct {
		bounds image.Rectangle
		tone   color.NRGBA
	}{
		{image.Rect(w/10, h/3, w*9/10, h*3/4), color.NRGBA{27, 52, 78, 255}},
		{image.Rect(w/8, h*2/5, w/4, h/2), color.NRGBA{231, 175, 74, 255}},
		{image.Rect(w*3/4, h*2/5, w*7/8, h/2), color.NRGBA{231, 175, 74, 255}},
	} {
		draw.Draw(img, block.bounds, image.NewUniform(block.tone), image.Point{}, draw.Src)
	}
	return img
}

func save(dir, name, format string, img image.Image, enforceLimit bool) image.Image {
	var data bytes.Buffer
	var err error
	switch format {
	case "png":
		encoder := png.Encoder{CompressionLevel: png.BestCompression}
		err = encoder.Encode(&data, img)
	case "jpeg":
		err = jpeg.Encode(&data, img, &jpeg.Options{Quality: 90})
	}
	if err != nil {
		log.Fatal(err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data.Bytes(), 0o600); err != nil {
		log.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	decoded, actual, err := image.Decode(bytes.NewReader(content))
	if err != nil || actual != format || decoded.Bounds() != img.Bounds() || (enforceLimit && len(content) > limit) {
		log.Fatalf("%s: unexpected output: %s %v", name, actual, err)
	}
	fmt.Printf("%s %s %dx%d %d %x\n", name, actual, decoded.Bounds().Dx(), decoded.Bounds().Dy(), len(content), sha256.Sum256(content))
	return decoded
}

func render(src image.Image, size, from, to image.Rectangle) image.Image {
	dst := image.NewNRGBA(size)
	draw.Draw(dst, size, image.White, image.Point{}, draw.Src)
	scale.CatmullRom.Scale(dst, to, src, from, draw.Over, nil)
	return dst
}

func main() {
	dir := os.Args[1]
	square := save(dir, "source-square.png", "png", fixture(1280, 1280), false)
	wide := save(dir, "source-wide.png", "png", fixture(1600, 1440), false)
	outSquare := image.Rect(0, 0, 1024, 1024)
	outDetail := image.Rect(0, 0, 960, 1280)
	save(dir, "shopify-square.png", "png", render(square, outSquare, square.Bounds(), outSquare), true)
	save(dir, "detail-padded.jpg", "jpeg", render(wide, outDetail, wide.Bounds(), image.Rect(0, 208, 960, 1072)), true)
	save(dir, "detail-cropped-preview.jpg", "jpeg", render(wide, outDetail, image.Rect(260, 0, 1340, 1440), outDetail), true)
	squareBytes, err := os.ReadFile(filepath.Join(dir, "shopify-square.png"))
	if err != nil {
		log.Fatal(err)
	}
	truncated := squareBytes[:len(squareBytes)/2]
	if _, _, err := image.DecodeConfig(bytes.NewReader(truncated)); err != nil {
		log.Fatal(err)
	}
	if _, _, err := image.Decode(bytes.NewReader(truncated)); err == nil {
		log.Fatal("full decode accepted truncated PNG")
	} else {
		fmt.Printf("truncated PNG: header accepted, full decode rejected (%v)\n", err)
	}
}
```
