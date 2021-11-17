package ext

import (
	"context"
	"github.com/bytepowered/runv"
	"sync"
)

var _ runv.Liveness = new(StateWorker)

type stepTask struct {
	Id         string
	OnStepWork func() error
}

type stateTask struct {
	Id          string
	OnStateWork func(ctx context.Context) error
}

type StateWorker struct {
	statectx   context.Context
	statefun   context.CancelFunc
	stepTasks  []stepTask
	stateTasks []stateTask
	workwg     sync.WaitGroup
}

func NewStateWorker(ctx context.Context) *StateWorker {
	sc, sf := context.WithCancel(ctx)
	return &StateWorker{
		statectx: sc, statefun: sf,
		stepTasks:  make([]stepTask, 0, 2),
		stateTasks: make([]stateTask, 0, 2),
	}
}

func (s *StateWorker) AddStepTask(id string, task func() error) *StateWorker {
	s.stepTasks = append(s.stepTasks, stepTask{Id: id, OnStepWork: task})
	return s
}

func (s *StateWorker) AddStateTask(id string, task func(ctx context.Context) error) *StateWorker {
	s.stateTasks = append(s.stateTasks, stateTask{Id: id, OnStateWork: task})
	return s
}

func (s *StateWorker) Startup(ctx context.Context) error {
	for _, t := range s.stepTasks {
		go s.doStepTask(t)
	}
	for _, t := range s.stateTasks {
		go s.doStateTask(t)
	}
	return nil
}

func (s *StateWorker) Shutdown(ctx context.Context) error {
	s.statefun()
	s.workwg.Wait()
	size := len(s.stepTasks)
	if size > 0 {
		runv.Log().Infof("tasks(%d): shutdown", size)
	}
	return nil
}

func (s *StateWorker) StateContext() context.Context {
	return s.statectx
}

func (s *StateWorker) doStateTask(state stateTask) {
	defer runv.Log().Infof("state-task(%s): terminaled", state.Id)
	runv.Log().Infof("state-task(%s): startup", state.Id)
	if err := state.OnStateWork(s.statectx); err != nil {
		runv.Log().Errorf("state-task(%s): stop by error: %s", state.Id, err)
	}
}

func (s *StateWorker) doStepTask(step stepTask) {
	defer s.workwg.Done()
	s.workwg.Add(1)
	defer runv.Log().Infof("step-task(%s): terminaled", step.Id)
	runv.Log().Infof("step-task(%s): startup", step.Id)
	for {
		select {
		case <-s.statectx.Done():
			runv.Log().Infof("step-task(%s): stop by signal", step.Id)
			return

		default:
			if err := step.OnStepWork(); err != nil {
				runv.Log().Errorf("step-task(%s): stop by error: %s", step.Id, err)
				return
			}
		}
	}
}
