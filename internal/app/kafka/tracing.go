package kafka

import (
	"bytes"
	"context"
	"strings"

	"github.com/Shopify/sarama"
	"github.com/opentracing/opentracing-go"
	"google.golang.org/grpc/metadata"
)

// Extract ...
func Extract(ctx context.Context, headers []*sarama.RecordHeader) (context.Context, error) {
	if len(headers) == 0 {
		return nil, opentracing.ErrInvalidCarrier
	}

	var (
		tracer = opentracing.GlobalTracer()
		h      = make(KafkaHeaders, len(headers))
	)

	for i := range headers {
		h = append(h, *headers[i])
	}

	parentSpanCtx, err := tracer.Extract(opentracing.TextMap, &h)
	if err != nil {
		return nil, err
	}

	span := tracer.StartSpan("Kafka Read Sync", opentracing.FollowsFrom(parentSpanCtx))

	return opentracing.ContextWithSpan(ctx, span), nil
}

// ExtractMesh ...
func ExtractMesh(ctx context.Context, headers []*sarama.RecordHeader) (context.Context, map[string]string) {
	if len(headers) == 0 {
		return ctx, nil
	}

	var val string
	hdrs := make(map[string]string)
	for i := range headers {
		hdr := *headers[i]
		key := string(hdr.Key)
		if strings.HasPrefix(strings.ToLower(key), header) {
			if key == headerMesh {
				val = string(hdr.Value)
			}

			hdrs[key] = string(hdr.Value)
		}

	}

	var out map[string]string
	if val != "" {
		out = make(map[string]string)
		meshes := strings.Split(val, ";")
		for _, mesh := range meshes {
			servs := strings.Split(mesh, ":")
			out[servs[0]] = servs[1]
		}
	}

	ctx = metadata.AppendToOutgoingContext(ctx, dataToPairs(hdrs)...)

	return context.WithValue(ctx, keyData, hdrs), out
}

// KafkaHeaders ...
type KafkaHeaders []sarama.RecordHeader

// ForeachKey implement opentracing TextMapReader
func (k *KafkaHeaders) ForeachKey(handler func(key, val string) error) error {
	for _, val := range *k {
		key := string(val.Key)
		value := string(val.Value)
		if err := handler(key, value); err != nil {
			return err
		}
	}
	return nil
}

// Set implement opentracing TextMapWriter
func (k *KafkaHeaders) Set(key, val string) {
	i := []byte(key)
	v := []byte(val)
	for j := range *k {
		if bytes.Equal(i, (*k)[j].Key) {
			(*k)[j].Value = v
			return
		}
	}
	*k = append(*k, sarama.RecordHeader{
		Key:   i,
		Value: v,
	})
}

type keyType string

const (
	header             = ""
	headerMesh         = ""
	keyData    keyType = "a"
)

func dataToPairs(d map[string]string) []string {
	ret := make([]string, 0, len(d)*2)
	for k, v := range d {
		ret = append(ret, k, v)
	}
	return ret
}
