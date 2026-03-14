package mailbox

type Metrics interface {
	JobEnqueued()                     // a job was put into the mailbox
	JobOverwritten()                  // a job was replaced because mailbox was full
	JobDequeued()                     // worker took the job
	SetCurrentJobAge(seconds float64) // age of the job currently in mailbox
}
