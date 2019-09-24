package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalln("provide a url or JSON file to download")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var downloadURLs []DownloadURL

	if strings.HasPrefix(os.Args[1], "http://") || strings.HasPrefix(os.Args[1], "https://") {
		var err error
		downloadURLs, err = collectDownloadURLs(ctx, os.Args[1])
		if err != nil {
			log.Fatalf("fail to collect download urls: %v", err)
		}
		bs, err := json.MarshalIndent(downloadURLs, "", "  ")
		if err != nil {
			log.Fatalf("fail to marshal json: %v", err)
		}
		if err := ioutil.WriteFile(downloadURLs[0].Title+".json", bs, 0644); err != nil {
			log.Fatalf("fail to write file: %v", err)
		}
	} else {
		bs, err := ioutil.ReadFile(os.Args[1])
		if err != nil {
			log.Fatalf("fail to read file: %v", err)
		}
		if err := json.Unmarshal(bs, &downloadURLs); err != nil {
			log.Fatalf("fail to unmarshal json: %v", err)
		}
	}

	svr := NewServer(downloadURLs)
	go func() {
		log.Print("listening " + svr.URL())
		if err := svr.Start(); err != nil {
			log.Printf("fail to start server: %v", err)
		}
	}()

	go func() {
		dpCtx, cancel := newChromedp(ctx, false)
		defer cancel()
		if err := chromedp.Run(dpCtx, chromedp.Navigate(svr.URL())); err != nil {
			log.Printf("fail to open server url: %v", err)
		}
		<-ctx.Done()
	}()

	download(downloadURLs, 5)

	signChan := make(chan os.Signal, 1)
	signal.Notify(signChan, os.Interrupt)
	<-signChan
}

func download(downloadURLs []DownloadURL, nParallel int) {
	var wg sync.WaitGroup
	downloadingChan := make(chan bool, nParallel)

	for _, du := range downloadURLs {
		wg.Add(1)
		downloadingChan <- true
		go func(du DownloadURL) {
			if err := du.Download(); err != nil {
				log.Fatalf("fail to download %s: %v", du.Title, err)
			}
			<-downloadingChan
			wg.Done()
		}(du)
	}

	wg.Wait()
	close(downloadingChan)
}

func collectDownloadURLs(ctx context.Context, startURL string) ([]DownloadURL, error) {
	var (
		baseURL    = strings.Split(startURL, ".com/")[0] + ".com"
		mp4URLChan = make(chan string, 2)
		titleChan  = make(chan string, 2)
		quit       = make(chan bool)

		downloadURLs []DownloadURL
	)

	ctx, cancel := newChromedp(ctx, true)
	defer cancel()

	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch ev := ev.(type) {
		case *network.EventRequestWillBeSent:
			url := ev.Request.URL
			if strings.HasSuffix(url, ".m3u8") {
				mp4URLChan <- url
			}
		}
	})

	sendTitle := chromedp.ActionFunc(func(ctx context.Context) error {
		var title string
		if err := chromedp.Text(".video-title > h1", &title, chromedp.NodeVisible).Do(ctx); err != nil {
			return err
		}
		title = strings.TrimRight(title, " 正在观看")
		titleChan <- title
		return nil
	})

	appendDownloadURL := func(url string) chromedp.Action {
		return chromedp.ActionFunc(func(ctx context.Context) error {
			downloadURLs = append(downloadURLs, DownloadURL{
				URL:    url,
				Title:  <-titleChan,
				Mp4URL: <-mp4URLChan,
			})
			return nil
		})
	}

	var nodes []*cdp.Node
	err := chromedp.Run(ctx,
		network.Enable(),
		chromedp.Navigate(startURL),
		sendTitle,
		appendDownloadURL(startURL),
		chromedp.Nodes(".play-list > .zhwli_1 > a", &nodes, chromedp.AtLeast(0)),
		chromedp.ActionFunc(func(ctxt context.Context) error {
			var playURLs []string
			for _, node := range nodes {
				playURL := baseURL + node.AttributeValue("href")
				if playURL == startURL {
					continue
				}
				playURLs = append(playURLs, playURL)
			}

			go func() {
				for _, playURL := range playURLs {
					_ = appendDownloadURL(playURL).Do(ctxt)
				}
				close(mp4URLChan)
				close(quit)
			}()

			for _, playURL := range playURLs {
				if err := (chromedp.Tasks{chromedp.Navigate(playURL), sendTitle}).Do(ctxt); err != nil {
					return err
				}
			}
			return nil
		}),
	)
	if err != nil {
		return nil, err
	}

	<-quit
	return downloadURLs, nil
}

func newChromedp(ctx context.Context, headless bool) (context.Context, context.CancelFunc) {
	var opts []chromedp.ExecAllocatorOption
	for _, opt := range chromedp.DefaultExecAllocatorOptions {
		opts = append(opts, opt)
	}
	if !headless {
		opts = append(opts,
			chromedp.Flag("headless", false),
			chromedp.Flag("hide-scrollbars", false),
			chromedp.Flag("mute-audio", false),
		)
	}

	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx, opts...)
	ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))

	return ctx, func() {
		cancel()
		allocCancel()
	}
}
