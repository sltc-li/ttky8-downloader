package main

import (
	"encoding/json"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"os/exec"
	"strconv"
	"time"
)

type server struct {
	downloadURLs []DownloadURL
}

func (s *server) Start() error {
	rand.Seed(time.Now().UnixNano())
	addr := ":" + strconv.Itoa(10_000+rand.Intn(10_000))
	http.Handle("/", s.HandleIndex(addr))
	http.Handle("/update", s.HandleUpdate())
	errChan := make(chan error)
	go func() {
		errChan <- http.ListenAndServe(addr, nil)
	}()
	timer := time.NewTimer(time.Second)
	select {
	case err := <-errChan:
		return err
	case <-timer.C:
		return exec.Command("open", "http://localhost"+addr).Run()
	}
}

func (s *server) HandleIndex(addr string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// language=html
		html := `
<!doctype html>
<html>
<head>
  <script type="text/javascript">
    function update() {
      var xhr = new XMLHttpRequest();
      xhr.onreadystatechange = function() {
        if (xhr.readyState === 4){
          var sizes = JSON.parse(xhr.responseText);
          for (var i=0; i<sizes.length; i++) {
            document.getElementById("size_"+i).innerText = sizes[i];
          }
        }
      };
      xhr.open('GET', "{{.updateURL}}");
      xhr.send();
    }
    setInterval(update, 1000);
  </script>
</head>
<body>
<table border="1">
	<tr><th>File</th><th>Size</th></tr>
	{{range $i, $du := .downloadURLs}}
	<tr>
		<td align="center">{{$du.Title}}</td>
		<td align="center" style="min-width: 100px;" id="size_{{$i}}">{{$du.DownloadedSize}}</td>
	</tr>
	{{end}}
</table>
</body>
</html>
`
		tmpl := template.New("index.html")
		if _, err := tmpl.Parse(html); err != nil {
			log.Printf("fail to parse index html: %v", err)
		}
		err := tmpl.Execute(w, map[string]interface{}{
			"downloadURLs": s.downloadURLs,
			"updateURL":    "http://localhost" + addr + "/update",
		})
		if err != nil {
			log.Printf("fail to handle index: %v", err)
		}
	}
}

func (s *server) HandleUpdate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var sizes []string
		for _, du := range s.downloadURLs {
			sizes = append(sizes, du.DownloadedSize())
		}
		err := json.NewEncoder(w).Encode(sizes)
		if err != nil {
			log.Printf("fail to handle update: %v", err)
		}
	}
}
