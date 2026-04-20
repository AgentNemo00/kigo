package frame

import (
	"context"
	"os"
	"path"

	"github.com/AgentNemo00/sca-instruments/log"
	"github.com/EBWi11/mmap_ringbuffer"
)

type IPC struct {
	path string
}

func NewIPC(path string) (*IPC, error) {
	if !pathExists(path) {
		err := os.MkdirAll(path, 0755)
		if err != nil {
			return nil, err
		}
	}
	return &IPC{path: path}, nil
}

func (i *IPC) Open(ctx context.Context, name string, frameSize int) (*Frame, error) {
	rb, err := ringbuffer.NewRingBuffer(path.Join(i.path, name), frameSize, true)
    if err != nil {
        return nil, err
    }
	return &Frame{
		read: func () ([]byte, error) {
		for {
			// blocking
			select{
			case <- ctx.Done():
				return nil, ctx.Err()
			default:
				msg, err := rb.ReadMsg()
				if err == ringbuffer.ErrBufferEmpty {
					continue
				}
				if err == ringbuffer.ErrClosed {
					return nil, nil
				}
				if err != nil {
					return nil, err
				}
				return msg, nil
			}
		}
		},
		close: func ()  {
			err := rb.Close()
			if err != nil {
				log.Ctx(ctx).Err(err)
			}
		},
	}, nil
}

func pathExists(path string) bool {
    _, err := os.Stat(path)
    if err == nil {
        return true
    }
    return !os.IsNotExist(err)
}