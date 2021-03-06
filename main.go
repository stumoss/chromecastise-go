package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	docopt "github.com/docopt/docopt-go"
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
	"MPEG Audio": false, //true, Changed to false for iOS
	"Vorbis":     false, //true, Changed to false for iOS
	"Ogg":        false, //true, Changed to false for iOS
	"AC-3":       false,
	"DTS":        false,
	"PCM":        false,
}

// Default video codec to convert to
const defaultVideoCodec = "libx264"

// Default audio codec to convert to
const defaultAudioCodec = "aac"

const usage = `chromecastise 1.0.0

Usage:
	chromecastise [--mp4 | --mkv] <file>...

Arguments:
	<file>	The file you wish to transcode for chromecast compatibility.

Options:
	-h --help     Show this screen.
	--version     Show version.
	--mp4         Convert to mp4 container format [default: true]
	--mkv         Convert to mkv container format [default: false]
`

func main() {
	arguments, err := docopt.ParseArgs(usage, nil, "chromecastise 1.0.0")
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	format := "mp4"
	if arguments["--mkv"].(bool) {
		format = "mkv"
	}

	for _, f := range arguments["<file>"].([]string) {
		if err := processFile(filepath.Clean(f), format); err != nil {
			log.Println(err)
		}
	}
}

func processFile(p string, format string) error {
	// Check for supported extension
	extension := filepath.Ext(p)
	if !isSupported(extension, fileExtensions) {
		return fmt.Errorf("[%s] unsupported video format found", p)
	}
	extension = strings.TrimPrefix(extension, ".")

	// Set container format
	outputContainerFormat := format
	out, err := exec.Command("mediainfo", "--Inform=General;%Format%", p).CombinedOutput()
	if err != nil {
		return fmt.Errorf("[%s] mediainfo failed to get the container format: %s", p, err)
	}

	// Set video encoding
	outVideoCodec := defaultVideoCodec
	out, err = exec.Command("mediainfo", "--Inform=Video;%Format%", p).CombinedOutput()
	if err != nil {
		return fmt.Errorf("[%s] mediainfo failed to get the encoding format: %s", p, err)
	}

	videoCodec := strings.TrimSpace(string(out))
	if isSupported(videoCodec, videoCodecs) {
		outVideoCodec = "copy"
	}

	// Set audio encoding
	outAudioCodec := defaultAudioCodec
	out, err = exec.Command("mediainfo", "--Inform=Audio;%Format%", p).CombinedOutput()
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

	args := []string{"ffmpeg", "-threads", "4", "-i", p, "-map",
		"0:0", "-c:v", outVideoCodec, "-preset", "slow", "-level", "4.0",
		"-crf", "20", "-bf", "16", "-b_strategy", "2", "-subq", "10",
		"-map", "0:1", "-c:a:0", outAudioCodec, "-b:a:0", "128k",
		"-strict", "-2", "-y"}

	args = append(args, filepath.Join(filepath.Dir(p), basename+"_new."+outputContainerFormat))

	if outputContainerFormat == "mkv" {
		args = append(args, "-c:s copy")
	}

	output, err := exec.Command("ffmpeg", args...).Output()

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
