package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
)

// fileExtensions shows what file extensions are supported
var fileExtensions = map[string]bool{
	".mkv":  true,
	".avi":  true,
	".mp4":  true,
	".3gp":  true,
	".mov":  true,
	".mpg":  true,
	".mpeg": true,
	".qt":   true,
	".wmv":  true,
	".m2ts": true,
	".flv":  true,
}

var videoCodecs = map[string]bool{
	"AVC":           true,
	"HEV":           true,
	"MPEG-4 Visual": false,
	"xvid":          false,
	"MPEG Video":    false,
}

var audioCodecs = map[string]bool{
	"AAC":        true,
	"MPEG Audio": false,
	"Vorbis":     false,
	"Ogg":        false,
	"AC-3":       false,
	"DTS":        false,
	"PCM":        false,
}

// ContainerFormat defines the supported container formats.
type ContainerFormat int

func (c ContainerFormat) String() string {
	return [...]string{"undefined", "mp4", "mkv"}[c]
}

const (
	// UNDEFINED is used for the container format when it has not been set
	UNDEFINED ContainerFormat = iota
	// MP4 is the mpeg version 4 container format
	MP4
	// MKV is the matroska container format
	MKV
)

// Default video codec to convert to
const defaultVideoCodec = "libx264"

// Default audio codec to convert to
const defaultAudioCodec = "aac"

// The version
var appVersion = "undefined"

func main() {
	mp4 := pflag.Bool("mp4", false, "whether or not to use mp4 container format")
	mkv := pflag.Bool("mkv", true, "wether or not to use mkv container format")
	version := pflag.Bool("version", false, "show version")
	suffix := pflag.String("suffix", "_new", "the file name suffix to append to the file")
	help := pflag.BoolP("help", "h", false, "show this help message")

	pflag.Parse()

	if *version {
		if appVersion == "undefined" {
			fmt.Printf("chromecastise %s", appVersion)
		} else {
			fmt.Printf("chromecastise v%s", appVersion)
		}
		os.Exit(0)
	}

	if *help || pflag.NArg() == 0 {
		pflag.Usage()
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	defer func() {
		signal.Stop(c)
		cancel()
	}()
	go func() {
		select {
		case <-c:
			cancel()
		case <-ctx.Done():
		}
	}()

	format := MKV
	if *mkv {
		format = MKV
	} else if *mp4 {
		format = MP4
	}

	for _, f := range pflag.Args() {
		if err := processFile(ctx, filepath.Clean(f), format, *suffix); err != nil {
			log.Println(err)
		}
	}
}

func processFile(ctx context.Context, p string, format ContainerFormat, fileSuffix string) error {
	// Check for supported extension
	extension := filepath.Ext(p)
	if !isSupported(extension, fileExtensions) {
		return fmt.Errorf("[%s] unsupported video format found", p)
	}
	extension = strings.TrimPrefix(extension, ".")

	// Set container format
	outputContainerFormat := format

	// Set video encoding
	outVideoCodec := defaultVideoCodec
	out, err := exec.Command("mediainfo", "--Inform=Video;%Format%", p).CombinedOutput()
	if err != nil {
		return fmt.Errorf("[%s] mediainfo failed to get the encoding format: %s", p, err)
	}

	videoCodec := strings.TrimSpace(string(out))
	if isSupported(videoCodec, videoCodecs) {
		outVideoCodec = "copy"
	}

	// Set audio encoding
	outAudioCodec := defaultAudioCodec
	_, err = exec.CommandContext(ctx, "mediainfo", "--Inform=Audio;%Format%", p).CombinedOutput()
	if err != nil {
		log.Fatal(err)
		return fmt.Errorf("[%s] mediainfo failed to get the audio encoding format: %s", p, err)
	}

	audioCodec := strings.TrimSpace(string(out))
	if isSupported(audioCodec, audioCodecs) {
		outAudioCodec = "copy"
	}

	if outVideoCodec == "copy" && outAudioCodec == "copy" && extension == format.String() {
		log.Printf("[%s] no conversion required", p)
		return nil
	}

	// Convert the file
	basename := strings.TrimSuffix(filepath.Base(p), "."+extension)

	args := []string{
		"-threads", strconv.Itoa(runtime.NumCPU()), "-i", p, "-map", "0:a?", "-map",
		"0:s?", "-map", "0:v", "-c:v", outVideoCodec, "-c:a", outAudioCodec,
		"-c:s", "copy",
	}

	args = append(args, filepath.Join(filepath.Dir(p), basename+fileSuffix+"."+outputContainerFormat.String()))

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("ffmpeg failed to transcode the file with command: \n\t%s %s\n\terror: %s", "ffmpeg", args, err)
	}

	return nil
}

func isSupported(s string, l map[string]bool) bool {
	v, ok := l[s]
	if ok {
		return v
	}
	return false
}
