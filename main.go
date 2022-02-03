package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/alecthomas/kong"
	log "github.com/sirupsen/logrus"
)

// Map of file extensions and if they are supported or not
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

// Map of formats and if they are supported or not
var formats = map[string]bool{
	"MPEG-4":      true,
	"Matroska":    true,
	"BDAV":        false,
	"AVI":         false,
	"Flash Video": false,
	"Unknown":     false,
}

var videoCodecs = map[string]bool{
	"AVC":           true,
	"MPEG-4 Visual": false,
	"xvid":          false,
	"MPEG Video":    false,
}

var audioCodecs = map[string]bool{
	"AAC":        true,
	"MPEG Audio": false, // true, Changed to false for iOS
	"Vorbis":     false, // true, Changed to false for iOS
	"Ogg":        false, // true, Changed to false for iOS
	"AC-3":       false,
	"DTS":        false,
	"PCM":        false,
}

const (
	defaultVideoCodec = "libx264" // Default video codec to convert to
	defaultAudioCodec = "aac"     // Default audio codec to convert to
)

type CLI struct {
	Transcoder Transcoder `embed:""`
	Log        struct {
		Level string `enum:"trace,debug,info,warn,error,fatal,panic" default:"info"`
		Type  string `enum:"json,console,none" default:"console"`
	} `embed:"" prefix:"logging."`
}

func (c *CLI) AfterApply() error {
	lvl, err := log.ParseLevel(c.Log.Level)
	if err != nil {
		return err
	}

	log.SetLevel(lvl)

	switch c.Log.Type {
	case "console":
		log.SetFormatter(&log.TextFormatter{})
	case "json":
		log.SetFormatter(&log.JSONFormatter{})
	case "none":
		log.SetFormatter(&log.TextFormatter{})
		log.SetOutput(io.Discard)
	}
	return nil
}

type Transcoder struct {
	Format string   `enum:"mp4,mkv" default:"mkv"`
	Files  []string `arg:"" name:"files" help:"files to convert." type:"path"`
}

func (c *CLI) Run() error {
	log.WithField("files", c.Transcoder.Files).Debug("file transcode starting")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	for _, f := range c.Transcoder.Files {
		if err := processFile(ctx, filepath.Clean(f), c.Transcoder.Format); err != nil {
			log.Println(err)
		}
	}
	return nil
}

func main() {
	ctx := kong.Parse(&CLI{})
	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}

func processFile(ctx context.Context, p string, format string) error {
	// Check for supported extension
	extension := filepath.Ext(p)
	if !isSupported(extension, fileExtensions) {
		return fmt.Errorf("[%s] unsupported video format found", p)
	}
	extension = strings.TrimPrefix(extension, ".")

	// Set container format
	outputContainerFormat := format
	_, err := exec.CommandContext(ctx, "mediainfo", "--Inform=General;%Format%", p).CombinedOutput()
	if err != nil {
		return fmt.Errorf("[%s] mediainfo failed to get the container format: %s", p, err)
	}

	// Set video encoding
	outVideoCodec := defaultVideoCodec
	out, err := exec.CommandContext(ctx, "mediainfo", "--Inform=Video;%Format%", p).CombinedOutput()
	if err != nil {
		return fmt.Errorf("[%s] mediainfo failed to get the encoding format: %s", p, err)
	}

	videoCodec := strings.TrimSpace(string(out))
	if isSupported(videoCodec, videoCodecs) {
		outVideoCodec = "copy"
	}

	// Set audio encoding
	outAudioCodec := defaultAudioCodec
	out, err = exec.CommandContext(ctx, "mediainfo", "--Inform=Audio;%Format%", p).CombinedOutput()
	if err != nil {
		log.Fatal(err)
		return fmt.Errorf("[%s] mediainfo failed to get the audio encoding format: %s", p, err)
	}

	audioCodec := strings.TrimSpace(string(out))
	if isSupported(audioCodec, audioCodecs) {
		outAudioCodec = "copy"
	}

	if outVideoCodec == "copy" && outAudioCodec == "copy" && extension == format {
		log.Printf("[%s] no conversion required", p)
		return nil
	}

	// Convert the file
	basename := strings.TrimSuffix(filepath.Base(p), extension)

	args := []string{
		"ffmpeg", "-threads", "4", "-i", p, "-map",
		"0:0", "-c:v", outVideoCodec, "-preset", "slow", "-level", "4.0",
		"-crf", "20", "-bf", "16", "-b_strategy", "2", "-subq", "10",
		"-map", "0:1", "-c:a:0", outAudioCodec, "-b:a:0", "128k",
		"-strict", "-2", "-y",
	}

	args = append(args, filepath.Join(filepath.Dir(p), basename+"_new."+outputContainerFormat))

	if outputContainerFormat == "mkv" {
		args = append(args, "-c:s copy")
	}

	output, err := exec.CommandContext(ctx, "ffmpeg", args...).Output()
	if err != nil {
		return fmt.Errorf("[%s] ffmpeg failed to transcode the file: %s", p, err)
	}

	fmt.Println(string(output))

	return nil
}

func isSupported(s string, l map[string]bool) bool {
	v, ok := l[s]
	if ok {
		return v
	}
	return false
}
