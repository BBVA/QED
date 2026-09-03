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

// #include "extended.h"
// #include <stdlib.h>
// #include "rocksdb/c.h"
import "C"
import "unsafe"

// WriteBatch holds a collection of updates to apply atomically to a DB.
//
// The updates are applied in the order in which they are added
// to the WriteBatch.  For example, the value of "key" will be "v3"
// after the following batch is written:
//
//    batch.Put("key", "v1");
//    batch.Delete("key");
//    batch.Put("key", "v2");
//    batch.Put("key", "v3");
//
type WriteBatch struct {
	c *C.rocksdb_writebatch_t
}

// NewWriteBatch creates a WriteBatch object.
func NewWriteBatch() *WriteBatch {
	return NewNativeWriteBatch(C.rocksdb_writebatch_create())
}

// NewNativeWriteBatch creates a WriteBatch object.
func NewNativeWriteBatch(c *C.rocksdb_writebatch_t) *WriteBatch {
	return &WriteBatch{c: c}
}

// WriteBatchFrom creates a write batch from a serialized WriteBatch.
func WriteBatchFrom(data []byte) *WriteBatch {
	return NewNativeWriteBatch(C.rocksdb_writebatch_create_from(bytesToChar(data), C.size_t(len(data))))
}

// Put stores the mapping "key->value" in the database.
func (wb *WriteBatch) Put(key, value []byte) {
	cKey := bytesToChar(key)
	cValue := bytesToChar(value)
	C.rocksdb_writebatch_put(wb.c, cKey, C.size_t(len(key)), cValue, C.size_t(len(value)))
}

// PutCF stores a mapping "key->value" in a column family.
func (wb *WriteBatch) PutCF(cf *ColumnFamilyHandle, key, value []byte) {
	cKey := bytesToChar(key)
	cValue := bytesToChar(value)
	C.rocksdb_writebatch_put_cf(wb.c, cf.c, cKey, C.size_t(len(key)), cValue, C.size_t(len(value)))
}

// Delete erases the mapping for "key" if it exists. Else, do nothing.
func (wb *WriteBatch) Delete(key []byte) {
	cKey := bytesToChar(key)
	C.rocksdb_writebatch_delete(wb.c, cKey, C.size_t(len(key)))
}

// DeleteCF erases the mapping for "key", in a column family, if it exists.
// Else, do nothing.
func (wb *WriteBatch) DeleteCF(cf *ColumnFamilyHandle, key []byte) {
	cKey := bytesToChar(key)
	C.rocksdb_writebatch_delete_cf(wb.c, cf.c, cKey, C.size_t(len(key)))
}

// DeleteRange erases all mappings in the range ["beginKey", "endKey")
// if the database contains them. Else do nothing.
func (wb *WriteBatch) DeleteRange(beginKey, endKey []byte) {
	cBeginKey := bytesToChar(beginKey)
	cEndKey := bytesToChar(endKey)
	C.rocksdb_writebatch_delete_range(wb.c, cBeginKey, C.size_t(len(beginKey)), cEndKey, C.size_t(len(endKey)))
}

// DeleteRangeCF erases all mappings in the range ["beginKey", "endKey")
// on the given column family if the database contains them. Else do nothing.
func (wb *WriteBatch) DeleteRangeCF(cf *ColumnFamilyHandle, beginKey, endKey []byte) {
	cBeginKey := bytesToChar(beginKey)
	cEndKey := bytesToChar(endKey)
	C.rocksdb_writebatch_delete_range_cf(wb.c, cf.c, cBeginKey, C.size_t(len(beginKey)), cEndKey, C.size_t(len(endKey)))
}

// Merge "value" with the existing value of "key" in the database.
// "key->merge(existing, value)"
func (wb *WriteBatch) Merge(key, value []byte) {
	cKey := bytesToChar(key)
	cValue := bytesToChar(value)
	C.rocksdb_writebatch_merge(wb.c, cKey, C.size_t(len(key)), cValue, C.size_t(len(value)))
}

// MergeCF "value" with the existing value of "key" in a column family.
// "key->merge(existing, value)"
func (wb *WriteBatch) MergeCF(cf *ColumnFamilyHandle, key, value []byte) {
	cKey := bytesToChar(key)
	cValue := bytesToChar(value)
	C.rocksdb_writebatch_merge_cf(wb.c, cf.c, cKey, C.size_t(len(key)), cValue, C.size_t(len(value)))
}

// Clear all updates buffered in this batch.
func (wb *WriteBatch) Clear() {
	C.rocksdb_writebatch_clear(wb.c)
}

// Count returns the number of updates in the batch.
func (wb *WriteBatch) Count() int {
	return int(C.rocksdb_writebatch_count(wb.c))
}

// Data returns the serialized version of this batch.
func (wb *WriteBatch) Data() []byte {
	var cSize C.size_t
	cValue := C.rocksdb_writebatch_data(wb.c, &cSize)
	return charToBytes(cValue, cSize)
}

// PutLogData appends a blob of arbitrary size to the records in this batch.
// The blob will be stored in the transaction log but not in any other files.
// In particular, it will not be persisted to the SST files. When iterating
// over this WriteBatch, WriteBatch::Handler::LogData will be called with the contents
// of the blob as it is encountered. Blobs, puts, deletes, and merges will be
// encountered in the same order in which they were inserted. The blob will
// NOT consume sequence number(s) and will NOT increase the count of the batch
//
// Example application: add timestamps to the transaction log for use in
// replication.
func (wb *WriteBatch) PutLogData(blob []byte, size int) {
	C.rocksdb_writebatch_put_log_data(wb.c, bytesToChar(blob), C.size_t(size))
}

// GetLogData retrieves the blob appended to this batch.
func (wb *WriteBatch) GetLogData(extractor *LogDataExtractor) []byte {
	if extractor != nil {
		C.rocksdb_writebatch_iterate_ext(wb.c, extractor.c)
		return extractor.Blob
	}
	return nil
}

// Destroy deallocates the WriteBatch object.
func (wb *WriteBatch) Destroy() {
	C.rocksdb_writebatch_destroy(wb.c)
	wb.c = nil
}

// WriteBatchHandler is used to iterate over the contents of a batch.
type WriteBatchHandler interface {
	ID() string
	LogData(blob []byte)
}

var writeBatchHandlers = NewRegistry()

//export rocksdb_writebatch_handler_log_data
func rocksdb_writebatch_handler_log_data(cIdx *C.char, cBlob *C.char, cBlobSize C.size_t) {
	blob := charToBytes(cBlob, cBlobSize)
	idx := C.GoString(cIdx)
	writeBatchHandlers.Lookup(idx).(WriteBatchHandler).LogData(blob)
}

// LogDataExtractor extracts metadata from a WriteBatch.
type LogDataExtractor struct {
	id    string
	ptrId *C.char
	Blob  []byte
	c     *C.rocksdb_writebatch_handler_t
}

// NewLogDataExtractor creates a new LogDataExtractor.
func NewLogDataExtractor(id string) *LogDataExtractor {
	extractor := &LogDataExtractor{
		id:    id,
		ptrId: C.CString(id),
		Blob:  make([]byte, 0),
	}
	extractor.c = C.rocksdb_writebatch_handler_create_ext(extractor.ptrId)
	writeBatchHandlers.Register(extractor.id, extractor)
	return extractor
}

// ID returns the unique identifier of this WriteBatchHandler.
func (e *LogDataExtractor) ID() string {
	return e.id
}

// LogData is a callback for the LogData.
func (e *LogDataExtractor) LogData(blob []byte) {
	e.Blob = append(blob[:0:0], blob...)
}

// Destroy destroys the LogDataExtractor.
func (e *LogDataExtractor) Destroy() {
	writeBatchHandlers.Unregister(e.ID())
	C.free(unsafe.Pointer(e.ptrId))
	C.rocksdb_writebatch_handler_destroy(e.c)
	e.Blob = nil
}
