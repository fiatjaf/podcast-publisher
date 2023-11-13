package main

import (
	"crypto/md5"
	"encoding/hex"
	"log"
	"os"
	"path/filepath"

	"github.com/adrg/frontmatter"
	"github.com/jbub/podcasts"
	"github.com/russross/blackfriday/v2"
)

type PodcastMetadata struct {
	Title        string `yaml:"title"`
	BaseURL      string `yaml:"base_url"`
	EpisodesPath string `yaml:"episodes_path"`
	Author       string `yaml:"author"`
}

type EpisodeMetadata struct {
	Title  string `yaml:"title"`
	Author string `yaml:"author"`
}

func main() {
	file, err := os.Open("podcast.md")
	if err != nil {
		log.Printf("failed to read podcast.md: %s\n", err)
		return
	}

	var m PodcastMetadata
	rest, err := frontmatter.Parse(file, &m)
	if err != nil {
		log.Printf("file is malformed yaml: %s\n", err)
		return
	}

	podcast := &podcasts.Podcast{
		Title:       m.Title,
		Description: string(blackfriday.Run(rest)),
		Link:        m.BaseURL,
	}

	entries, err := os.ReadDir(m.EpisodesPath)
	if err != nil {
		log.Printf("failed to read directory '%s': %s\n", m.EpisodesPath, err)
		return
	}

	for _, d := range entries {
		if !d.IsDir() {
			continue
		}

		audiopath := filepath.Join(m.EpisodesPath, d.Name(), "audio.mp3")
		metapath := filepath.Join(m.EpisodesPath, d.Name(), "episode.md")

		b, _ := os.ReadFile(audiopath)
		hash := md5.Sum(b)

		var meta EpisodeMetadata
		metafile, err := os.Open(metapath)
		if err != nil {
			log.Printf("couldn't open episode metadata (file '%s'): %s\n", metapath, err)
			continue
		}

		info, _ := os.Stat(audiopath)
		rest, err := frontmatter.Parse(metafile, &meta)
		if err != nil {
			log.Printf("failed to decode frontmatter: %s\n", err)
			continue
		}

		author := meta.Author
		if author == "" {
			author = m.Author
		}

		podcast.AddItem(&podcasts.Item{
			Title:   meta.Title,
			PubDate: &podcasts.PubDate{Time: info.ModTime()},
			GUID:    hex.EncodeToString(hash[:]),
			Enclosure: &podcasts.Enclosure{
				URL:  m.BaseURL + audiopath,
				Type: "audio/mp3",
			},
			Author: author,
			Summary: &podcasts.ItunesSummary{
				Value: string(blackfriday.Run(rest)),
			},
		})
	}

	feed, err := podcast.Feed()
	if err != nil {
		log.Printf("error making podcast feed: %s\n", err)
		return
	}

	out := filepath.Join(m.EpisodesPath, "feed.xml")
	outfile, err := os.Create(out)
	if err != nil {
		log.Printf("error opening xml file: %s\n", err)
		return
	}

	err = feed.Write(outfile)
	if err != nil {
		log.Printf("error writing feed: %s\n", err)
		return
	}

	log.Println("feed generated at " + out)
}
