package dedupe

import (
	"crypto/md5"
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"os"

	"github.com/jucardi/go-osx/paths"
)

const (
	HashMD5    = HashMode("md5")
	HashSHA256 = HashMode("sha256")
)

type HashMode string

type IDedupe interface {
	FindDupes(path string) (*DupeReport, error)
	SetOptions(opts *Options)
}

type Options struct {
	Mode                  HashMode
	Recursive             bool
	PotentialDupeCallback func(paths []string, size int64)
	CurrentDirCallback    func(dir string)
	ReadingHashCallback   func(file string)
	HashReadCallback      func(file, hash string)
}

type DupeReport struct {
	Dupes  map[string][]string
	Errors []error
}

type service struct {
	precheckMap map[int64][]string
	errs        []error
	options     *Options
}

func New() IDedupe {
	return &service{}
}

func (s *service) init() {
	s.precheckMap = map[int64][]string{}
	s.errs = []error{}
	if s.options == nil {
		s.SetOptions(&Options{})
	}
}

func (s *service) SetOptions(opts *Options) {
	s.options = opts
}

func (s *service) FindDupes(path string) (*DupeReport, error) {
	s.init()
	fi, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("error reading path %s, %v", path, err)
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("the give path is not a directory, %s", path)
	}

	s.processDir(path)
	result := s.dedupe()

	ret := &DupeReport{
		Errors: s.errs,
		Dupes:  map[string][]string{},
	}

	for k, v := range result {
		if len(v) <= 1 {
			continue
		}
		ret.Dupes[k] = v
	}
	return ret, nil
}

func (s *service) processDir(path string) {
	if s.options.CurrentDirCallback != nil {
		s.options.CurrentDirCallback(path)
	}

	items, err := ioutil.ReadDir(path)

	if err != nil {
		s.errs = append(s.errs, fmt.Errorf("error reading contents of path %s, %s", path, err.Error()))
		return
	}

	var dirs []string

	for _, item := range items {
		if item.IsDir() {
			dirs = append(dirs, item.Name())
			continue
		}
		s.precheck(path, item)
	}

	if !s.options.Recursive {
		return
	}

	for _, dir := range dirs {
		s.processDir(paths.Combine(path, dir))
	}
}

func (s *service) precheck(path string, fInfo os.FileInfo) {
	filePath := paths.Combine(path, fInfo.Name())
	if _, ok := s.precheckMap[fInfo.Size()]; !ok {
		s.precheckMap[fInfo.Size()] = []string{filePath}
		return
	}
	s.precheckMap[fInfo.Size()] = append(s.precheckMap[fInfo.Size()], filePath)
	if s.options.PotentialDupeCallback != nil {
		s.options.PotentialDupeCallback(s.precheckMap[fInfo.Size()], fInfo.Size())
	}
}

func (s *service) dedupe() map[string][]string {
	m := map[string][]string{}
	for _, v := range s.precheckMap {
		if len(v) <= 1 {
			continue
		}

		for _, file := range v {
			checksum, err := s.getHash(file)
			if err != nil {
				s.errs = append(s.errs, err)
				continue
			}
			if _, ok := m[checksum]; !ok {
				m[checksum] = []string{file}
			} else {
				m[checksum] = append(m[checksum], file)
			}
		}
	}
	return m
}

func (s *service) getHasher() hash.Hash {
	switch s.options.Mode {
	case HashMD5:
		return md5.New()
	default:
		return sha256.New()
	}
}

func (s *service) getHash(file string) (string, error) {
	if s.options.ReadingHashCallback != nil {
		s.options.ReadingHashCallback(file)
	}

	f, err := os.Open(file)
	if err != nil {
		return "", fmt.Errorf("unable to read file %s, %s", file, err.Error())
	}

	defer f.Close()
	h := s.getHasher()

	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("unable to calculate checksum of file %s, %s", file, err.Error())
	}

	checksum := fmt.Sprintf("%x", h.Sum(nil))
	if s.options.HashReadCallback != nil {
		s.options.HashReadCallback(file, checksum)
	}

	return checksum, nil
}
