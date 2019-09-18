package main

import (
	"encoding/json"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

func NewServer(downloadURLs []DownloadURL) *server {
	rand.Seed(time.Now().UnixNano())
	addr := ":" + strconv.Itoa(10_000+rand.Intn(10_000))
	return &server{
		downloadURLs: downloadURLs,
		addr:         addr,
	}
}

type server struct {
	downloadURLs []DownloadURL
	addr         string
}

func (s *server) URL() string {
	return "http://localhost" + s.addr
}

func (s *server) Start() error {
	http.Handle("/", s.HandleIndex(s.addr))
	http.Handle("/update", s.HandleUpdate())
	return http.ListenAndServe(s.addr, nil)
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
