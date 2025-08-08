package hash

import (
	"bytes"
	"context"
	"encoding/gob"
	"hash/fnv"
	"strconv"

	"github.com/gfa-inc/gfa/common/logger"
)

func Hash[T any](ctx context.Context, value T) (string, error) {
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(value)
	if err != nil {
		logger.TError(ctx, err)
		return "", err
	}

	h := fnv.New64a()
	_, err = h.Write(buf.Bytes())
	if err != nil {
		logger.TError(ctx, err)
		return "", err
	}

	hashStr := strconv.FormatUint(h.Sum64(), 16)
	return hashStr, nil
}
