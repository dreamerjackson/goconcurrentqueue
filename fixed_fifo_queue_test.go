package goconcurrentqueue

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/suite"
)

const (
	fixedFIFOQueueCapacity = 500
)

type FixedFIFOTestSuite struct {
	suite.Suite
	fifo *FixedFIFO
}

func (suite *FixedFIFOTestSuite) SetupTest() {
	suite.fifo = NewFixedFIFO(fixedFIFOQueueCapacity)
}

// ***************************************************************************************
// ** Run suite
// ***************************************************************************************

func TestFixedFIFOTestSuite(t *testing.T) {
	suite.Run(t, new(FixedFIFOTestSuite))
}

// ***************************************************************************************
// ** Enqueue && GetLen
// ***************************************************************************************

// single enqueue lock verification
func (suite *FixedFIFOTestSuite) TestEnqueueLockSingleGR() {
	suite.NoError(suite.fifo.Enqueue(1), "Unlocked queue allows to enqueue elements")

	suite.fifo.Lock()
	suite.Error(suite.fifo.Enqueue(1), "Locked queue does not allow to enqueue elements")
}

// single enqueue (1 element, 1 goroutine)
func (suite *FixedFIFOTestSuite) TestEnqueueLenSingleGR() {
	suite.fifo.Enqueue(testValue)
	len := suite.fifo.GetLen()
	suite.Equalf(1, len, "Expected number of elements in queue: 1, currently: %v", len)

	suite.fifo.Enqueue(5)
	len = suite.fifo.GetLen()
	suite.Equalf(2, len, "Expected number of elements in queue: 2, currently: %v", len)
}

// single enqueue at full capacity, 1 goroutine
func (suite *FixedFIFOTestSuite) TestEnqueueFullCapacitySingleGR() {
	total := 5
	suite.fifo = NewFixedFIFO(total)

	for i := 0; i < total; i++ {
		suite.NoError(suite.fifo.Enqueue(i), "no error expected when queue is not full")
	}

	suite.Error(suite.fifo.Enqueue(0), "error expected when queue is full")
}

// TestEnqueueLenMultipleGR enqueues elements concurrently
//
// Detailed steps:
//	1 - Enqueue totalGRs concurrently (from totalGRs different GRs)
//	2 - Verifies the len, it should be equal to totalGRs
//	3 - Verifies that all elements from 0 to totalGRs were enqueued
func (suite *FixedFIFOTestSuite) TestEnqueueLenMultipleGR() {
	var (
		totalGRs = 500
		wg       sync.WaitGroup
	)

	// concurrent enqueueing
	// multiple GRs concurrently enqueueing consecutive integers from 0 to (totalGRs - 1)
	for i := 0; i < totalGRs; i++ {
		wg.Add(1)
		go func(value int) {
			defer wg.Done()
			suite.fifo.Enqueue(value)
		}(i)
	}
	wg.Wait()

	// check that there are totalGRs elements enqueued
	totalElements := suite.fifo.GetLen()
	suite.Equalf(totalGRs, totalElements, "Total enqueued elements should be %v, currently: %v", totalGRs, totalElements)

	// checking that the expected elements (1, 2, 3, ... totalGRs-1 ) were enqueued
	var (
		tmpVal                interface{}
		val                   int
		err                   error
		totalElementsVerified int
	)

	// slice to check every element
	allElements := make([]bool, totalGRs)
	for i := 0; i < totalElements; i++ {
		tmpVal, err = suite.fifo.Dequeue()
		suite.NoError(err, "No error should be returned trying to get an existent element")

		val = tmpVal.(int)
		if allElements[val] {
			suite.Failf("Duplicated element", "Unexpected duplicated value: %v", val)
		} else {
			allElements[val] = true
			totalElementsVerified++
		}
	}
	suite.True(totalElementsVerified == totalGRs, "Enqueued elements are missing")
}

// call GetLen concurrently
func (suite *FixedFIFOTestSuite) TestGetLenMultipleGRs() {
	var (
		totalGRs               = 100
		totalElementsToEnqueue = 10
		wg                     sync.WaitGroup
	)

	for i := 0; i < totalElementsToEnqueue; i++ {
		suite.fifo.Enqueue(i)
	}

	for i := 0; i < totalGRs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			total := suite.fifo.GetLen()
			suite.Equalf(totalElementsToEnqueue, total, "Expected len: %v", totalElementsToEnqueue)
		}()
	}
	wg.Wait()
}

// ***************************************************************************************
// ** GetCap
// ***************************************************************************************

// single GR getCapacity
func (suite *FixedFIFOTestSuite) TestGetCapSingleGR() {
	// initial capacity
	suite.Equal(fixedFIFOQueueCapacity, suite.fifo.GetCap(), "unexpected capacity")

	// new fifo with capacity == 10
	suite.fifo = NewFixedFIFO(10)
	suite.Equal(10, suite.fifo.GetCap(), "unexpected capacity")
}

// ***************************************************************************************
// ** Dequeue
// ***************************************************************************************

// single dequeue lock verification
func (suite *FixedFIFOTestSuite) TestDequeueLockSingleGR() {
	suite.fifo.Enqueue(1)
	_, err := suite.fifo.Dequeue()
	suite.NoError(err, "Unlocked queue allows to dequeue elements")

	suite.fifo.Enqueue(1)
	suite.fifo.Lock()
	_, err = suite.fifo.Dequeue()
	suite.Error(err, "Locked queue does not allow to dequeue elements")
}

// dequeue an empty queue
func (suite *FixedFIFOTestSuite) TestDequeueEmptyQueueSingleGR() {
	val, err := suite.fifo.Dequeue()
	suite.Errorf(err, "Can't dequeue an empty queue")
	suite.Equal(nil, val, "Can't get a value different than nil from an empty queue")
}

// dequeue all elements
func (suite *FixedFIFOTestSuite) TestDequeueSingleGR() {
	suite.fifo.Enqueue(testValue)
	suite.fifo.Enqueue(5)

	// dequeue the first element
	val, err := suite.fifo.Dequeue()
	suite.NoError(err, "Unexpected error")
	suite.Equal(testValue, val, "Wrong element's value")
	len := suite.fifo.GetLen()
	suite.Equal(1, len, "Incorrect number of queue elements")

	// get the second element
	val, err = suite.fifo.Dequeue()
	suite.NoError(err, "Unexpected error")
	suite.Equal(5, val, "Wrong element's value")
	len = suite.fifo.GetLen()
	suite.Equal(0, len, "Incorrect number of queue elements")

}

// dequeue an item after closing the empty queue's channel
func (suite *FixedFIFOTestSuite) TestDequeueClosedChannelSingleGR() {
	// enqueue a dummy item
	suite.fifo.Enqueue(1)
	// close the internal queue's channel
	close(suite.fifo.queue)
	// dequeue the dummy item
	suite.fifo.Dequeue()

	// dequeue after the queue's channel was closed
	val, err := suite.fifo.Dequeue()
	suite.Error(err, "error expected after internal queue channel was closed")
	suite.Nil(val, "nil value expected after internal channel was closed")
}

// TestDequeueMultipleGRs dequeues elements concurrently
//
// Detailed steps:
//	1 - Enqueues totalElementsToEnqueue consecutive integers
//	2 - Dequeues totalElementsToDequeue concurrently from totalElementsToDequeue GRs
//	3 - Verifies the final len, should be equal to totalElementsToEnqueue - totalElementsToDequeue
//	4 - Verifies that the next dequeued element's value is equal to totalElementsToDequeue
func (suite *FixedFIFOTestSuite) TestDequeueMultipleGRs() {
	var (
		wg                     sync.WaitGroup
		totalElementsToEnqueue = 100
		totalElementsToDequeue = 90
	)

	for i := 0; i < totalElementsToEnqueue; i++ {
		suite.fifo.Enqueue(i)
	}

	for i := 0; i < totalElementsToDequeue; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := suite.fifo.Dequeue()
			suite.NoError(err, "Unexpected error during concurrent Dequeue()")
		}()
	}
	wg.Wait()

	// check len, should be == totalElementsToEnqueue - totalElementsToDequeue
	totalElementsAfterDequeue := suite.fifo.GetLen()
	suite.Equal(totalElementsToEnqueue-totalElementsToDequeue, totalElementsAfterDequeue, "Total elements on queue (after Dequeue) does not match with expected number")

	// check current first element
	val, err := suite.fifo.Dequeue()
	suite.NoError(err, "No error should be returned when dequeuing an existent element")
	suite.Equalf(totalElementsToDequeue, val, "The expected last element's value should be: %v", totalElementsToEnqueue-totalElementsToDequeue)
}

// ***************************************************************************************
// ** Lock / Unlock / IsLocked
// ***************************************************************************************

// single lock
func (suite *FixedFIFOTestSuite) TestLockSingleGR() {
	suite.fifo.Lock()
	suite.True(suite.fifo.IsLocked(), "fifo.isLocked has to be true after fifo.Lock()")
}

func (suite *FixedFIFOTestSuite) TestMultipleLockSingleGR() {
	for i := 0; i < 5; i++ {
		suite.fifo.Lock()
	}

	suite.True(suite.fifo.IsLocked(), "queue must be locked after Lock() operations")
}

// single unlock
func (suite *FixedFIFOTestSuite) TestUnlockSingleGR() {
	suite.fifo.Lock()
	suite.fifo.Unlock()
	suite.True(suite.fifo.IsLocked() == false, "fifo.isLocked has to be false after fifo.Unlock()")

	suite.fifo.Unlock()
	suite.True(suite.fifo.IsLocked() == false, "fifo.isLocked has to be false after fifo.Unlock()")
}
