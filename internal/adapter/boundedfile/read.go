// Package boundedfile reads a whole file without letting its length decide how
// much memory the run costs.
//
// The manifest reader and the response reader both hold a file as one value
// while a comparison runs, and both bound what they will accept at the same
// forty eight megabytes. This package is the reading half they share. The
// refusal itself stays with them, because a manifest and a response record are
// different things to be handed one that is too large, and each says so in its
// own words.
//
// The recipe reader is deliberately not a caller. Its limit is 256 kB, so the
// doubling this package avoids costs half a megabyte there, and the change
// needed to route it through here was worth less than the plain call it has.
package boundedfile

import (
	"errors"
	"io"
	"io/fs"
	"os"
)

// ErrTooLarge reports a file past the limit it was read under.
//
// It is a sentinel rather than a message because the caller words the refusal.
// Match it with errors.Is.
var ErrTooLarge = errors.New("the file is longer than the limit it was read under")

// ErrChangedWhileRead reports a file that grew past its limit while being read.
//
// It is separate from ErrTooLarge because the two are different events to the
// person holding the file: one was already too big, the other was being written
// while this process read it, and telling them apart is the difference between
// changing the file and finding out who else has it open.
var ErrChangedWhileRead = errors.New("the file grew past the limit while it was being read")

// Read returns the whole contents of an open file, refusing one longer than
// limit.
//
// # Why the length is asked for before anything is allocated
//
// io.ReadAll grows its buffer by doubling, so it holds the old buffer and the
// new one at the moment the last doubling copies. A file just under the limit
// therefore costs about twice its own size. Measured on a 38.6 MB manifest,
// five runs each with the first discarded: 88.8 MB peak read that way against
// 46.6 MB when the length is asked for first. The limit is untouched by this.
// What changes is what staying under it costs.
//
// # Why an oversized file is refused before it is read
//
// Reading up to the limit in order to say no was the most expensive thing this
// code did: the same measurement put the refusal at 126.6 MB, more than any
// accepted file cost. Asking first turns that into a stat call.
//
// # Why the length is not trusted on its own
//
// Something else may be writing the file. The read is still bounded, and a file
// that grew past the limit in between is refused rather than truncated, because
// a truncated document comes back as a complaint about broken JSON and sends
// the reader looking for a fault in a file that is fine.
func Read(file *os.File, limit int64) ([]byte, error) {
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if regular(info) && info.Size() > limit {
		return nil, ErrTooLarge
	}

	// One byte past the limit, so that a file exactly at it is accepted and a
	// file over it is known to be over rather than guessed at.
	raw, err := readInto(make([]byte, 0, room(info, limit)), io.LimitReader(file, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) > limit {
		if regular(info) && info.Size() <= limit {
			return nil, ErrChangedWhileRead
		}
		return nil, ErrTooLarge
	}
	return raw, nil
}

// regular says whether the length this file reports means anything. A pipe, a
// device and a directory all report a length that has nothing to do with how
// much comes out of them.
func regular(info fs.FileInfo) bool {
	return info.Mode().IsRegular()
}

// room is how much to allocate up front: the file plus the one byte that tells
// a file at the limit from a file over it. A source whose length says nothing
// starts small and grows, which is what io.ReadAll would have done anyway.
func room(info fs.FileInfo, limit int64) int {
	if !regular(info) {
		return 0
	}
	size := info.Size()
	if size < 0 || size > limit {
		return 0
	}
	return int(size) + 1
}

// readInto fills the buffer it is given, growing it only if the source turns
// out to be longer than it claimed. It is io.ReadAll with the first allocation
// already made, which is the whole point of this package.
func readInto(buf []byte, from io.Reader) ([]byte, error) {
	for {
		if len(buf) == cap(buf) {
			buf = append(buf, 0)[:len(buf)]
		}
		read, err := from.Read(buf[len(buf):cap(buf)])
		buf = buf[:len(buf)+read]
		if err != nil {
			if errors.Is(err, io.EOF) {
				err = nil
			}
			return buf, err
		}
	}
}
