/* Copyright (C) 2013 CompleteDB LLC.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with PubSubSQL.  If not, see <http://www.gnu.org/licenses/>.
 */

package pubsubsql

import (
	"net"
)

type ActionType int

const (
	NoAction ActionType = iota
	Insert
	Select
	Update
	Delete
	PubSubInsert
	PubSubAdd
	PubSubUpdate
	PubSubDelete
)

type Client interface {

	// Connect connects to the pubsubsql server.
	// Address has the form host:port.
	Connect(address string) bool

	// Disconnect disconnects from the pubsubsql server.
	Disconnect() 

	// Ok determines if last operation succeeded. 
	Ok() bool

	// Failed determines if last operation failed.
	Failed() bool

	// ErrorString returns error string when last operation fails.
	ErrorString() string

	// Execute executes a command.
	// Returns true on success.
	Execute(command string) bool
	
	// JSON() returns JSON response string returned from the last operation.
	JSON() string
	
	// Id returns unique record identifier generated by the database table.
	// Valid for actions: insert, select (when id is in selected columns),
	// pubsub insert, pubsub add, pubsub delete, pubsub update.
	Id() string

	// PubSubId returns unique pubsub identifier generated by the database.  
	// Valid for actions: subscribe, pubsub insert, pubsub add, pubsub delete, pubsub update.
	// PubSubId is generated by pubsubsql server when client subscribes to a table and
	// used to uniqely identify particual subscription.
	PubSubId() string	

	// Action returns action for last operation.
	Action() ActionType				

	// RecordCount returns number of records returned in the result set batch.
	RecordCount() int

	// Next move cusrsor to the next data record of the result result set batch.    
	// Returns false when all records are read.
	// Must be called initially to position cursor to the first record. 
	Next() bool

	// JSONRecord returns current record in JSON format.
	JSONRecord() string
	
	// Value returns column value by column name.
	// If column does not exist in current result set it returns empty string.	
	Value(column string) string

	// ValueByOrdinal returns column value by column ordinal.
	// If column ordinal does not exist in current result set it returns empty string.	
	ValueByColumnOrdinal(ordinal int) string

	// Columns returns array of valid column names returned by last operation. 		
	Columns() []string

	// ColumnCount returns number of valid columns
	ColumnCount() int

	// WaitForPubSub waits until publish message is retreived or
	// timeout expired.
	// Returns false on timeout.
	WaitForPubSub(timeout int) bool
				
	ValidId() bool		
	ValidPubSubId() bool
	ValidColumn(column string) bool
	ValidColumnOrdinal(ordinal int) bool
	ValidRecord() bool
}

func NewClient() Client {
	var c client
	return &c
}

var CLIENT_DEFAULT_BUFFER_SIZE int = 2048

type client struct {
	Client
	rw NetMessageReaderWriter
	requestId uint32
	errorString string	
	rawjson string
}

func (this *client)	Connect(address string) bool {
	this.Disconnect()	
	conn , err := net.Dial("tcp", address)
	if err != nil {
		this.errorString = err.Error()	
		return false
	}
	this.rw.Set(conn, CLIENT_DEFAULT_BUFFER_SIZE)
	return true
} 

func (this *client) Disconnect() {
	this.write("close")	
	this.reset()
	this.rw.Close()
}

func (this *client) Ok() bool {
	return len(this.errorString) == 0
}

func (this *client) Failed() bool {
	return !this.Ok()
}

func (this *client) ErrorString() string {
	return this.errorString
}

func (this *client) Execute(command string) bool {
	ok := this.write(command)
	var bytes []byte
	var header *NetworkHeader
	if ok {
		header, bytes, ok = this.read()
		if header.RequestId != this.requestId {
			ok = false
			this.errorString = "invalid requestId"
		}
	}
	if ok {
		this.rawjson = string(bytes)
		// decode message
	}	
	return ok
}

func (this *client) JSON() string {
	return this.rawjson
}

func (this *client) reset() {
	this.resetError()
	this.rawjson = ""
}

func (this *client) resetError() {
	this.errorString = ""
}

func (this *client) setError(err error) {
	this.errorString = err.Error()
}

func (this *client) write(message string) bool {
	this.requestId++
	this.resetError()	
	if this.rw.Valid() {
		err := this.rw.WriteHeaderAndMessage(this.requestId, []byte(message)) 	
		if err == nil {
			return true	
		}
		this.setError(err)
		return false
	}
	this.errorString = "Not connected"
	return false
}

func (this *client) read() (*NetworkHeader, []byte, bool) {
	this.reset()
	if this.rw.Valid() {
		header, bytes, err := this.rw.ReadMessage()
		if err == nil {
			return header, bytes, true
		}
		this.setError(err)
		return nil, nil, false
	}
	this.errorString = "Not connected"
	return nil, nil, false
}

