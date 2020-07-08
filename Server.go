package main

import (
    "bufio"
    "io"
    "os"
	"strconv"
	"log"
    "net/http"
	"fmt"
	"strings"
	"math/rand"
	"time"
	"sync"
	"io/ioutil"
)

var wins, indexs []int
var frees, winFrees []bool
var seek int
var freeRemain int
var m sync.Mutex

func main() {
	stage := []int{0, 1, 2, 3, 5, 10, 20 ,25 ,50 ,100 ,150 ,200 ,250 ,300 ,500 ,10000}
	wins = make([]int, 0)
	indexs = make([]int, 0)
	frees = make([]bool, 0)
	winFrees = make([]bool, 0)
	seek = -1
	freeRemain = 0
	file, _ := os.Open("10000000.csv")
	fin := bufio.NewReader(file)
	for {
        a, _, c := fin.ReadLine()
        if c == io.EOF {
            break
        }
		d := strings.Split(string(a),",")
		n, _ := strconv.Atoi(d[1])
		wins = append(wins, n)
		f := float32(n) / 15
		for i := 0; i < len(stage); i++ {
			if f <= float32(stage[i]) {
				indexs = append(indexs, i)
				break
			}
		}
		frees = append(frees, d[2] == "true")
		if len(frees) == 1 {
			continue
		}
		winFrees = append(winFrees, d[2] == "true" && freeRemain == 0)
		if d[2] == "true" {
			if freeRemain == 0 {
				freeRemain += 15
			}
			freeRemain--
		} else {
			freeRemain = 0
		}
    }
	winFrees = append(winFrees, false)
	file.Close()
	rand.Seed(time.Now().UnixNano())
	fmt.Println("Init OK.")
	
	http.HandleFunc("/sim", func(w http.ResponseWriter, r *http.Request) {
		m.Lock()
		w.Header().Set("Content-Type","application/json; charset=utf-8")
		w.Header().Add("Access-Control-Allow-Origin","*")
		w.Header().Add("Access-Control-Allow-Headers","*")
		buf := make([]byte, 16384)
		l, _ := r.Body.Read(buf)
		buf = buf[:l]
		body := strings.Split(string(buf),";")
		if len(body) == 1 {
			m.Unlock()
			fmt.Println("Failed.")
			fmt.Println()
			fmt.Fprintf(w,"Failed.")
			return
		}
		times, _ := strconv.Atoi(body[0])
		baseString := strings.Split(body[1],",")
		freeString := strings.Split(body[2],",")
		baseConfig := make([]int, 0)
		freeConfig := make([]int, 0)
		for i := 0; i < len(baseString); i++ {
			n, _ := strconv.Atoi(baseString[i])
			baseConfig = append(baseConfig, n)
		}
		for i := 0; i < len(freeString); i++ {
			n, _ := strconv.Atoi(freeString[i])
			freeConfig = append(freeConfig, n)
		}
		fmt.Println("Seek Position: ", seek)
		fmt.Println("Times: ", times)
		fmt.Println("BaseConfig: ", baseConfig)
		fmt.Println("FreeConfig: ", freeConfig)
		
		start := time.Now()
		basecount := []int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
		freecount := []int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
		var RTP, baseRTP, freeRTP float32
		basehit := []float32{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
		freehit := []float32{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
		freegame := 0
		sum := 0
		sumbase := 0
		sumfree := 0
		freeBets := 0
		freeRemain = 0
		for i := 0; i < times; i++ {
			win, free, freeAdd:= next(baseConfig, freeConfig)
			sum += win
			if free {
				i--
				freegame++
				sumfree += win
				freecount[indexs[seek]]++
			} else {
				sumbase += win
				basecount[indexs[seek]]++
			}
			if freeAdd {
				freeBets++
			}
		}
		for i := 0; i < len(stage) - 1; i++ {
			basehit[i] = float32(basecount[i]) / float32(times)
			if freegame > 0 {
				freehit[i] = float32(freecount[i]) / float32(freegame)
			} else {
				freehit[i] = 0
			}
		}
		RTP = float32(sum) / (float32(times) * 15)
		baseRTP = float32(sumbase) / (float32(times) * 15)
		if freeBets > 0 {
			freeRTP = float32(sumfree) / (float32(freeBets) * 15)
		} else {
			freeRTP = 0
		}
		fmt.Println("RTP: ", RTP)
		fmt.Println("BasegameRTP: ", baseRTP)
		fmt.Println("FreegameRTP: ", freeRTP)
		fmt.Println("BasegameHitRate: ", basehit)
		fmt.Println("FreegameHitRate: ", freehit)
		fmt.Println("Cost time: " ,time.Since(start))
		fmt.Println()
		fmt.Fprintf(w,"{\"RTP\":%f,\"BasegameRTP\":%f,\"FreegameRTP\":%f,",RTP,baseRTP,freeRTP)
		fmt.Fprintf(w,"\"BasegameHitRate\":[%f",basehit[0])
		for i := 1; i < len(basehit); i++ {
			fmt.Fprintf(w,",%f",basehit[i])
		}
		fmt.Fprintf(w,"],\"FreegameHitRate\":[%f",freehit[0])
		for i := 1; i < len(freehit); i++ {
			fmt.Fprintf(w,",%f",freehit[i])
		}
		fmt.Fprintf(w,"]}")
		m.Unlock()
    })
	http.HandleFunc("/web/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Access-Control-Allow-Origin","*")
		w.Header().Add("Access-Control-Allow-Headers","*")
		t := strings.Split(r.URL.Path, ".")
		if len(t) > 1 {
			if t[1] == "css" {
				w.Header().Set("Content-Type","text/css; charset=utf-8")
			}
		}
		index, _ := ioutil.ReadFile("web/index.html")
		file, err := ioutil.ReadFile(r.URL.Path[1:])
		if err != nil {
			fmt.Println("GET: web/index.html")
			fmt.Fprintf(w, "%s", index)
		} else {
			fmt.Println("GET:", r.URL.Path[1:])
			fmt.Fprintf(w, "%s", file)
		}
	})
	http.HandleFunc("/save", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Access-Control-Allow-Origin","*")
		w.Header().Add("Access-Control-Allow-Headers","*")
		if r.Method != "POST" {
			fmt.Fprintf(w,"Not POST.")
			return
		}
		fmt.Println("SAVE");
		buf := make([]byte, 1048576)
		l, _ := r.Body.Read(buf)
		buf = buf[:l]
		now := time.Now()
		fout, err := os.Create("DNA/" + now.Format("2006-01-02-15-04-05") + ".txt")
		if err != nil {
			panic(err)
			fmt.Println()
			fmt.Fprintf(w,"Save Failde.")
		} else {
			fout.Write(buf)
			fout.Close()
			fmt.Println("Save Filed: " + now.Format("2006-01-02-15-04-05") + ".txt")
			fmt.Println()
			fmt.Fprintf(w,"Save Filed.")
		}
    })
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func next(baseConfig, freeConfig []int) (int, bool, bool){
	freeAdd := false
	seek = (seek + 1) % len(wins)
	for {
		if frees[seek] {
			if freeRemain > 0 {
				if indexs[seek] < len(freeConfig) - 1 {
					if winFrees[seek] {
						if rand.Intn(1000) < freeConfig[len(freeConfig) - 1] {
							freeRemain = (freeRemain + 15) % 150
							freeAdd = true
							break
						}
					} else if rand.Intn(1000) < freeConfig[indexs[seek]] {
						break
					}
				}
			}
		} else {
			if freeRemain <= 0 {
				if indexs[seek] < len(baseConfig) - 1 {
					if winFrees[seek] { 
						if rand.Intn(1000) < baseConfig[len(baseConfig) - 1] {
							freeRemain += 15
							freeAdd = true
							break
						}
					} else if rand.Intn(1000) < baseConfig[indexs[seek]] {
						break
					}
				}
			}
		}
		seek = (seek + 1) % len(wins)
	}
	if freeRemain > 0 {
		freeRemain--
	} else {
		freeRemain = 0
	}
	return wins[seek], frees[seek], freeAdd
}