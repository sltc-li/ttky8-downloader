package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
)

func formatFileSize(size int64) string {
	units := []string{"G", "M", "K"}
	sizePerUnits := []int64{1_000_000_000, 1_000_000, 1_000}
	for i, unit := range units {
		if size >= sizePerUnits[i] {
			if size/sizePerUnits[i] >= 10 {
				return strconv.FormatInt(size/sizePerUnits[i], 10) + unit
			}
			return fmt.Sprintf("%.1f%s", float64(size)/float64(sizePerUnits[i]), unit)
		}
	}
	return strconv.FormatInt(size, 10)
}

type DownloadURL struct {
	URL    string `json:"url"`
	Title  string `json:"title"`
	Mp4URL string `json:"mp4_url"`
}

func (du DownloadURL) Mp4File() string {
	return du.Title + ".mp4"
}

func (du DownloadURL) DownloadedSize() string {
	if info, err := os.Stat(du.Mp4File()); err == nil {
		return formatFileSize(info.Size())
	}
	return "0"
}

func (du DownloadURL) Download() error {
	downloadedSize := du.DownloadedSize()
	if downloadedSize != "0" {
		log.Printf("%s already downloaded %s", du.Mp4File(), downloadedSize)
		return nil
	}

	shCmd := fmt.Sprintf(`ffmpeg -i "%s" -vcodec copy -c copy -c:a aac "%s"`, du.Mp4URL, du.Mp4File())
	cmd := exec.Command("sh", "-c", shCmd)
	err := cmd.Run()
	if err != nil {
		return err
	}
	log.Printf("%s downloaded %s", du.Mp4File(), du.DownloadedSize())
	return nil
}
