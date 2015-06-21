package main

import (
	"bytes"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const (
	SoundFolder = "snd"
)

type CTL string

const (
	Stop CTL = ""
)

func main() {
	var (
		control = make(chan CTL, 3)
		errs    = make(chan error, 30)
	)
	go func() {
		for {
			err := <-errs
			log.Println(err)
		}
	}()

	os.Mkdir(SoundFolder, 0700)

	go playLoop(control, errs)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		type toT struct {
			List []string
		}
		var err error
		to := toT{}
		folder, err := os.Open(SoundFolder)
		if err != nil {
			http.Error(w, err.Error(), 200)
			return
		}
		to.List, err = folder.Readdirnames(-1)
		folder.Close()
		if err != nil {
			http.Error(w, err.Error(), 200)
			return
		}
		buf := &bytes.Buffer{}
		err = indexT.Execute(buf, to)
		if err != nil {
			http.Error(w, err.Error(), 200)
			return
		}
		w.Write(buf.Bytes())
	})
	var bbOk = []byte("OK")
	http.HandleFunc("/api/stop", func(w http.ResponseWriter, r *http.Request) {
		control <- Stop
		w.Write(bbOk)
	})
	http.HandleFunc("/api/choose", func(w http.ResponseWriter, r *http.Request) {
		control <- CTL(r.FormValue("name"))
		w.Write(bbOk)
	})
	http.HandleFunc("/api/load", func(w http.ResponseWriter, r *http.Request) {
		file, header, err := r.FormFile("sound")
		if err != nil {
			http.Error(w, err.Error(), 200)
			return
		}
		defer file.Close()

		f, err := os.OpenFile(filepath.Join(SoundFolder, header.Filename), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
		if err != nil {
			http.Error(w, err.Error(), 200)
			return
		}
		defer f.Close()

		io.Copy(f, file)

		w.Write(bbOk)
	})
	http.ListenAndServe(":9000", nil)
}

type LoopCmd struct {
	path string
	args []string
	cmd *exec.Cmd

	stop chan struct{}
	next chan struct{}
}

func LoopCmdStart(cmd string, args ...string) (*LoopCmd, error) {
	lc := &LoopCmd{
		path: cmd,
		args: args,
		stop: make(chan struct{}, 3),
		next: make(chan struct{}, 3),
	}
	err := lc.start()
	go lc.run()
	return lc, err
}

func (lc *LoopCmd) Stop() {
	if lc == nil {
		return
	}
	close(lc.stop)
	if lc.cmd == nil || lc.cmd.Process == nil {
		return
	}
	lc.cmd.Process.Kill()
}

func (lc *LoopCmd) start() error {
	cmd := exec.Command(lc.path, lc.args...)
	lc.cmd = cmd
	err := cmd.Start()
	if err == nil {
		go lc.wait(cmd)
	}
	return err
}

func (lc *LoopCmd) wait(cmd *exec.Cmd) {
	cmd.Wait()
	lc.next <- struct{}{}
}

func (lc *LoopCmd) run() {
	tick := time.NewTicker(time.Millisecond * 15)
	for {
		select {
		case <-lc.next:
			err := lc.start()
			if err != nil {
				log.Print(err)
			}
		case <-lc.stop:
			tick.Stop()
			return
		}
	}
}

func playLoop(control chan CTL, errs chan error) {
	var cmd *LoopCmd
	var err error

	for {
		select {
		case ctl := <-control:
			switch ctl {
			default:
				cmd.Stop()
				cmd, err = LoopCmdStart("play", filepath.Join(SoundFolder, string(ctl)))
				if err != nil {
					errs <- err
					continue
				}
			case Stop:
				cmd.Stop()
				cmd = nil
			}
		}
	}
}

var indexT = template.Must(template.New("").Parse(index))

var index = `<!doctype html>
<html>
<head>
	<title>BG Music</title>
	
	<script src="https://code.jquery.com/jquery-2.1.4.min.js"></script>
	
<style>
#list li {
	cursor: pointer;
}
#list li:hover {
	background: lightgray;
}
</style>
</head>
<body>
	<h2>BG Music</h2>
	
	<!--<button id="start">Start</button>-->
	<button id="stop">Stop</button>
	
	<br>
	<ul id="list">
	{{range $.List}}
		<li>{{.}}</li>
	{{end}}
	</ul>
	<br>
	<br>
	<br>
	
	<b> upload next sound</b>
	<br><input type="file" name="sound" id="sound" />
	<button id="load">Load</button>
	
<script>
function send(to, data, done) {
	var xhr = new XMLHttpRequest();
	xhr.open('POST', to, true);
	xhr.responseType = 'text';
	xhr.onload = function(ev) {
		if (this.status == 200 && typeof done == "function") {
			done(this.response);
		}
	};
	if(data) {
		xhr.send(data);
	} else {
		xhr.send();
	}
}
$("#start").on("click", function(ev) {
	send("/api/start");
});
$("#stop").on("click", function(ev) {
	send("/api/stop");
});
$("#load").on("click", function(ev) {
	var data = new FormData();
	var sound = $("#sound")[0];
	if(sound.files.length === 0) {
		alert("no file selected");
		return;
	}
	data.set("sound", sound.files[0]);
	send("/api/load", data, function() {
		location.href = "/";
	});
});
$("#list").on("click", "li", function(ev) {
	var data = new FormData();
	data.set("name", $(this).text());
	send("/api/choose", data);
})
</script>
</body>
</html>
`
