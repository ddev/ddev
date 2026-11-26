package main

import (
	"bytes"
	"image/color"
	"image/png"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStateOf(t *testing.T) {
	up := project{Name: "a", Status: "running", Up: true}
	down := project{Name: "b", Status: "stopped"}
	starting := project{Name: "c", Status: "starting", Up: true}
	require.Equal(t, trayIdle, stateOf(nil, 0))
	require.Equal(t, trayIdle, stateOf([]project{down}, 0))
	require.Equal(t, trayRunning, stateOf([]project{down, up}, 0))
	require.Equal(t, trayBusy, stateOf([]project{down, up}, 1))
	require.Equal(t, trayBusy, stateOf([]project{down, starting}, 0))
}

func TestWithDot(t *testing.T) {
	for _, icon := range [][]byte{trayIcon, trayIconColor} {
		out, err := withDot(icon, dotColors[trayRunning])
		require.NoError(t, err)
		img, err := png.Decode(bytes.NewReader(out))
		require.NoError(t, err)
		b := img.Bounds()
		size := b.Dx()
		r := float64(size) * 0.18
		center := int(float64(size) - r - 1)
		require.Equal(t, color.NRGBAModel.Convert(dotColors[trayRunning]), color.NRGBAModel.Convert(img.At(center, center)))
		// The ring around the dot is cut out of the mark.
		ringX := int(float64(size) - 2*r - 1 - float64(size)*0.035)
		_, _, _, a := img.At(ringX, center).RGBA()
		require.Zero(t, a, "the ring at x=%d of a %dpx icon", ringX, size)
		// The top left of the mark is untouched.
		orig, err := png.Decode(bytes.NewReader(icon))
		require.NoError(t, err)
		require.Equal(t, orig.At(size/4, size/4), img.At(size/4, size/4))
	}
}

func TestParseDotLabel(t *testing.T) {
	parts, err := parseDotLabel("\033[38;2;46;204;113m●\033[0m")
	require.NoError(t, err)
	require.Equal(t, "●", parts[0].Text)
	require.Equal(t, "#2ecc71", parts[0].FgColor)
}

func TestScaleDown(t *testing.T) {
	for _, size := range []int{16, 20, 24, 32} {
		out, err := scaleDown(trayIconColor, size)
		require.NoError(t, err)
		img, err := png.Decode(bytes.NewReader(out))
		require.NoError(t, err)
		require.Equal(t, size, img.Bounds().Dx())
		require.Equal(t, size, img.Bounds().Dy())
	}
	// An icon no larger than the size stays as it is.
	out, err := scaleDown(trayIconColor, 128)
	require.NoError(t, err)
	require.Equal(t, trayIconColor, out)
}
