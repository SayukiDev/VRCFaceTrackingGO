package service

import (
	"errors"
	"fmt"

	"VRCFaceTracking/internal/mutation"
)

// MutatorEnabled reports whether mutations are applied at all.
func (s *Service) MutatorEnabled() bool { return s.mutator.Enabled() }

// SetMutatorEnabled toggles the whole mutation pipeline.
func (s *Service) SetMutatorEnabled(v bool) { s.mutator.SetEnabled(v) }

// Mutations returns the ordered mutation list.
func (s *Service) Mutations() []mutation.Mutation { return s.mutator.Mutations() }

// SetMutationActive toggles one mutation and persists it.
func (s *Service) SetMutationActive(name string, active bool) error {
	m, ok := s.mutator.Find(name)
	if !ok {
		return fmt.Errorf("unknown mutation %q", name)
	}
	m.SetActive(active)
	return s.mutator.SaveMutation(m)
}

func (s *Service) findComponent(mutationName, componentName string) (mutation.Mutation, mutation.Component, error) {
	m, ok := s.mutator.Find(mutationName)
	if !ok {
		return nil, mutation.Component{}, fmt.Errorf("unknown mutation %q", mutationName)
	}
	for _, c := range m.Components() {
		if c.Name == componentName {
			return m, c, nil
		}
	}
	return nil, mutation.Component{}, fmt.Errorf("unknown component %q on %q", componentName, mutationName)
}

// SetMutationValue updates a checkbox, slider or range component. value
// must be a bool, a number or a mutation.Range.
func (s *Service) SetMutationValue(mutationName, componentName string, value any) error {
	m, c, err := s.findComponent(mutationName, componentName)
	if err != nil {
		return err
	}
	if c.Set == nil {
		return errors.New("component is not editable")
	}
	if err := c.Set(value); err != nil {
		return err
	}
	if c.SaveImmediately {
		return s.mutator.SaveMutation(m)
	}
	return nil
}

// InvokeMutationButton runs a button component's action asynchronously.
func (s *Service) InvokeMutationButton(mutationName, componentName string) error {
	_, c, err := s.findComponent(mutationName, componentName)
	if err != nil {
		return err
	}
	if c.Invoke == nil {
		return errors.New("component is not a button")
	}
	go c.Invoke()
	return nil
}
