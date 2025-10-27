package tasks

import (
	"bufio"
	"container/heap"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ---------------------------
// SortFile: external sort (merge sort on disk) for large files.
// Input: file with one integer per line.
// algo: "merge" (external merge) or "quick" (in-memory quick sort).
// Returns: sorted filename and elapsed milliseconds (as int64) or error.
// ---------------------------
type chunkInfo struct {
	path string
	size int64
}

func SortFile(name, algo string) (sortedFile string, elapsedMs int64, err error) {
	start := time.Now()
	// open input
	f, err := os.Open(name)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()

	// temp dir
	tmpDir, err := os.MkdirTemp("", "sortchunks")
	if err != nil {
		return "", 0, err
	}
	defer os.RemoveAll(tmpDir)

	reader := bufio.NewScanner(f)
	// increase buffer for long lines
	const maxBuf = 10 * 1024 * 1024
	buf := make([]byte, 0, 64*1024)
	reader.Buffer(buf, maxBuf)

	chunkPaths := []string{}
	chunkSizeLimit := int64(20 * 1024 * 1024) // 20MB per chunk
	if strings.ToLower(algo) == "quick" {
		// read whole file in memory (may OOM)
		var nums []int
		for reader.Scan() {
			line := strings.TrimSpace(reader.Text())
			if line == "" {
				continue
			}
			v, e := strconv.Atoi(line)
			if e != nil {
				return "", 0, fmt.Errorf("line parse error: %v", e)
			}
			nums = append(nums, v)
		}
		if err := reader.Err(); err != nil {
			return "", 0, err
		}
		sort.Ints(nums)
		out := name + ".sorted"
		of, err := os.Create(out)
		if err != nil {
			return "", 0, err
		}
		for _, v := range nums {
			fmt.Fprintln(of, v)
		}
		of.Close()
		return out, time.Since(start).Milliseconds(), nil
	}

	// external merge: chunk, sort, write
	var curChunk []int
	var curBytes int64 = 0
	for reader.Scan() {
		line := strings.TrimSpace(reader.Text())
		if line == "" {
			continue
		}
		v, e := strconv.Atoi(line)
		if e != nil {
			return "", 0, fmt.Errorf("line parse error: %v", e)
		}
		curChunk = append(curChunk, v)
		curBytes += int64(len(line) + 1)
		if curBytes >= chunkSizeLimit {
			sort.Ints(curChunk)
			chunkFile := filepath.Join(tmpDir, fmt.Sprintf("chunk-%d.tmp", len(chunkPaths)))
			if err := writeIntSlice(chunkFile, curChunk); err != nil {
				return "", 0, err
			}
			chunkPaths = append(chunkPaths, chunkFile)
			curChunk = nil
			curBytes = 0
		}
	}
	if len(curChunk) > 0 {
		sort.Ints(curChunk)
		chunkFile := filepath.Join(tmpDir, fmt.Sprintf("chunk-%d.tmp", len(chunkPaths)))
		if err := writeIntSlice(chunkFile, curChunk); err != nil {
			return "", 0, err
		}
		chunkPaths = append(chunkPaths, chunkFile)
	}

	if err := reader.Err(); err != nil {
		return "", 0, err
	}

	// if only one chunk, rename
	if len(chunkPaths) == 1 {
		out := name + ".sorted"
		if err := os.Rename(chunkPaths[0], out); err != nil {
			return "", 0, err
		}
		return out, time.Since(start).Milliseconds(), nil
	}

	// k-way merge
	outName := name + ".sorted"
	outFile, err := os.Create(outName)
	if err != nil {
		return "", 0, err
	}
	defer outFile.Close()

	if err := kWayMerge(chunkPaths, outFile); err != nil {
		return "", 0, err
	}

	elapsed := time.Since(start).Milliseconds()
	return outName, elapsed, nil
}

func writeIntSlice(path string, data []int) error {
	of, err := os.Create(path)
	if err != nil {
		return err
	}
	defer of.Close()
	w := bufio.NewWriter(of)
	for _, v := range data {
		fmt.Fprintln(w, v)
	}
	return w.Flush()
}

// k-way merge using min-heap
type fileScanner struct {
	val int
	sc  *bufio.Scanner
	f   *os.File
}

type minHeap []*fileScanner

func (h minHeap) Len() int            { return len(h) }
func (h minHeap) Less(i, j int) bool  { return h[i].val < h[j].val }
func (h minHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *minHeap) Push(x interface{}) { *h = append(*h, x.(*fileScanner)) }
func (h *minHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

func kWayMerge(chunkPaths []string, out io.Writer) error {
	h := &minHeap{}
	heap.Init(h)
	// open all chunk files
	for _, p := range chunkPaths {
		f, err := os.Open(p)
		if err != nil {
			return err
		}
		sc := bufio.NewScanner(f)
		// allow large buffer
		buf := make([]byte, 0, 64*1024)
		sc.Buffer(buf, 10*1024*1024)
		if sc.Scan() {
			v, err := strconv.Atoi(strings.TrimSpace(sc.Text()))
			if err != nil {
				f.Close()
				return err
			}
			fs := &fileScanner{val: v, sc: sc, f: f}
			heap.Push(h, fs)
		} else {
			f.Close()
		}
	}
	w := bufio.NewWriter(out)
	for h.Len() > 0 {
		fs := heap.Pop(h).(*fileScanner)
		fmt.Fprintln(w, fs.val)
		if fs.sc.Scan() {
			v, err := strconv.Atoi(strings.TrimSpace(fs.sc.Text()))
			if err != nil {
				return err
			}
			fs.val = v
			heap.Push(h, fs)
		} else {
			fs.f.Close()
		}
	}
	return w.Flush()
}

// ---------------------------
// WordCount: lines, words, bytes (streaming)
// ---------------------------
func WordCount(name string) (lines, words, bytesCount int64, err error) {
	f, err := os.Open(name)
	if err != nil {
		return 0, 0, 0, err
	}
	defer f.Close()

	r := bufio.NewReader(f)
	var inWord bool
	for {
		buf := make([]byte, 32*1024)
		n, e := r.Read(buf)
		if n > 0 {
			bytesCount += int64(n)
			for i := 0; i < n; i++ {
				c := buf[i]
				if c == '\n' {
					lines++
				}
				if (c == ' ' || c == '\n' || c == '\t' || c == '\r') && inWord {
					words++
					inWord = false
				} else if !(c == ' ' || c == '\n' || c == '\t' || c == '\r') {
					inWord = true
				}
			}
		}
		if e == io.EOF {
			if inWord {
				words++
			}
			break
		}
		if e != nil {
			return lines, words, bytesCount, e
		}
	}
	return lines, words, bytesCount, nil
}

// ---------------------------
// Grep: regex search, return count and first up to 10 matched lines
// ---------------------------
func Grep(name, pattern string) (count int64, matched []string, err error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return 0, nil, err
	}
	f, err := os.Open(name)
	if err != nil {
		return 0, nil, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if re.FindStringIndex(line) != nil {
			count++
			if len(matched) < 10 {
				matched = append(matched, line)
			}
		}
	}
	if sc.Err() != nil {
		return count, matched, sc.Err()
	}
	return count, matched, nil
}

// ---------------------------
// Compress: gzip (std) or xz (shell 'xz' required)
// Returns output filename and size in bytes.
// ---------------------------
func Compress(name, codec string) (outName string, outSize int64, err error) {
	switch strings.ToLower(codec) {
	case "gzip", "gz":
		in, err := os.Open(name)
		if err != nil {
			return "", 0, err
		}
		defer in.Close()
		outName = name + ".gz"
		out, err := os.Create(outName)
		if err != nil {
			return "", 0, err
		}
		defer out.Close()
		gw := gzip.NewWriter(out)
		defer gw.Close()
		if _, err := io.Copy(gw, in); err != nil {
			return "", 0, err
		}
		if fi, err := os.Stat(outName); err == nil {
			return outName, fi.Size(), nil
		}
		return outName, 0, nil
	case "xz":
		// require 'xz' command available on system
		outName = name + ".xz"
		cmd := exec.Command("xz", "-k", "-c", name) // -k keep input
		outfile, err := os.Create(outName)
		if err != nil {
			return "", 0, err
		}
		defer outfile.Close()
		cmd.Stdout = outfile
		if err := cmd.Run(); err != nil {
			return "", 0, fmt.Errorf("error running xz: %v", err)
		}
		if fi, err := os.Stat(outName); err == nil {
			return outName, fi.Size(), nil
		}
		return outName, 0, nil
	default:
		return "", 0, errors.New("codec no soportado (gzip|xz)")
	}
}

// ---------------------------
// HashFile: sha256 streaming
// ---------------------------
func HashFile(name string) (string, error) {
	
	f, err := os.Open(name)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
