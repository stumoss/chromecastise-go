# Chromecastise

A simple command line application to transcode video files so that they are
able to be played by chromecast devices.

```bash
Usage:
        chromecastise [--mp4 | --mkv] [--suffix=<suffix>] <file>...

Arguments:
        <file>    The file you wish to transcode.

Options:
        -h --help            Show this screen.
        --version            Show version.
        --mp4                Convert to mp4 container format [default: true].
        --mkv                Convert to mkv container format.
        --suffix=<suffix>    The file suffix to append to the filename (before the file extension) [default: _new]
```
