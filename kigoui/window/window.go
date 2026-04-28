package window

import (
	"context"
	"image"
	"sync"

	"github.com/AgentNemo00/kigo-ui/paint"
	"github.com/AgentNemo00/sca-instruments/log"
	"github.com/gogpu/gg"
	"github.com/gogpu/gg/integration/ggcanvas"
	"github.com/gogpu/gogpu"
)

type Window struct {
	app 	*gogpu.App
	mu 		*sync.Mutex
	frames 	*map[string]Frame
}

func NewWindow(ctx context.Context) (*Window ,error) {
	w := &Window{}
	app := gogpu.NewApp(gogpu.DefaultConfig().
		WithTitle(" ").
		WithSize(1080, 900). // TODO: make automated fullscreen
		WithContinuousRender(false))
	var canvas *ggcanvas.Canvas
	mu := new(sync.Mutex)
	frames := make(map[string]Frame, 0)
	w.app = app
	w.frames = &frames
	w.mu = mu

	app.OnDraw(func(dc *gogpu.Context) {
		w, h := dc.Width(), dc.Height()
		if w <= 0 || h <= 0 {
			return
		}

		// Lazy canvas init.
		if canvas == nil {
			provider := app.GPUContextProvider()
			if provider == nil {
				return
			}
			var err error
			canvas, err = ggcanvas.New(provider, w, h)
			if err != nil {
				log.Ctx(ctx).Err(err)
				return
			}
		}

		cc := canvas.Context()

		// Dark background.
		cc.SetRGBA(0.08, 0.08, 0.12, 1)
		cc.Clear()

		mu.Lock()
		log.Ctx(ctx).Debug("rendering %d frames", len(frames))
		for _, frame := range frames {
			if frame.Data == nil {
				continue
			}
			buf := gg.ImageBufFromImage(frame.Data)
			cc.DrawImage(buf, float64(frame.PositionX), float64(frame.PositionY))
		}
		mu.Unlock()

		// Present: upload pixmap to GPU texture and blit to surface.
		canvas.MarkDirty()
		rt := dc.RenderTarget()
		if err := canvas.Render(rt); err != nil {
			log.Ctx(ctx).Err(err)
		}
	})

	w.app.OnClose(func() {
		gg.CloseAccelerator()
		if canvas != nil {
			err := canvas.Close()
			log.Ctx(ctx).Err(err)
		}
	})
	return w, nil
}

func (w *Window) Stop() {
	w.app.Close()
}

func (w *Window) Start(ctx context.Context) {
	if err := w.app.Run(); err != nil {
		log.Ctx(ctx).Err(err)
	}
}

func (w *Window) Draw() {
	w.app.RequestRedraw()
}

func (w *Window) Add(pkg paint.Package) error {
	img := &image.RGBA{
		Pix:    pkg.Data,
		Stride: 4 * pkg.Width,
		Rect:   image.Rect(0, 0, pkg.Width, pkg.Height),
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	(*w.frames)[pkg.ID] = Frame{
		Data: img,
		PositionX: pkg.PositionX,
		PositionY: pkg.PositionY,
	}

	return nil
}

func (w *Window) Remove(id string) {
	delete(*w.frames, id)
}

func (w *Window) Clear() {
	w.mu.Lock()
	defer w.mu.Unlock()
	for k := range *w.frames {
		delete(*w.frames, k)
	}
}
