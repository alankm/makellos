/*
Package messages implements a powerful and advanced logging & messaging system
for use by other Vorteil modules. It is also a Vorteil service, providing web-
accessible interfaces to access and interact with the messages database.

Usage

The 'Messages' object implements Vorteil's 'Module' interface. Simply create a
new Messages object and register it with Vorteil's Core as per the Core's
general module registration functionality.

Message

A Message is identical to the Log found within Core's Shared package.

	Severity	// enumeration of supported severity levels
	Time 		// epoch time
	Rules 		// unix-style object rules (user group mode)
	Code 		// a string used to identify the type of message
	Message 	// human-readable description of the message
	Args		// an optional list of contextual key-value pairs

Services

Common URL Query Parameters:
	length:
		Dictates the maximum number of messages to return.
		Default value is -1, which means unlimited.
	offset:
		Dictates the number of messages to skip. Useful for pagination.
		Default value is 0.
	start:
		An epoch time integer that dictates the earliest messages to be
		returned. Default value is 0.
	end:
		An epoch time integer that dictates the latest messages to be
		returned. Default value is time.Now().Unix()
	sort:
		A boolean argument that alters the ordering of returned values.
		If 'true', values are sorted primarily on severity level (most
		severe first) and then secondarily on age (newest first).
		Default value is false, which sorts purely by age (newest
		first).

/:
	Accepts http POST requests, allowing users to post messages directly
	into the system. It will fail if it does not receive enough information
	to adequately define a Message, which it looks for in the request's
	headers:
		Severity (string representation)
		Code
		Message
		Args (optional, but must contain an even number of elements)

	The time will be set to the current time, and the rules will be set to
	(owner group mode) = (username gid 0770).

/debug:
	Accepts http GET requests and returns all relevant severity-level (0)
	messages stored within the database. Modifiable with URL query
	parameters for length, offset, start, end, and sort.
/info:
	Accepts http GET requests and returns all relevant severity-level (1)
	messages stored within the database. Modifiable with URL query
	parameters for length, offset, start, end, and sort.
/warning:
	Accepts http GET requests and returns all relevant severity-level (2)
	messages stored within the database. Modifiable with URL query
	parameters for length, offset, start, end, and sort.
/error:
	Accepts http GET requests and returns all relevant severity-level (3)
	messages stored within the database. Modifiable with URL query
	parameters for length, offset, start, end, and sort.
/critical:
	Accepts http GET requests and returns all relevant severity-level (4)
	messages stored within the database. Modifiable with URL query
	parameters for length, offset, start, end, and sort.
/alert:
	Accepts http GET requests and returns all relevant severity-level
	(2,3,4)	messages stored within the database. Modifiable with URL query
	parameters for length, offset, start, end, and sort.
/all:
	Accepts http GET requests and returns all relevant messages stored
	within the database. Modifiable with URL query parameters for length,
	offset, start, end, and sort.
/ws/debug:
	Accepts websocket requests and returns all relevant severity-level (0)
	messages as they are added to the database.
/ws/info:
	Accepts websocket requests and returns all relevant severity-level (1)
	messages as they are added to the database.
/ws/warning:
	Accepts websocket requests and returns all relevant severity-level (2)
	messages as they are added to the database.
/ws/error:
	Accepts websocket requests and returns all relevant severity-level (3)
	messages as they are added to the database.
/ws/critical:
	Accepts websocket requests and returns all relevant severity-level (4)
	messages as they are added to the database.
/ws/alert:
	Accepts websocket requests and returns all relevant severity-level
	(2,3,4) messages as they are added to the database.
/ws/all:
	Accepts websocket requests and returns all relevant messages as they are
	added to the database.
*/
package messages
