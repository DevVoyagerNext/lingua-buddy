// Package storage 提供对象存储抽象与阿里云 OSS 实现。
// 语音音频走 OSS：上传后生成签名 URL 供 Paraformer 拉取。
package storage

import (
	"bytes"
	"fmt"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"

	"lingua-buddy/internal/config"
)

// Provider 对象存储抽象。
type Provider interface {
	// Save 上传对象。
	Save(key string, data []byte, contentType string) error
	// SignedURL 生成可公网访问的临时下载 URL。
	SignedURL(key string, expiry time.Duration) (string, error)
	// Delete 删除对象。
	Delete(key string) error
	// Available 是否已正确配置（可用于真实上传）。
	Available() bool
}

// OSS 阿里云对象存储实现。
type OSS struct {
	bucket *oss.Bucket
	name   string
}

// NewOSS 构造 OSS Provider；未配置时返回不可用实例。
func NewOSS(cfg config.UploadConfig) (*OSS, error) {
	if cfg.OSSEndpt == "" || cfg.OSSKey == "" || cfg.OSSSecret == "" || cfg.OSSBucket == "" {
		return &OSS{}, nil
	}
	client, err := oss.New(cfg.OSSEndpt, cfg.OSSKey, cfg.OSSSecret)
	if err != nil {
		return &OSS{}, fmt.Errorf("初始化 OSS 客户端失败: %w", err)
	}
	bucket, err := client.Bucket(cfg.OSSBucket)
	if err != nil {
		return &OSS{}, fmt.Errorf("获取 OSS bucket 失败: %w", err)
	}
	return &OSS{bucket: bucket, name: cfg.OSSBucket}, nil
}

// Available 是否可用。
func (o *OSS) Available() bool { return o.bucket != nil }

// Save 上传对象。
func (o *OSS) Save(key string, data []byte, contentType string) error {
	if o.bucket == nil {
		return fmt.Errorf("OSS 未配置")
	}
	var opts []oss.Option
	if contentType != "" {
		opts = append(opts, oss.ContentType(contentType))
	}
	return o.bucket.PutObject(key, bytes.NewReader(data), opts...)
}

// SignedURL 生成签名下载 URL。
func (o *OSS) SignedURL(key string, expiry time.Duration) (string, error) {
	if o.bucket == nil {
		return "", fmt.Errorf("OSS 未配置")
	}
	return o.bucket.SignURL(key, oss.HTTPGet, int64(expiry.Seconds()))
}

// Delete 删除对象。
func (o *OSS) Delete(key string) error {
	if o.bucket == nil {
		return nil
	}
	return o.bucket.DeleteObject(key)
}
