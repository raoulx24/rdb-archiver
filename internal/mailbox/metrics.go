package mailbox

type Metrics interface {
	JobEnqueued()
	JobOverwritten()
	JobDequeued()
	SetCurrentJobAge(seconds float64)
}
