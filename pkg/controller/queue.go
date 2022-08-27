package controller

type updateObj struct {
	oldObj interface{}
	newObj interface{}
}

var (
	queueSize = 100

	addQueue    = make(chan interface{}, queueSize)
	updateQueue = make(chan updateObj, queueSize)
	deleteQueue = make(chan interface{}, queueSize)

	addFunc    = func(obj interface{}) { addQueue <- obj }
	updateFunc = func(oldObj interface{}, newObj interface{}) {
		uo := updateObj{}
		uo.oldObj = oldObj
		uo.newObj = newObj
		updateQueue <- uo
	}
	deleteFunc = func(obj interface{}) { deleteQueue <- obj }
)
