/*
   Copyright 2018-2019 Banco Bilbao Vizcaya Argentaria, S.A.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package rocksdb

// #include <stdlib.h>
// #include "rocksdb/c.h"
// #include "extended.h"
import "C"
import (
	"errors"
	"unsafe"
)

// BackupEngineInfo represents the information about the backups
// in a backup engine instance. Use this to get the state of the
// backup like number of backups and their ids and timestamps etc.
type BackupEngineInfo struct {
	c *C.rocksdb_backup_engine_info_t
}

// GetCount gets the number of backups available.
func (b *BackupEngineInfo) GetCount() int {
	return int(C.rocksdb_backup_engine_info_count(b.c))
}

// GetTimestamp gets the timestamp at which the backup index was taken.
func (b *BackupEngineInfo) GetTimestamp(index int) int64 {
	return int64(C.rocksdb_backup_engine_info_timestamp(b.c, C.int(index)))
}

// GetBackupID gets an id that uniquely identifies a backup
// regardless of its position.
func (b *BackupEngineInfo) GetBackupID(index int) int64 {
	return int64(C.rocksdb_backup_engine_info_backup_id(b.c, C.int(index)))
}

// GetSize gets the size of the backup in bytes.
func (b *BackupEngineInfo) GetSize(index int) int64 {
	return int64(C.rocksdb_backup_engine_info_size(b.c, C.int(index)))
}

// GetNumFiles gets the number of files in the backup index.
func (b *BackupEngineInfo) GetNumFiles(index int) int32 {
	return int32(C.rocksdb_backup_engine_info_number_files(b.c, C.int(index)))
}

// GetAppMetadata gets the backup associated metadata.
func (b *BackupEngineInfo) GetAppMetadata(index int) string {
	metadata := C.rocksdb_backup_engine_info_metadata(b.c, C.int(index))
	return C.GoString(metadata)
}

// Destroy destroys the backup engine info instance.
func (b *BackupEngineInfo) Destroy() {
	C.rocksdb_backup_engine_info_destroy(b.c)
	b.c = nil
}

// RestoreOptions captures the options to be used during
// restoration of a backup.
type RestoreOptions struct {
	c *C.rocksdb_restore_options_t
}

// NewRestoreOptions creates a RestoreOptions instance.
func NewRestoreOptions() *RestoreOptions {
	return &RestoreOptions{
		c: C.rocksdb_restore_options_create(),
	}
}

// SetKeepLogFiles is used to set or unset the keep_log_files option
// If true, restore won't overwrite the existing log files in wal_dir. It will
// also move all log files from archive directory to wal_dir.
// By default, this is false.
func (ro *RestoreOptions) SetKeepLogFiles(v int) {
	C.rocksdb_restore_options_set_keep_log_files(ro.c, C.int(v))
}

// Destroy destroys this RestoreOptions instance.
func (ro *RestoreOptions) Destroy() {
	C.rocksdb_restore_options_destroy(ro.c)
}

// BackupEngine is a reusable handle to a RocksDB Backup, created by
// OpenBackupEngine.
type BackupEngine struct {
	c    *C.rocksdb_backup_engine_t
	path string
	opts *Options
}

// OpenBackupEngine opens a backup engine with specified options.
func OpenBackupEngine(opts *Options, path string) (*BackupEngine, error) {
	var cErr *C.char
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))

	be := C.rocksdb_backup_engine_open(opts.c, cpath, &cErr)
	if cErr != nil {
		defer C.free(unsafe.Pointer(cErr))
		return nil, errors.New(C.GoString(cErr))
	}
	return &BackupEngine{
		c:    be,
		path: path,
		opts: opts,
	}, nil
}

// UnsafeGetBackupEngine returns the underlying c backup engine.
func (b *BackupEngine) UnsafeGetBackupEngine() unsafe.Pointer {
	return unsafe.Pointer(b.c)
}

// CreateNewBackup takes a new backup from db.
func (b *BackupEngine) CreateNewBackup(db *DB) error {
	var cErr *C.char

	C.rocksdb_backup_engine_create_new_backup(b.c, db.c, &cErr)
	if cErr != nil {
		defer C.free(unsafe.Pointer(cErr))
		return errors.New(C.GoString(cErr))
	}

	return nil
}

// CreateNewBackupWithMetadata is same as CreateNewBackup,
// but stores extra application metadata.
// Flush will always trigger if 2PC is enabled.
// If write-ahead logs are disabled, set flush_before_backup=true to
// avoid losing unflushed key/value pairs from the memtable.
func (b *BackupEngine) CreateNewBackupWithMetadata(db *DB, metadata string) error {
	var cErr *C.char
	cMetadata := C.CString(metadata)
	defer C.free(unsafe.Pointer(cMetadata))

	C.rocksdb_backup_engine_create_new_backup_with_metadata(b.c, db.c, cMetadata, &cErr)
	if cErr != nil {
		defer C.free(unsafe.Pointer(cErr))
		return errors.New(C.GoString(cErr))
	}

	return nil
}

// DeleteBackup deletes a single backup from a backup engine. 
func (b *BackupEngine) DeleteBackup(backupID uint32) error {
	var cErr *C.char

	C.rocksdb_backup_engine_delete_backup(b.c, C.uint(backupID), &cErr)
	if cErr != nil {
		defer C.free(unsafe.Pointer(cErr))
		return errors.New(C.GoString(cErr))
	}
	return nil
}

// GetInfo gets an object that gives information about
// the backups that have already been taken
func (b *BackupEngine) GetInfo() *BackupEngineInfo {
	return &BackupEngineInfo{
		c: C.rocksdb_backup_engine_get_backup_info(b.c),
	}
}

// RestoreDBFromLatestBackup restores the latest backup to dbDir. walDir
// is where the write ahead logs are restored to and usually the same as dbDir.
func (b *BackupEngine) RestoreDBFromLatestBackup(dbDir, walDir string, ro *RestoreOptions) error {
	var cErr *C.char
	cDbDir := C.CString(dbDir)
	cWalDir := C.CString(walDir)
	defer func() {
		C.free(unsafe.Pointer(cDbDir))
		C.free(unsafe.Pointer(cWalDir))
	}()

	C.rocksdb_backup_engine_restore_db_from_latest_backup(b.c, cDbDir, cWalDir, ro.c, &cErr)
	if cErr != nil {
		defer C.free(unsafe.Pointer(cErr))
		return errors.New(C.GoString(cErr))
	}
	return nil
}

// RestoreDBFromBackup restores a singular backup to dbDir. walDir
// is where the write ahead logs are restored to and usually the same as dbDir.
func (b *BackupEngine) RestoreDBFromBackup(backupID uint32, dbDir, walDir string, ro *RestoreOptions) error {
	var cErr *C.char
	cDbDir := C.CString(dbDir)
	cWalDir := C.CString(walDir)
	defer func() {
		C.free(unsafe.Pointer(cDbDir))
		C.free(unsafe.Pointer(cWalDir))
	}()

	C.rocksdb_backup_engine_restore_db_from_backup(b.c, C.uint(backupID), cDbDir, cWalDir, ro.c, &cErr)
	if cErr != nil {
		defer C.free(unsafe.Pointer(cErr))
		return errors.New(C.GoString(cErr))
	}
	return nil
}

// VerifyBackup checks that each file exists and that the size of the file matches our
// expectations. It does not check file checksum.
// Returns Status::OK() if all checks are good
func (b *BackupEngine) VerifyBackup(index uint32) error{
	var cErr *C.char
	C.rocksdb_backup_engine_verify_backup(b.c, C.uint(index), &cErr)
	if cErr != nil{
		defer C.free(unsafe.Pointer(cErr))
		return errors.New(C.GoString(cErr))
	}
	return nil
}

// Close closes the backup engine and cleans up state
// The backups already taken remain on storage.
func (b *BackupEngine) Close() {
	C.rocksdb_backup_engine_close(b.c)
	b.c = nil
}
