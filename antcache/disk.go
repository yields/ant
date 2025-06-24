package antcache

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/golang/snappy"
)

// File represents an in-memory file.
type file struct {
	key   uint64
	path  string
	mtime time.Time
	size  int64
}

// DiskOption represents a disk option.
type DiskOption func(*Diskstore) error

// Maxage sets the maxage to age.
//
// When <= 0, the disk will not track file age
// and will not remove files that have expired.
//
// Defaults to 24 hours.
func Maxage(age time.Duration) DiskOption {
	return func(ds *Diskstore) error {
		ds.maxage = age
		return nil
	}
}

// Maxsize sets the maxsize to size.
//
// When <= 0, the disk will not track the disk
// the file sizes and remove files to ensure the
// disk usage stays constant.
//
// Defaults to 1gb.
func Maxsize(size int64) DiskOption {
	return func(ds *Diskstore) error {
		ds.maxsize = size
		return nil
	}
}

// SweepEvery sweeps the files every d.
//
// By default the disk will sweep all files every 5 minutes,
// when d <= 0, no background file sweeper is done and so the
// disk size may keep growing.
//
// The disk will remove all files that have exceeded the maxage
// when maxsize is set, the sweeper will remove files until
// the maxsize is reached.
func SweepEvery(d time.Duration) DiskOption {
	return func(ds *Diskstore) error {
		if d > 0 {
			ds.ticker = time.NewTicker(d)
		} else {
			ds.ticker.Stop()
			ds.ticker = nil
		}
		return nil
	}
}

// Compress makes the diskstore compress and uncompress all
// cached items.
//
// Note that the diskstore will not check the cached item
// before attempting to de-compress therefore the items
// are not interchangeable between to disks where one
// has no compression and the other one has compression.
//
// By default compression is turned off.
func Compress() DiskOption {
	return func(ds *Diskstore) error {
		ds.compress = true
		return nil
	}
}

// DebugFunc represents a debug func.
//
// By default the diskstore outputs no debug logs.
type DebugFunc func(format string, args ...any)

// Debug sets the debug logging func.
//
// When set on the diskstore debug will be enabled
// and debug logs are written to it.
//
// By default the func is set to nil which disables
// debug logging.
//
// Example:
//
//	Open("root", Debug(log.Printf))
//
// Debug logs are automatically prefixed with `"antcache/disk: "`.
func Debug(f DebugFunc) DiskOption {
	return func(ds *Diskstore) error {
		ds.debug = f
		return nil
	}
}

// Diskstore implements disk cache storage.
//
// The storage is expected to be configured with
// an existing directory `path` where it will write
// all cached responses.
//
// The storage ensures that store calls are not visible
// to load calls until the file is written to disk successfully
// and fsynced, it does this by writing a temporary file, fsyncing it
// and then renaming it to the expected filename.
//
// When the disk is configured with invalid directory name
// all its method return the same error.
type Diskstore struct {
	path     string
	dir      *os.File
	maxage   time.Duration
	maxsize  int64
	stop     chan struct{}
	warm     chan struct{}
	wg       sync.WaitGroup
	ticker   *time.Ticker
	readymu  sync.RWMutex
	ready    map[uint64]file
	now      func() time.Time
	debug    DebugFunc
	compress bool
}

// Open opens a new disk storage.
//
// It is up to the caller to ensure that the given path will
// not be changed by different processes, the diskstore doesn't
// implement any filesystem level locking.
func Open(path string, opts ...DiskOption) (*Diskstore, error) {
	disk := &Diskstore{
		path:     path,
		maxage:   24 * time.Hour,
		maxsize:  1 << 30,
		stop:     make(chan struct{}),
		wg:       sync.WaitGroup{},
		warm:     make(chan struct{}),
		readymu:  sync.RWMutex{},
		ready:    make(map[uint64]file),
		ticker:   time.NewTicker(5 * time.Minute),
		now:      time.Now,
		debug:    nil,
		compress: false,
	}

	for _, opt := range opts {
		if err := opt(disk); err != nil {
			return nil, err
		}
	}

	if err := disk.init(); err != nil {
		return nil, err
	}

	disk.wg.Add(1)
	go disk.warmup()

	if disk.ticker != nil {
		disk.wg.Add(1)
		go disk.sweeper()
	}

	return disk, nil
}

// Debugf writes debug logs if `ds.debug` is non nil.
func (d *Diskstore) debugf(format string, args ...any) {
	if d.debug != nil {
		d.debug("antcache/disk: "+format, args...)
	}
}

// Init initializes the disk store.
func (d *Diskstore) init() error {
	if !filepath.IsAbs(d.path) {
		return fmt.Errorf("antcache: disk expects an absolute path, got %q", d.path)
	}

	f, err := os.Open(d.path)
	if err != nil {
		return fmt.Errorf("antcache: disk %w", err)
	}

	stat, err := f.Stat()
	if err != nil {
		f.Close()
		return fmt.Errorf("antcache: disk stat %q - %w", d.path, err)
	}

	if !stat.IsDir() {
		f.Close()
		return fmt.Errorf("antcache: disk expected a directory")
	}

	d.debugf("opened root %s", d.path)
	d.dir = f
	return nil
}

// Wait waits for the disk to read all files.
//
// When the disk is initialized it will spawn a goroutine
// to read all files in the configured path, if there are many
// files it will typically take a while.
//
// The method returns the context's error if canceled, otherwise
// it will block until the disk cache is warm.
func (d *Diskstore) Wait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-d.warm:
		return nil
	}
}

// Warmup ensures that all on disk files are referenced in-memory.
//
// The method loops over all files in the configured directory and
// attempts to add them into the .ready map to be used by `Load()`.
//
// The method logs if any errors occur.
func (d *Diskstore) warmup() {
	var files []file

	defer func() {
		close(d.warm)
		d.wg.Done()
	}()

	for {
		names, err := d.dir.Readdirnames(5)

		if errors.Is(err, io.EOF) {
			break
		}

		for _, name := range names {
			var path = filepath.Join(d.path, name)

			if filepath.Ext(path) == "tmp" {
				continue
			}

			n, err := strconv.ParseUint(name, 10, 64)
			if err != nil {
				log.Printf("antcache: disk cache invalid entry - %s", name)
				continue
			}

			stat, err := os.Stat(path)
			if err != nil {
				log.Printf("antcache: disk stat %s - %s", path, err)
				continue
			}

			files = append(files, file{
				key:   n,
				path:  path,
				size:  stat.Size(),
				mtime: stat.ModTime(),
			})
		}
	}

	d.readymu.Lock()
	for _, f := range files {
		if _, ok := d.ready[f.key]; !ok {
			d.ready[f.key] = f
		}
	}
	d.readymu.Unlock()

	d.debugf("found %d cached pages", len(files))
}

// Sweeper sweeps the directory.
//
// Every minute, the sweeper will wake and loop over all
// "ready" files, if any of the items maxage exceeds that of
// the configured maxage, the method will acquire a lock
// and delete the file.
//
// If the size of all items exceeds the configured maxsize
// the method will delete old files until the maxsize is reached.
func (d *Diskstore) sweeper() {
	defer func() {
		d.ticker.Stop()
		d.wg.Done()
	}()

	for {
		select {
		case <-d.stop:
			return

		case <-d.ticker.C:
			if _, err := d.sweep(); err != nil {
				log.Printf("antcache: disk sweep - %s", err)
			}
		}
	}
}

// Sweep sweeps the directory.
func (d *Diskstore) sweep() (int, error) {
	var files = d.files()
	var now = d.now()
	var removed int
	var remove []file
	var sum int64

	sort.Slice(files, func(i, j int) bool {
		a := files[i].mtime
		b := files[j].mtime
		return a.Before(b)
	})

	for _, f := range files {
		if d.maxage > 0 {
			if now.Sub(f.mtime) > d.maxage {
				remove = append(remove, f)
			}
		}

		if d.maxsize > 0 {
			if sum += f.size; sum > d.maxsize {
				remove = append(remove, f)
				sum -= f.size
			}
		}
	}

	d.readymu.Lock()
	defer d.readymu.Unlock()

	for _, f := range remove {
		if _, ok := d.ready[f.key]; ok {
			if err := os.Remove(f.path); err != nil {
				log.Printf("antcache: disk remove - %s", err)
				continue
			}
			delete(d.ready, f.key)
			removed++
		}
	}

	if removed > 0 {
		d.debugf("removed %d expired pages", removed)
	}

	return removed, nil
}

// Files returns the files.
func (d *Diskstore) files() []file {
	d.readymu.RLock()
	ret := make([]file, 0, len(d.ready))
	for _, f := range d.ready {
		ret = append(ret, f)
	}
	d.readymu.RUnlock()
	return ret
}

// Store implementation.
func (d *Diskstore) Store(ctx context.Context, key uint64, v []byte) error {
	f, err := os.CreateTemp(d.path, "*.tmp")
	if err != nil {
		return fmt.Errorf("antcache: open tempfile - %w", err)
	}

	cleanup := func() {
		f.Close()
		os.Remove(f.Name())
	}

	if d.compress {
		v = snappy.Encode(nil, v)
	}

	if _, err := f.Write(v); err != nil {
		cleanup()
		return fmt.Errorf("antcache: disk write - %w", err)
	}

	if err := f.Sync(); err != nil {
		cleanup()
		return fmt.Errorf("antcache: disk fsync - %w", err)
	}

	if err := d.add(key, f); err != nil {
		cleanup()
		return fmt.Errorf("antcache: add - %w", err)
	}

	d.debugf("store %d", key)
	return nil
}

// Load implementation.
func (d *Diskstore) Load(_ context.Context, key uint64) (v []byte, err error) {
	d.readymu.RLock()
	defer d.readymu.RUnlock()

	if f, ok := d.ready[key]; ok {
		if v, err = os.ReadFile(f.path); err != nil {
			return nil, fmt.Errorf("antcache: disk read %q - %w", f.path, err)
		}
		d.debugf("load %d %s", key)
	}

	if v != nil && d.compress {
		if v, err = snappy.Decode(nil, v); err != nil {
			err = fmt.Errorf(
				"antcache: compress is on but snappy can't decode %s/%d - %w",
				d.path,
				key,
				err,
			)
		}
	}

	return
}

// Add adds the given file to the keys cache.
func (d *Diskstore) add(key uint64, f *os.File) error {
	var newname = strconv.FormatUint(key, 10)
	var newpath = filepath.Join(d.path, newname)

	stat, err := f.Stat()
	if err != nil {
		return fmt.Errorf("antcache: disk stat - %w", err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("antcache: close %s - %w", f.Name(), err)
	}

	if err := os.Rename(f.Name(), newpath); err != nil {
		return fmt.Errorf("antcache: disk rename - %w", err)
	}

	if err := d.dir.Sync(); err != nil {
		return fmt.Errorf("antcache: disk fsync - %w", err)
	}

	d.readymu.Lock()
	d.ready[key] = file{
		key:   key,
		path:  newpath,
		size:  stat.Size(),
		mtime: stat.ModTime(),
	}
	d.readymu.Unlock()

	return nil
}

// Close closes the diskstore.
func (d *Diskstore) Close() error {
	close(d.stop)
	d.wg.Wait()

	if err := d.dir.Close(); err != nil {
		return fmt.Errorf("antcache: disk close dir - %w", err)
	}

	d.debugf("closed %s", d.path)
	return nil
}
