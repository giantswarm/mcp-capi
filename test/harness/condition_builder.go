package harness

import (
	corev1 "k8s.io/api/core/v1"
)

// simpleCondition holds a condition configuration for machines, nodes, and machine sets.
type simpleCondition struct {
	condType string
	status   corev1.ConditionStatus
	reason   string
	message  string
}

// simpleConditionBuilder provides a fluent API for configuring conditions.
// P is the parent builder type returned by Done().
type simpleConditionBuilder[P any] struct {
	condType string
	status   corev1.ConditionStatus
	reason   string
	message  string
	done     func(simpleCondition) P
}

// True sets the condition status to True.
func (cb *simpleConditionBuilder[P]) True() *simpleConditionBuilder[P] {
	cb.status = corev1.ConditionTrue
	return cb
}

// False sets the condition status to False.
func (cb *simpleConditionBuilder[P]) False() *simpleConditionBuilder[P] {
	cb.status = corev1.ConditionFalse
	return cb
}

// Unknown sets the condition status to Unknown.
func (cb *simpleConditionBuilder[P]) Unknown() *simpleConditionBuilder[P] {
	cb.status = corev1.ConditionUnknown
	return cb
}

// Reason sets the reason for this condition.
func (cb *simpleConditionBuilder[P]) Reason(reason string) *simpleConditionBuilder[P] {
	cb.reason = reason
	return cb
}

// Message sets the message for this condition.
func (cb *simpleConditionBuilder[P]) Message(message string) *simpleConditionBuilder[P] {
	cb.message = message
	return cb
}

// Done finalizes the condition and returns to the parent builder.
func (cb *simpleConditionBuilder[P]) Done() P {
	return cb.done(simpleCondition{
		condType: cb.condType,
		status:   cb.status,
		reason:   cb.reason,
		message:  cb.message,
	})
}
